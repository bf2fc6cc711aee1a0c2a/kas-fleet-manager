package config

import (
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/files"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"io/fs"
	"os"
	"sort"
	"strings"
)

type ProcessorsConfig struct {
	ProcessorsEnabled           bool                    `json:"processors_enabled"`
	ProcessorCatalogDirs        []string                `json:"processor_types"`
	ProcessorMetadataDirs       []string                `json:"processor_metadata"`
	CatalogEntries              []ProcessorCatalogEntry `json:"processor_type_urls"`
	CatalogChecksums            map[string]string       `json:"processor_catalog_checksums"`
	ProcessorsSupportedChannels []string                `json:"processors_supported_channels"`
}

type ProcessorChannelConfig struct {
	ShardMetadata map[string]interface{} `json:"shard_metadata,omitempty"`
}

type ProcessorCatalogEntry struct {
	Channels      map[string]ProcessorChannelConfig `json:"channels,omitempty"`
	ProcessorType public.ProcessorType              `json:"processor_type"`
}

type ProcessorMetadata struct {
	ProcessorTypeId string            `json:"id" yaml:"id"`
	FeaturedRank    int32             `json:"featured-rank" yaml:"featured-rank"`
	Labels          []string          `json:"labels" yaml:"labels"`
	Annotations     map[string]string `json:"annotations" yaml:"annotations"`
}

var _ environments.ConfigModule = &ProcessorsConfig{}

func NewProcessorsConfig() *ProcessorsConfig {
	return &ProcessorsConfig{
		ProcessorsEnabled: false,
		CatalogChecksums:  make(map[string]string),
	}
}

func (c *ProcessorsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.ProcessorsEnabled, "processors-enabled", c.ProcessorsEnabled, "Enable support for (Smart Event) Processors")
	fs.StringArrayVar(&c.ProcessorCatalogDirs, "processor-catalog", c.ProcessorCatalogDirs, "Directory containing processor catalog entries")
	fs.StringArrayVar(&c.ProcessorMetadataDirs, "processor-metadata", c.ProcessorMetadataDirs, "Directory containing processor metadata configuration files")
	fs.StringSliceVar(&c.ProcessorsSupportedChannels, "processors-supported-channels", c.ProcessorsSupportedChannels, "Processor channels that are visible")
}

func (c *ProcessorsConfig) ReadFiles() error {
	// read metadata first to merge with catalog next
	processorMetadata, err := c.readProcessorMetadata()
	if err != nil {
		return err
	}

	// read catalogs and merge metadata, removing entries from the map
	err = c.readProcessorCatalog(processorMetadata)
	if err != nil {
		return err
	}

	// check if there are any unused metadata entries left
	remainingIds := len(processorMetadata)
	if remainingIds > 0 {
		ids := make([]string, 0, remainingIds)
		for id := range processorMetadata {
			ids = append(ids, id)
		}
		return fmt.Errorf("found %d unrecognized processor metadata with ids: %s", remainingIds, ids)
	}

	glog.Infof("loaded %d processor types", len(c.CatalogEntries))

	return nil
}

func (c *ProcessorsConfig) readProcessorMetadata() (processorMetadata map[string]ProcessorMetadata, err error) {
	processorMetadata = make(map[string]ProcessorMetadata)
	for _, dir := range c.ProcessorMetadataDirs {
		metaDir := shared.BuildFullFilePath(dir)
		err = files.Walk(metaDir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// skip over hidden files and dirs..
			if info.IsDir() || strings.HasPrefix(path, ".") {
				return nil
			}

			glog.Infof("Loading Processor metadata from file %s", path)

			// Read the file
			var metas []ProcessorMetadata
			err = shared.ReadYamlFile(path, &metas)
			if err != nil {
				return fmt.Errorf("error reading Processor metadata from %s: %s", path, err)
			}
			for _, m := range metas {
				processorMetadata[m.ProcessorTypeId] = m
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("error listing Processor metadata in %s: %s", metaDir, err)
		}
	}

	return
}

func (c *ProcessorsConfig) readProcessorCatalog(processorMetadata map[string]ProcessorMetadata) (err error) {
	typesLoaded := map[string]string{}
	for _, dir := range c.ProcessorCatalogDirs {
		dir = shared.BuildFullFilePath(dir)

		err = files.Walk(dir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// skip over hidden files and dirs..
			if info.IsDir() || strings.HasPrefix(path, ".") {
				return nil
			}

			glog.Infof("Loading Processors from file %s", path)

			// Read the file
			buf, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("error reading catalog file %s: %s", path, err)
			}

			entry := ProcessorCatalogEntry{}
			err = json.Unmarshal(buf, &entry)
			if err != nil {
				return fmt.Errorf("error unmarshaling catalog file %s: %s", path, err)
			}

			// set catalog metadata from metadata config read earlier
			id := entry.ProcessorType.Id
			if meta, found := processorMetadata[id]; found {
				entry.ProcessorType.FeaturedRank = meta.FeaturedRank
				entry.ProcessorType.Labels = meta.Labels
				entry.ProcessorType.Annotations = meta.Annotations
			} else {
				return fmt.Errorf("missing metadata for Procesor %s", id)
			}

			// compute checksum for catalog entry to look for updates
			sum, err := checksum(entry)
			if err != nil {
				return fmt.Errorf("error computing checksum for catalog file %s: %s", path, err)
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

			c.CatalogEntries = append(c.CatalogEntries, entry)

			glog.Infof("loaded connector %s from file %s", id, path)

			return nil
		})

		if err != nil {
			return fmt.Errorf("error listing connector catalogs in %s: %s", dir, err)
		}
	}

	// remove all processed metadata entries
	for _, entry := range c.CatalogEntries {
		delete(processorMetadata, entry.ProcessorType.Id)
	}

	sort.Slice(c.CatalogEntries, func(i, j int) bool {
		return c.CatalogEntries[i].ProcessorType.Id < c.CatalogEntries[j].ProcessorType.Id
	})

	return nil
}
