package config

import (
	"os"
	"testing"
	"time"

	"github.com/onsi/gomega"
)

func TestConnectorsConfig_ReadFiles(t *testing.T) {
	g := gomega.NewWithT(t)

	tmpCatalog, err := createSymLinkedCatalogDir("connector")
	g.Expect(err).To(gomega.BeNil())

	defer func() {
		_ = os.RemoveAll(tmpCatalog)
	}()

	type fields struct {
		ConnectorEvalDuration               time.Duration
		ConnectorEvalOrganizations          []string
		ConnectorNamespaceLifecycleAPI      bool
		ConnectorEnableUnassignedConnectors bool
		ConnectorCatalogDirs                []string
		ConnectorMetadataDirs               []string
		CatalogEntries                      []ConnectorCatalogEntry
		CatalogChecksums                    map[string]string
	}

	connectorMetadataGoodDirs := []string{"./internal/connector/test/integration/resources/connector-metadata"}
	connectorCatalogGoodDirs := []string{"./internal/connector/test/integration/resources/connector-catalog"}
	connectorCatalogRootDirs := []string{"./internal/connector/test/integration/resources/connector-catalog-root"}
	connectorCatalogBadDirs := []string{"./internal/connector/test/integration/resources/bad-connector-catalog"}
	connectorMetadataMissingDirs := []string{"./internal/connector/test/integration/resources/missing-connector-metadata"}
	connectorMetadataBadDirs := []string{"./internal/connector/test/integration/resources/bad-connector-metadata"}
	connectorMetadataUnknownDirs := []string{"./internal/connector/test/integration/resources/unknown-connector-metadata"}

	tests := []struct {
		name          string
		fields        fields
		wantErr       bool
		err           string
		connectorsIDs []string
	}{
		{
			name: "valid catalog",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ConnectorMetadataDirs: connectorMetadataGoodDirs,
				ConnectorCatalogDirs:  connectorCatalogGoodDirs},
			wantErr:       false,
			connectorsIDs: []string{"log_sink_0.1", "aws-sqs-source-v1alpha1", "postgresql_sink_0.1"},
		},
		{
			name: "valid catalog walk with symlink",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ConnectorMetadataDirs: connectorMetadataGoodDirs,
				ConnectorCatalogDirs:  []string{tmpCatalog}},
			wantErr:       false,
			connectorsIDs: []string{"log_sink_0.1", "aws-sqs-source-v1alpha1", "postgresql_sink_0.1"},
		},
		{
			name: "valid catalog walk",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ConnectorMetadataDirs: connectorMetadataGoodDirs,
				ConnectorCatalogDirs:  connectorCatalogRootDirs},
			wantErr:       false,
			connectorsIDs: []string{"log_sink_0.1", "aws-sqs-source-v1alpha1", "postgresql_sink_0.1"},
		},
		{
			name: "bad catalog directory",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ConnectorMetadataDirs: connectorMetadataGoodDirs,
				ConnectorCatalogDirs:  []string{"./bad-catalog-directory"}},
			wantErr: true,
			err:     "^error listing connector catalogs in .+/bad-catalog-directory: lstat .+/bad-catalog-directory: no such file or directory$",
		},
		{
			name: "bad catalog file",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ConnectorMetadataDirs: connectorMetadataGoodDirs,
				ConnectorCatalogDirs:  connectorCatalogBadDirs},
			wantErr: true,
			err:     ".*error unmarshaling catalog file .+/internal/connector/test/integration/resources/bad-connector-catalog/bad-connector-type.json: invalid character 'b' looking for beginning of value$",
		},
		{
			name: "missing metadata",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ConnectorMetadataDirs: connectorMetadataMissingDirs,
				ConnectorCatalogDirs:  connectorCatalogGoodDirs},
			wantErr: true,
			err:     "^error listing connector catalogs in .+/internal/connector/test/integration/resources/connector-catalog: missing metadata for connector aws-sqs-source-v1alpha1$",
		},
		{
			name: "bad metadata directory",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ConnectorMetadataDirs: []string{"./bad-metadata-directory"},
				ConnectorCatalogDirs:  []string{"./bad-catalog-directory"}},
			wantErr: true,
			err:     "^error listing connector metadata in .+/bad-metadata-directory: lstat .+/bad-metadata-directory: no such file or directory$",
		},
		{
			name: "bad metadata file",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ConnectorMetadataDirs: connectorMetadataBadDirs,
				ConnectorCatalogDirs:  connectorCatalogBadDirs},
			wantErr: true,
			err:     "^error listing connector metadata in .+/internal/connector/test/integration/resources/bad-connector-metadata: error reading connector metadata from .+/internal/connector/test/integration/resources/bad-connector-metadata/bad-connector-metadata.yaml: yaml: unmarshal errors:\\n\\s*line 1: cannot unmarshal !!str `bad-con...` into \\[\\]config.ConnectorMetadata$",
		},
		{
			name: "unknown metadata",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ConnectorMetadataDirs: connectorMetadataUnknownDirs,
				ConnectorCatalogDirs:  connectorCatalogGoodDirs},
			wantErr:       true,
			err:           "^found 1 unrecognized connector metadata with ids: \\[unknown\\]$",
			connectorsIDs: []string{"log_sink_0.1", "aws-sqs-source-v1alpha1", "postgresql_sink_0.1"},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ConnectorsConfig{
				ConnectorEvalDuration:               tt.fields.ConnectorEvalDuration,
				ConnectorEvalOrganizations:          tt.fields.ConnectorEvalOrganizations,
				ConnectorNamespaceLifecycleAPI:      tt.fields.ConnectorNamespaceLifecycleAPI,
				ConnectorEnableUnassignedConnectors: tt.fields.ConnectorEnableUnassignedConnectors,
				ConnectorMetadataDirs:               tt.fields.ConnectorMetadataDirs,
				ConnectorCatalogDirs:                tt.fields.ConnectorCatalogDirs,
				CatalogEntries:                      tt.fields.CatalogEntries,
				CatalogChecksums:                    tt.fields.CatalogChecksums,
			}
			if err := c.ReadFiles(); (err != nil) != tt.wantErr {
				t.Errorf("ReadFiles() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr {
				g.Expect(err.Error()).To(gomega.MatchRegexp(tt.err))
			}

			g.Expect(c.CatalogEntries).To(gomega.HaveLen(len(tt.connectorsIDs)))

			for _, connectorID := range tt.connectorsIDs {
				connectorID := connectorID
				g.Expect(c.CatalogEntries).To(gomega.Satisfy(func(entries []ConnectorCatalogEntry) bool {
					for i := range entries {
						if entries[i].ConnectorType.Id == connectorID {
							return true
						}
					}
					return false
				}))
			}
		})
	}

}
