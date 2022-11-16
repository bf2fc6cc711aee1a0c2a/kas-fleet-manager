package config

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/files"

	gherrors "github.com/pkg/errors"

	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

type ConnectorsConfig struct {
	ConnectorEvalDuration               time.Duration           `json:"connector_eval_duration"`
	ConnectorEvalOrganizations          []string                `json:"connector_eval_organizations"`
	ConnectorNamespaceLifecycleAPI      bool                    `json:"connector_namespace_lifecycle_api"`
	ConnectorEnableUnassignedConnectors bool                    `json:"connector_enable_unassigned_connectors"`
	ConnectorCatalogDirs                []string                `json:"connector_types"`
	ConnectorMetadataDirs               []string                `json:"connector_metadata"`
	CatalogEntries                      []ConnectorCatalogEntry `json:"connector_type_urls"`
	CatalogChecksums                    map[string]string       `json:"connector_catalog_checksums"`
}

var _ environments.ConfigModule = &ConnectorsConfig{}

type ConnectorChannelConfig struct {
	ShardMetadata map[string]interface{} `json:"shard_metadata,omitempty"`
}

type ConnectorCatalogEntry struct {
	Channels      map[string]ConnectorChannelConfig `json:"channels,omitempty"`
	ConnectorType public.ConnectorType              `json:"connector_type"`
}

type ConnectorMetadata struct {
	ConnectorTypeId string   `json:"id" yaml:"id"`
	FeaturedRank    int32    `json:"featured-rank" yaml:"featured-rank"`
	Labels          []string `json:"labels" yaml:"labels"`
}

func NewConnectorsConfig() *ConnectorsConfig {
	return &ConnectorsConfig{
		CatalogChecksums: make(map[string]string),
	}
}

func (c *ConnectorsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringArrayVar(&c.ConnectorCatalogDirs, "connector-catalog", c.ConnectorCatalogDirs, "Directory containing connector catalog entries")
	fs.StringArrayVar(&c.ConnectorMetadataDirs, "connector-metadata", c.ConnectorMetadataDirs, "Directory containing connector metadata configuration files")
	fs.DurationVar(&c.ConnectorEvalDuration, "connector-eval-duration", c.ConnectorEvalDuration, "Connector eval duration in golang duration format")
	fs.StringArrayVar(&c.ConnectorEvalOrganizations, "connector-eval-organizations", c.ConnectorEvalOrganizations, "Connector eval organization IDs")
	fs.BoolVar(&c.ConnectorNamespaceLifecycleAPI, "connector-namespace-lifecycle-api", c.ConnectorNamespaceLifecycleAPI, "Enable APIs to create, update, delete non-eval Namespaces")
	fs.BoolVar(&c.ConnectorEnableUnassignedConnectors, "connector-enable-unassigned-connectors", c.ConnectorEnableUnassignedConnectors, "Enable support for 'unassigned' state for Connectors")
}

func (c *ConnectorsConfig) ReadFiles() error {
	// read metadata first to merge with catalog next
	connectorMetadata, err := c.readConnectorMetadata()
	if err != nil {
		return err
	}

	c.CatalogEntries, err = c.readConnectorCatalog(connectorMetadata)
	if err != nil {
		return err
	}

	remainingIds := len(connectorMetadata)
	if remainingIds > 0 {
		ids := make([]string, 0, remainingIds)
		for id := range connectorMetadata {
			ids = append(ids, id)
		}
		return gherrors.Errorf("Found %d unrecognized connector metadata with ids: %s", remainingIds, ids)
	}

	glog.Infof("loaded %d connector types", len(c.CatalogEntries))

	return nil
}

func (c *ConnectorsConfig) readConnectorMetadata() (connectorMetadata map[string]ConnectorMetadata, err error) {
	connectorMetadata = make(map[string]ConnectorMetadata)
	for _, dir := range c.ConnectorMetadataDirs {
		metaDir := shared.BuildFullFilePath(dir)
		err = files.Walk(metaDir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// skip over hidden files and dirs..
			if info.IsDir() || strings.HasPrefix(path, ".") {
				return nil
			}

			glog.Infof("loading connector metadata from file %s", path)

			// Read the file
			var metas []ConnectorMetadata
			err = shared.ReadYamlFile(path, &metas)
			if err != nil {
				return gherrors.Errorf("error reading connector metadata from %s: %s", path, err)
			}
			for _, m := range metas {
				connectorMetadata[m.ConnectorTypeId] = m
			}

			return nil
		})

		if err != nil {
			return nil, gherrors.Errorf("error listing connector metadata in %s: %s", metaDir, err)
		}
	}

	return
}

func (c *ConnectorsConfig) readConnectorCatalog(connectorMetadata map[string]ConnectorMetadata) (values []ConnectorCatalogEntry, err error) {
	typesLoaded := map[string]string{}
	for _, dir := range c.ConnectorCatalogDirs {
		dir = shared.BuildFullFilePath(dir)

		err = files.Walk(dir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// skip over hidden files and dirs..
			if info.IsDir() || strings.HasPrefix(path, ".") {
				return nil
			}

			glog.Infof("loading connectors from file %s", path)

			// Read the file
			buf, err := os.ReadFile(path)
			if err != nil {
				return gherrors.Errorf("error reading catalog file %s: %s", path, err)
			}

			entry := ConnectorCatalogEntry{}
			err = json.Unmarshal(buf, &entry)
			if err != nil {
				return gherrors.Errorf("error unmarshaling catalog file %s: %s", path, err)
			}

			// set catalog metadata from metadata config read earlier
			id := entry.ConnectorType.Id
			if meta, found := connectorMetadata[id]; found {
				entry.ConnectorType.FeaturedRank = meta.FeaturedRank
				entry.ConnectorType.Labels = meta.Labels
				// TODO annotations
			} else {
				return gherrors.Errorf("Missing metadata for connector %s", id)
			}

			// compute checksum for catalog entry to look for updates
			sum, err := checksum(entry)
			if err != nil {
				return gherrors.Errorf("error computing checksum for catalog file %s: %s", path, err)
			}

			// when walking directories with symlink such as what kubernetes does when mounting
			// a volume from a configmap where the actual files are double-symlinked from some
			// random named path so this method is invoked twice or more, but it is not actually
			// always possible to reliably determine if files have been already processed, so we
			// first check if:
			//
			// - the previous file and the new file are the same (os.SameFile)
			// - the file has already been processed
			// - the previous file checksum is the same
			//
			// if any of the above condition is true, we assume the previous file and the current
			// one are actually the same, so we can safely ignore it

			if prev, found := typesLoaded[id]; found {
				prevInfo, prevErr := os.Lstat(prev)
				if prevErr != nil {
					return nil
				}

				if os.SameFile(info, prevInfo) {
					return nil
				}
				if typesLoaded[id] == path {
					return nil
				}
				if c.CatalogChecksums[id] == sum {
					return nil
				}

				return fmt.Errorf("connector type '%s' defined in '%s' and '%s'", id, path, prev)
			}

			c.CatalogChecksums[id] = sum
			typesLoaded[id] = path

			values = append(values, entry)

			glog.Infof("loaded connector %s from file %s", id, path)

			return nil
		})

		if err != nil {
			err = gherrors.Errorf("error listing connector catalogs in %s: %s", dir, err)
			return nil, err
		}
	}

	// remove all processed metadata entries
	for _, entry := range values {
		delete(connectorMetadata, entry.ConnectorType.Id)
	}

	sort.Slice(values, func(i, j int) bool {
		return values[i].ConnectorType.Id < values[j].ConnectorType.Id
	})

	return values, nil
}

func checksum(spec interface{}) (string, error) {
	h := sha1.New()
	err := json.NewEncoder(h).Encode(spec)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
