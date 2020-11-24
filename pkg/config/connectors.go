package config

import (
	"github.com/bep/debounce"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type ConnectorsConfig struct {
	Enabled              bool     `json:"enabled"`
	ConnectorTypesDir    string   `json:"connector_types"`
	ConnectorTypeSvcUrls []string `json:"connector_type_urls"`
}

func NewConnectorsConfig() *ConnectorsConfig {
	return &ConnectorsConfig{
		Enabled:           false,
		ConnectorTypesDir: "config/connector-types",
	}
}

func (c *ConnectorsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Enabled, "enable-connectors", c.Enabled, "Enable connectors")
	fs.StringVar(&c.ConnectorTypesDir, "connector-types", c.ConnectorTypesDir, "Directory containing connector types service URLs")
}

func (c *ConnectorsConfig) ReadFiles() error {
	files, err := ioutil.ReadDir(c.ConnectorTypesDir)
	if err != nil {
		return err
	}

	glog.Info("loading connector types")
	var values []string
	for _, f := range files {
		if f.IsDir() || strings.HasPrefix(f.Name(), ".") {
			continue
		}
		value := ""
		err := readFileValueString(filepath.Join(c.ConnectorTypesDir, f.Name()), &value)
		if err != nil {
			return err
		}
		values = append(values, value)
	}
	sort.Strings(values)
	c.ConnectorTypeSvcUrls = values
	return nil
}

func stringArrayEquals(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func (c *ConnectorsConfig) WatchFiles(onChange func(newVersion ConnectorsConfig)) (Close func(), err error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Lets debounce to avoid generating too many change events when lots
	// of files are being updated in short time.
	debounced := debounce.New(100 * time.Millisecond)
	mu := sync.Mutex{}
	lastConnectorTypeSvcUrls := c.ConnectorTypeSvcUrls

	checkForNewVersion := func() {
		// We emit update copies of the config to avoid race conditions
		newVersion := *c
		err := newVersion.ReadFiles()
		if err != nil {
			glog.Warningf("failed reading connectors config files: %v", err)
			return
		}

		// to be safe, make sure we lock the go routine while processing the even (to avoid concurrent updates).
		mu.Lock()
		defer mu.Unlock()

		// only fire the change event if the URLs actually changed. (ignores touching files etc.)
		if !stringArrayEquals(lastConnectorTypeSvcUrls, newVersion.ConnectorTypeSvcUrls) {
			lastConnectorTypeSvcUrls = newVersion.ConnectorTypeSvcUrls
			onChange(newVersion)
		}
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return
				}
				debounced(checkForNewVersion)
			case <-watcher.Errors:
				return
			}
		}
	}()

	err = watcher.Add(c.ConnectorTypesDir)
	if err != nil {
		_ = watcher.Close()
		return nil, err
	}

	return func() {
		_ = watcher.Close()
	}, nil
}
