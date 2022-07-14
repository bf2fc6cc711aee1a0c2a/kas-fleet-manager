package config

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_NewKasFleetshardConfig(t *testing.T) {
	tests := []struct {
		name string
		want *KasFleetshardConfig
	}{
		{
			name: "should return NewKasFleetshardConfig",
			want: &KasFleetshardConfig{
				PollInterval:   "15s",
				ResyncInterval: "60s",
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewKasFleetshardConfig()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ReadFilesKasFleetshardConfig(t *testing.T) {
	type fields struct {
		config *KasFleetshardConfig
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return no error when running ReadFiles with default NewKasFleetshardConfig",
			fields: fields{
				config: NewKasFleetshardConfig(),
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.config.ReadFiles() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
