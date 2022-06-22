package sentry

import (
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/getsentry/sentry-go"
	. "github.com/onsi/gomega"
)

func TestInitialize(t *testing.T) {
	type args struct {
		envName environments.EnvName
		c       *Config
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{
			name: "Return error when sentry error monitoring is enabled and project config is invalid",
			args: args{
				envName: environments.EnvName("testEnv"),
				c: &Config{
					Enabled: true,
					Key:     "1234",
					URL:     "test.url",
					Project: "invalid-id",
					Debug:   true,
					Timeout: time.Hour,
					KeyFile: "secrets/sentry.key",
				},
			},
			want: &sentry.DsnParseError{Message: "invalid project id"},
		},
		{
			name: "Return nil with sentry error monitoring disabled",
			args: args{
				envName: environments.EnvName("testEnv"),
				c: &Config{
					Enabled: false,
				},
			},
			want: nil,
		},
		{
			name: "Return nil when sentry config is enabled and config is valid",
			args: args{
				envName: environments.EnvName("testEnv"),
				c: &Config{
					Enabled: true,
					Key:     "1234",
					URL:     "https//:test-url.domain",
					Project: "3",
					Debug:   true,
					Timeout: time.Hour,
					KeyFile: "secrets/sentry.key",
				},
			},
			want: nil,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			got := Initialize(tt.args.envName, tt.args.c)
			Expect(got == nil).To(Equal(tt.want == nil))
			if got != nil {
				Expect(got).To(Equal(tt.want))
			}
		})
	}
}
