package config

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/rs/xid"

	"github.com/onsi/gomega"
)

func TestConnectorsConfig_ReadFiles(t *testing.T) {
	g := gomega.NewWithT(t)

	tmpCatalog, err := createSymLinkedCatalogDir()
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
		CatalogEntries                      []ConnectorCatalogEntry
		CatalogChecksums                    map[string]string
	}
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
				CatalogChecksums:     make(map[string]string),
				ConnectorCatalogDirs: []string{"./internal/connector/test/integration/connector-catalog"}},
			wantErr:       false,
			connectorsIDs: []string{"log_sink_0.1", "aws-sqs-source-v1alpha1"},
		},
		{
			name: "valid catalog walk with symlink",
			fields: fields{
				CatalogChecksums:     make(map[string]string),
				ConnectorCatalogDirs: []string{tmpCatalog}},
			wantErr:       false,
			connectorsIDs: []string{"log_sink_0.1", "aws-sqs-source-v1alpha1"},
		},
		{
			name: "valid catalog walk",
			fields: fields{
				CatalogChecksums:     make(map[string]string),
				ConnectorCatalogDirs: []string{"./internal/connector/test/integration/connector-catalog-root"}},
			wantErr:       false,
			connectorsIDs: []string{"log_sink_0.1", "aws-sqs-source-v1alpha1"},
		},
		{
			name: "bad catalog directory",
			fields: fields{
				CatalogChecksums:     make(map[string]string),
				ConnectorCatalogDirs: []string{"./bad-catalog-directory"}},
			wantErr: true,
			err:     "^error listing connector catalogs in .+/bad-catalog-directory: lstat .+/bad-catalog-directory: no such file or directory$",
		},
		{
			name: "bad catalog file",
			fields: fields{
				CatalogChecksums:     make(map[string]string),
				ConnectorCatalogDirs: []string{"./internal/connector/test/bad-connector-catalog"}},
			wantErr: true,
			err:     ".*error unmarshaling catalog file .+/internal/connector/test/bad-connector-catalog/bad-connector-type.json: invalid character 'b' looking for beginning of value$",
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
				g.Expect(c.CatalogEntries).To(gomega.Satisfy(func(entries []ConnectorCatalogEntry) bool {
					for i := range entries {
						//nolint:scopelint
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

// this function re-creates what kubernetes does when mounting a volume from a
// configmap where the actual files are double-symlinked from some random named
// path:
//
//   drwxrwxrwx. 3 root root 88 Jul  7 13:18 .
//   drwxr-xr-x. 3 root root 53 Jun  8 12:12 ..
//   drwxr-xr-x. 2 root root 35 Jul  7 13:18 ..2020_07_07_13_18_32.149716995
//   lrwxrwxrwx. 1 root root 31 Jul  7 13:18 ..data -> ..2020_07_07_13_18_32.149716995
//   lrwxrwxrwx. 1 root root 28 Jul  7 13:18 aws-sqs-source-v1alpha1.json -> ..data/aws-sqs-source-v1alpha1.json
func createSymLinkedCatalogDir() (string, error) {
	dir, err := ioutil.TempDir("", "connector-catalog-")
	if err != nil {
		return "", err
	}

	contentDir := path.Join(dir, xid.New().String())
	if err := os.Mkdir(contentDir, 0755); err != nil {
		return "", err
	}
	catalogDir := path.Join(dir, "catalog")
	if err := os.Mkdir(catalogDir, 0755); err != nil {
		return "", err
	}

	dataDir := path.Join(dir, "data")
	if err := os.Symlink(contentDir, dataDir); err != nil {
		return "", err
	}

	source := "./internal/connector/test/integration/connector-catalog"
	source = shared.BuildFullFilePath(source)

	files, err := ioutil.ReadDir(source)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		src := path.Join(source, file.Name())
		dst := path.Join(contentDir, file.Name())

		data, err := ioutil.ReadFile(src)
		if err != nil {
			return "", err
		}
		err = ioutil.WriteFile(dst, data, 0644)
		if err != nil {
			return "", err
		}

		err = os.Symlink(path.Join(dataDir, file.Name()), path.Join(catalogDir, file.Name()))
		if err != nil {
			return "", err
		}
	}

	return dir, nil
}
