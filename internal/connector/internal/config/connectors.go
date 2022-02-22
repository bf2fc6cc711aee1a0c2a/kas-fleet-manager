package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"time"
)

type ConnectorsConfig struct {
	ConnectorEvalDurationString string                  `json:"connector_eval_duration"`
	ConnectorEvalOrganizations  []string                `json:"connector_eval_organizations"`
	ConnectorCatalogDirs        []string                `json:"connector_types"`
	CatalogEntries              []ConnectorCatalogEntry `json:"connector_type_urls"`
	ConnectorEvalDuration       time.Duration
}

var _ environments.ConfigModule = &ConnectorsConfig{}

var _ environments.ServiceValidator = &ConnectorsConfig{}

type ConnectorChannelConfig struct {
	Revision      int64                  `json:"revision,omitempty"`
	ShardMetadata map[string]interface{} `json:"shard_metadata,omitempty"`
}

type ConnectorCatalogEntry struct {
	Channels      map[string]*ConnectorChannelConfig `json:"channels,omitempty"`
	ConnectorType public.ConnectorType               `json:"connector_type"`
}

func NewConnectorsConfig() *ConnectorsConfig {
	return &ConnectorsConfig{}
}

func (c *ConnectorsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringArrayVar(&c.ConnectorCatalogDirs, "connector-catalog", c.ConnectorCatalogDirs, "Directory containing connector catalog entries")
	fs.StringArrayVar(&c.ConnectorEvalOrganizations, "connector-eval-duration", c.ConnectorCatalogDirs, "Connector eval duration in go duration format")
	fs.StringArrayVar(&c.ConnectorEvalOrganizations, "connector-eval-organizations", c.ConnectorCatalogDirs, "Connector eval organization IDs")
}

func (c *ConnectorsConfig) ReadFiles() error {
	typesLoaded := map[string]string{}
	var values []ConnectorCatalogEntry
	for _, dir := range c.ConnectorCatalogDirs {
		dir = shared.BuildFullFilePath(dir)
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, f := range files {

			// skip over hidden files and dirs..
			if f.IsDir() || strings.HasPrefix(f.Name(), ".") {
				continue
			}

			file := filepath.Join(dir, f.Name())

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
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].ConnectorType.Id < values[j].ConnectorType.Id
	})
	glog.Infof("loaded %d connector types", len(values))
	c.CatalogEntries = values
	return nil
}

func (c *ConnectorsConfig) Validate(env *environments.Env) error {
	// validate duration
	duration, err := time.ParseDuration(c.ConnectorEvalDurationString)
	if err != nil {
		return err
	}
	c.ConnectorEvalDuration = duration
	return nil
}
