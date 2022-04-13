package sentry

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_Sentry_ReadFiles(t *testing.T) {
	configWithSentryDisabled := NewConfig()
	configWithSentryDisabled.Enabled = false
	configWithSentryDisabled.KeyFile = "a file that does not exists so that to show that we are not performing file reading"
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "Do not read file when sentry disabled",
			wantErr: false,
			config:  configWithSentryDisabled,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ReadFiles()
			Expect(tt.wantErr).To(Equal(err != nil))
		})
	}
}
