package sentry

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_Sentry_ReadFiles(t *testing.T) {
	tests := []struct {
		name     string
		modifyFn func(config *Config)
		wantErr  bool
	}{
		{
			name: "Do not read file when sentry disabled",
			modifyFn: func(config *Config) {
				config.Enabled = false
				config.KeyFile = "a file that does not exists so that to show that we are not performing file reading"
			},
			wantErr: false,
		},
		{
			name: "Should return an error with non-existent KeyFile and sentry enabled.",
			modifyFn: func(config *Config) {
				config.Enabled = true
				config.KeyFile = "a file that does not exists to show that we will perform a file reading with sentry enabled and return error"
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			config := NewConfig()
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			g.Expect(config.ReadFiles() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
