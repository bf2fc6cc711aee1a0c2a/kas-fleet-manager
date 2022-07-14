package ocm

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_ReadFiles(t *testing.T) {
	tests := []struct {
		name     string
		modifyFn func(config *OCMConfig)
		wantErr  bool
	}{
		{
			name:    "should return no error when running ReadFiles with default OCMConfig",
			wantErr: false,
		},
		{
			name: "should return no error with misconfigured ClientIDFile",
			modifyFn: func(config *OCMConfig) {
				config.ClientIDFile = "invalid"
			},
			wantErr: false,
		},
		{
			name: "should return no error with misconfigured ClientSecretFile",
			modifyFn: func(config *OCMConfig) {
				config.ClientSecretFile = "invalid"
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			config := NewOCMConfig()
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			g.Expect(config.ReadFiles() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
