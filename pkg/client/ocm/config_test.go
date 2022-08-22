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

func TestOCMConfig_HasCredentials(t *testing.T) {
	type fields struct {
		ClientID     string
		ClientSecret string
		SelfToken    string
	}
	ClientID := "ClientID"
	ClientSecret := "ClientSecret"
	SelfToken := "SelfToken"
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return true if ClientID and ClientSecret are set",
			fields: fields{
				ClientID:     ClientID,
				ClientSecret: ClientSecret,
			},
			want: true,
		},
		{
			name: "should return true if SelfToken is set",
			fields: fields{
				SelfToken: SelfToken,
			},
			want: true,
		},
		{
			name:   "should return false if no credentials are set",
			fields: fields{},
			want:   false,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &OCMConfig{
				ClientID:     tt.fields.ClientID,
				ClientSecret: tt.fields.ClientSecret,
				SelfToken:    tt.fields.SelfToken,
			}
			g.Expect(c.HasCredentials()).To(gomega.Equal(tt.want))
		})
	}
}
