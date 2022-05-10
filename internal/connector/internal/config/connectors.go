package config

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	gherrors "github.com/pkg/errors"
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
	ConnectorEvalDuration               time.Duration           `json:"connector_eval_duration"`
	ConnectorEvalOrganizations          []string                `json:"connector_eval_organizations"`
	ConnectorNamespaceLifecycleAPI      bool                    `json:"connector_namespace_lifecycle_api"`
	ConnectorEnableUnassignedConnectors bool                    `json:"connector_enable_unassigned_connectors"`
	ConnectorCatalogDirs                []string                `json:"connector_types"`
	CatalogEntries                      []ConnectorCatalogEntry `json:"connector_type_urls"`
	CatalogChecksums                    map[string]string       `json:"connector_catalog_checksums"`
}

var _ environments.ConfigModule = &ConnectorsConfig{}

type ConnectorChannelConfig struct {
	Revision      int64                  `json:"revision,omitempty"`
	ShardMetadata map[string]interface{} `json:"shard_metadata,omitempty"`
}

type ConnectorCatalogEntry struct {
	Channels      map[string]ConnectorChannelConfig `json:"channels,omitempty"`
	ConnectorType public.ConnectorType              `json:"connector_type"`
}

func NewConnectorsConfig() *ConnectorsConfig {
	return &ConnectorsConfig{
		CatalogChecksums: make(map[string]string),
	}
}

func (c *ConnectorsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringArrayVar(&c.ConnectorCatalogDirs, "connector-catalog", c.ConnectorCatalogDirs, "Directory containing connector catalog entries")
	fs.DurationVar(&c.ConnectorEvalDuration, "connector-eval-duration", c.ConnectorEvalDuration, "Connector eval duration in golang duration format")
	fs.StringArrayVar(&c.ConnectorEvalOrganizations, "connector-eval-organizations", c.ConnectorEvalOrganizations, "Connector eval organization IDs")
	fs.BoolVar(&c.ConnectorNamespaceLifecycleAPI, "connector-namespace-lifecycle-api", c.ConnectorNamespaceLifecycleAPI, "Enable APIs to create, update, delete non-eval Namespaces")
	fs.BoolVar(&c.ConnectorEnableUnassignedConnectors, "connector-enable-unassigned-connectors", c.ConnectorEnableUnassignedConnectors, "Enable support for 'unassigned' state for Connectors")
}

func (c *ConnectorsConfig) ReadFiles() error {
	typesLoaded := map[string]string{}
	var values []ConnectorCatalogEntry
	for _, dir := range c.ConnectorCatalogDirs {
		dir = shared.BuildFullFilePath(dir)
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			err = gherrors.Errorf("error listing connector catalogs in %s: %s", dir, err)
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
				err = gherrors.Errorf("error reading catalog file %s: %s", file, err)
				return err
			}

			entry := ConnectorCatalogEntry{}
			err = json.Unmarshal(buf, &entry)
			if err != nil {
				err = gherrors.Errorf("error unmarshaling catalog file %s: %s", file, err)
				return err
			}

			// compute checksum for catalog entry to look for updates
			sum, err := checksum(entry)
			if err != nil {
				err = gherrors.Errorf("error computing checksum for catalog file %s: %s", file, err)
				return err
			}
			c.CatalogChecksums[entry.ConnectorType.Id] = sum

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

func checksum(spec interface{}) (string, error) {
	h := sha1.New()
	err := json.NewEncoder(h).Encode(spec)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
