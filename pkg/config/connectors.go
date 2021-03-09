package config

import (
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
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
	if c.ConnectorTypesDir == "" {
		return nil
	}

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
