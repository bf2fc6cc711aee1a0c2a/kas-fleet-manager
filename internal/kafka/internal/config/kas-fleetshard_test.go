package config

import (
	"testing"

	. "github.com/onsi/gomega"
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(NewKasFleetshardConfig()).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.config.ReadFiles() != nil).To(Equal(tt.wantErr))
		})
	}
}
