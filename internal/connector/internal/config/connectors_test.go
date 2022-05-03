package config

import (
	gomega "github.com/onsi/gomega"
	"testing"
	"time"
)

func TestConnectorsConfig_ReadFiles(t *testing.T) {

	gomega.RegisterTestingT(t)

	type fields struct {
		ConnectorEvalDuration               time.Duration
		ConnectorEvalOrganizations          []string
		ConnectorNamespaceLifecycleAPI      bool
		ConnectorEnableUnassignedConnectors bool
		ConnectorCatalogDirs                []string
		CatalogEntries                      []ConnectorCatalogEntry
		CatalogChecksums                    map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		err     string
	}{
		{
			name: "valid catalog",
			fields: fields{
				CatalogChecksums:     make(map[string]string),
				ConnectorCatalogDirs: []string{"./internal/connector/test/integration/connector-catalog"}},
			wantErr: false,
		},
		{
			name: "bad catalog directory",
			fields: fields{
				CatalogChecksums:     make(map[string]string),
				ConnectorCatalogDirs: []string{"./bad-catalog-directory"}},
			wantErr: true,
			err:     "^error listing connector catalogs in .+/bad-catalog-directory: open .+/bad-catalog-directory: no such file or directory$",
		},
		{
			name: "bad catalog file",
			fields: fields{
				CatalogChecksums:     make(map[string]string),
				ConnectorCatalogDirs: []string{"./internal/connector/test/bad-connector-catalog"}},
			wantErr: true,
			err:     "^error unmarshaling catalog file .+/internal/connector/test/bad-connector-catalog/bad-connector-type.json: invalid character 'b' looking for beginning of value$",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ConnectorsConfig{
				ConnectorEvalDuration:               tt.fields.ConnectorEvalDuration,
				ConnectorEvalOrganizations:          tt.fields.ConnectorEvalOrganizations,
				ConnectorNamespaceLifecycleAPI:      tt.fields.ConnectorNamespaceLifecycleAPI,
				ConnectorEnableUnassignedConnectors: tt.fields.ConnectorEnableUnassignedConnectors,
				ConnectorCatalogDirs:                tt.fields.ConnectorCatalogDirs,
				CatalogEntries:                      tt.fields.CatalogEntries,
				CatalogChecksums:                    tt.fields.CatalogChecksums,
			}
			if err := c.ReadFiles(); (err != nil) != tt.wantErr {
				t.Errorf("ReadFiles() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr {
				gomega.Expect(err.Error()).To(gomega.MatchRegexp(tt.err))
			}
		})
	}
}
