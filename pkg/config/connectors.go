package config

import (
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/connector/openapi"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ConnectorsConfig struct {
	Enabled              bool                    `json:"enabled"`
	ConnectorCatalogDirs []string                `json:"connector_types"`
	CatalogEntries       []ConnectorCatalogEntry `json:"connector_type_urls"`
}

type ConnectorChannelConfig struct {
	Revision      int64                  `json:"revision,omitempty"`
	ShardMetadata map[string]interface{} `json:"shard_metadata,omitempty"`
}

type ConnectorCatalogEntry struct {
	Channels      map[string]*ConnectorChannelConfig `json:"channels,omitempty"`
	ConnectorType openapi.ConnectorType              `json:"connector_type"`
}

func NewConnectorsConfig() *ConnectorsConfig {
	return &ConnectorsConfig{
		Enabled:              false,
		ConnectorCatalogDirs: []string{"config/connector-types"},
	}
}

func (c *ConnectorsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Enabled, "enable-connectors", c.Enabled, "Enable connectors")
	fs.StringArrayVar(&c.ConnectorCatalogDirs, "connector-catalog", c.ConnectorCatalogDirs, "Directory containing connector catalog entries")
}

func (c *ConnectorsConfig) ReadFiles() error {
	typesLoaded := map[string]string{}
	var values []ConnectorCatalogEntry
	for _, dir := range c.ConnectorCatalogDirs {
		dir = BuildFullFilePath(dir)

		err := c.walkCatalogDirs(dir, dir, func(path string, info os.FileInfo, err error) error {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			if info.IsDir() {
				return nil
			}

			file := path

			// Read the file
			buf, err := ioutil.ReadFile(file)
			if err != nil {
				return err
			}

			entry := ConnectorCatalogEntry{}
			err = json.Unmarshal(buf, &entry)
			if err != nil {
				return err
			}

			if prev, found := typesLoaded[entry.ConnectorType.Id]; found {
				return fmt.Errorf("connector type '%s' defined in '%s' and '%s'", entry.ConnectorType.Id, file, prev)
			}
			typesLoaded[entry.ConnectorType.Id] = file
			values = append(values, entry)

			return nil
		})

		if err != nil {
			return err
		}
	}

	sort.Slice(values, func(i, j int) bool {
		return values[i].ConnectorType.Id < values[j].ConnectorType.Id
	})
	glog.Infof("loaded %d connector types", len(values))
	c.CatalogEntries = values
	return nil
}

func (c *ConnectorsConfig) walkCatalogDirs(filename string, linkDirname string, walkFn filepath.WalkFunc) error {
	symWalkFunc := func(path string, info os.FileInfo, err error) error {

		if fname, err := filepath.Rel(filename, path); err == nil {
			path = filepath.Join(linkDirname, fname)
		} else {
			return err
		}

		if err == nil && info.Mode()&os.ModeSymlink == os.ModeSymlink {
			finalPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			info, err := os.Lstat(finalPath)
			if err != nil {
				return walkFn(path, info, err)
			}
			if info.IsDir() {
				return c.walkCatalogDirs(finalPath, path, walkFn)
			}
		}

		return walkFn(path, info, err)
	}

	return filepath.Walk(filename, symWalkFunc)
}

