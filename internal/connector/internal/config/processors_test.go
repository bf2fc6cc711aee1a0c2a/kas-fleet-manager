package config

import (
	"github.com/onsi/gomega"
	"os"
	"testing"
)

func TestProcessorsConfig_ReadFiles(t *testing.T) {
	g := gomega.NewWithT(t)

	tmpCatalog, err := createSymLinkedCatalogDir("processor")
	g.Expect(err).To(gomega.BeNil())

	defer func() {
		_ = os.RemoveAll(tmpCatalog)
	}()

	type fields struct {
		ProcessorCatalogDirs  []string
		ProcessorMetadataDirs []string
		CatalogEntries        []ProcessorCatalogEntry
		CatalogChecksums      map[string]string
	}

	processorMetadataGoodDirs := []string{"./internal/connector/test/integration/resources/processor-metadata"}
	processorCatalogGoodDirs := []string{"./internal/connector/test/integration/resources/processor-catalog"}
	//processorCatalogRootDirs := []string{"./internal/processor/test/integration/resources/processor-catalog-root"}
	//processorCatalogBadDirs := []string{"./internal/processor/test/integration/resources/bad-processor-catalog"}
	//processorMetadataMissingDirs := []string{"./internal/processor/test/integration/resources/missing-processor-metadata"}
	//processorMetadataBadDirs := []string{"./internal/processor/test/integration/resources/bad-processor-metadata"}
	//processorMetadataUnknownDirs := []string{"./internal/processor/test/integration/resources/unknown-processor-metadata"}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		err     string
	}{
		{
			name: "valid catalog",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ProcessorMetadataDirs: processorMetadataGoodDirs,
				ProcessorCatalogDirs:  processorCatalogGoodDirs},
			wantErr: false,
		},
		{
			name: "valid catalog walk with symlink",
			fields: fields{
				CatalogChecksums:      make(map[string]string),
				ProcessorMetadataDirs: processorMetadataGoodDirs,
				ProcessorCatalogDirs:  []string{tmpCatalog}},
			wantErr: false,
		},
		//{
		//	name: "valid catalog walk",
		//	fields: fields{
		//		CatalogChecksums:      make(map[string]string),
		//		ProcessorMetadataDirs: processorMetadataGoodDirs,
		//		ProcessorCatalogDirs:  processorCatalogRootDirs},
		//	wantErr: false,
		//},
		//{
		//	name: "bad catalog directory",
		//	fields: fields{
		//		CatalogChecksums:      make(map[string]string),
		//		ProcessorMetadataDirs: processorMetadataGoodDirs,
		//		ProcessorCatalogDirs:  []string{"./bad-catalog-directory"}},
		//	wantErr: true,
		//	err:     "^error listing processor catalogs in .+/bad-catalog-directory: lstat .+/bad-catalog-directory: no such file or directory$",
		//},
		//{
		//	name: "bad catalog file",
		//	fields: fields{
		//		CatalogChecksums:      make(map[string]string),
		//		ProcessorMetadataDirs: processorMetadataGoodDirs,
		//		ProcessorCatalogDirs:  processorCatalogBadDirs},
		//	wantErr: true,
		//	err:     ".*error unmarshaling catalog file .+/internal/processor/test/integration/resources/bad-processor-catalog/bad-processor-type.json: invalid character 'b' looking for beginning of value$",
		//},
		//{
		//	name: "missing metadata",
		//	fields: fields{
		//		CatalogChecksums:      make(map[string]string),
		//		ProcessorMetadataDirs: processorMetadataMissingDirs,
		//		ProcessorCatalogDirs:  processorCatalogGoodDirs},
		//	wantErr: true,
		//	err:     "^error listing processor catalogs in .+/internal/processor/test/integration/resources/processor-catalog: missing metadata for processor aws-sqs-source-v1alpha1$",
		//},
		//{
		//	name: "bad metadata directory",
		//	fields: fields{
		//		CatalogChecksums:      make(map[string]string),
		//		ProcessorMetadataDirs: []string{"./bad-metadata-directory"},
		//		ProcessorCatalogDirs:  []string{"./bad-catalog-directory"}},
		//	wantErr: true,
		//	err:     "^error listing processor metadata in .+/bad-metadata-directory: lstat .+/bad-metadata-directory: no such file or directory$",
		//},
		//{
		//	name: "bad metadata file",
		//	fields: fields{
		//		CatalogChecksums:      make(map[string]string),
		//		ProcessorMetadataDirs: processorMetadataBadDirs,
		//		ProcessorCatalogDirs:  processorCatalogBadDirs},
		//	wantErr: true,
		//	err:     "^error listing processor metadata in .+/internal/processor/test/integration/resources/bad-processor-metadata: error reading processor metadata from .+/internal/processor/test/integration/resources/bad-processor-metadata/bad-processor-metadata.yaml: yaml: unmarshal errors:\\n\\s*line 1: cannot unmarshal !!str `bad-con...` into \\[\\]config.ConnectorMetadata$",
		//},
		//{
		//	name: "unknown metadata",
		//	fields: fields{
		//		CatalogChecksums:      make(map[string]string),
		//		ProcessorMetadataDirs: processorMetadataUnknownDirs,
		//		ProcessorCatalogDirs:  processorCatalogGoodDirs},
		//	wantErr: true,
		//	err:     "^found 1 unrecognized processor metadata with ids: \\[unknown\\]$",
		//},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ProcessorsConfig{
				ProcessorCatalogDirs:  tt.fields.ProcessorCatalogDirs,
				ProcessorMetadataDirs: tt.fields.ProcessorMetadataDirs,
				CatalogEntries:        tt.fields.CatalogEntries,
				CatalogChecksums:      tt.fields.CatalogChecksums,
			}
			if err := c.ReadFiles(); (err != nil) != tt.wantErr {
				t.Errorf("ReadFiles() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr {
				g.Expect(err.Error()).To(gomega.MatchRegexp(tt.err))
			}

			g.Expect(c.CatalogEntries).To(gomega.HaveLen(1))
			processorID := "processor_0.1"
			g.Expect(c.CatalogEntries).To(gomega.Satisfy(func(entries []ProcessorCatalogEntry) bool {
				for i := range entries {
					if entries[i].ProcessorType.Id == processorID {
						return true
					}
				}
				return false
			}))
		})
	}

}
