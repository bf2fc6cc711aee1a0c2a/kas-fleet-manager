package ocm

import (
	"testing"

	. "github.com/onsi/gomega"
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewOCMConfig()
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			Expect(config.ReadFiles() != nil).To(Equal(tt.wantErr))
		})
	}
}
