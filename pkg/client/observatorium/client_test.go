package observatorium

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/onsi/gomega"
	pAPI "github.com/prometheus/client_golang/api"
	pV1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var (
	observabilityConfiguration = NewObservabilityConfigurationConfig()
	configuration              = ClientConfiguration{
		Timeout:  observabilityConfiguration.Timeout,
		Insecure: observabilityConfiguration.Insecure,
		BaseURL:  "",
	}
	invalidUrl = ":::"
)

func Test_NewObservatoriumClient(t *testing.T) {
	type args struct {
		c *ObservabilityConfiguration
	}

	tests := []struct {
		name     string
		modifyFn func(config *ObservabilityConfiguration)
		args     args
		wantErr  bool
	}{
		{
			name: "should not return an error when providing default ObservabilityConfiguration",
			args: args{
				c: NewObservabilityConfigurationConfig(),
			},
			wantErr: false,
		},
		{
			name: "should not return an error when providing default ObservabilityConfiguration",
			args: args{
				c: NewObservabilityConfigurationConfig(),
			},
			wantErr: false,
		},
		{
			name: "should not return an error when providing default ObservabilityConfiguration with mock enabled",
			args: args{
				c: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.EnableMock = true
			},
			wantErr: false,
		},
		{
			name: "should return an error when providing default ObservabilityConfiguration with and an invalid token refresher url format",
			args: args{
				c: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.RedHatSsoTokenRefresherUrl = invalidUrl
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			config := tt.args.c
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			_, err := NewObservatoriumClient(config)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_NewClient(t *testing.T) {
	type args struct {
		c *ClientConfiguration
	}

	tests := []struct {
		name     string
		args     args
		modifyFn func(config *ClientConfiguration)
		wantErr  bool
	}{
		// only one test case required. the rest is covered in the Test_NewObservatoriumClient
		{
			name: "should return an error when creating a client with invalid url",
			args: args{
				c: &configuration,
			},
			modifyFn: func(config *ClientConfiguration) {
				config.BaseURL = invalidUrl
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			config := tt.args.c
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			g.Expect(NewClient(config)).Error().Should(gomega.HaveOccurred())
		})
	}
}

func Test_RoundTrip(t *testing.T) {
	g := gomega.NewWithT(t)

	req, err := http.NewRequest(http.MethodGet, "https://httpbin.org/get", nil)
	g.Expect(err).To(gomega.BeNil())

	config := ClientConfiguration{
		Timeout:    configuration.Timeout,
		AuthToken:  configuration.AuthToken,
		EnableMock: false,
		Insecure:   configuration.Insecure,
	}
	type fields struct {
		p observatoriumRoundTripper
	}
	type args struct {
		request *http.Request
	}
	tests := []struct {
		name     string
		args     args
		fields   fields
		modifyFn func(config *observatoriumRoundTripper)
		wantErr  bool
	}{
		{
			name: "should return no error with default values of roundTripper and sample valid request",
			fields: fields{
				p: observatoriumRoundTripper{
					config: config,
					wrapped: observatoriumRoundTripper{
						config:  config,
						wrapped: pAPI.DefaultRoundTripper,
					},
				},
			},
			args: args{
				request: req,
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := tt.fields.p
			if tt.modifyFn != nil {
				tt.modifyFn(&p)
			}
			resp, err := p.RoundTrip(tt.args.request)
			if resp != nil {
				_ = resp.Body.Close()
			}
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_Query(t *testing.T) {
	c := observabilityConfiguration
	c.EnableMock = true
	g := gomega.NewWithT(t)

	client, err := NewObservatoriumClient(c)
	g.Expect(err).To(gomega.BeNil())
	type fields struct {
		c *Client
	}
	type args struct {
		queryTemplate string
		label         string
	}

	tests := []struct {
		name   string
		args   args
		fields fields
	}{
		{
			name: "should successfully execute Query",
			fields: fields{
				c: client,
			},
			args: args{
				queryTemplate: "kafka_instance_connection_limit{%s}",
				label:         "",
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.c.Query(tt.args.queryTemplate, tt.args.label).Err).ToNot(gomega.HaveOccurred())
		})
	}
}

func Test_QueryRange(t *testing.T) {
	c := observabilityConfiguration
	c.EnableMock = true
	g := gomega.NewWithT(t)

	client, err := NewObservatoriumClient(c)
	g.Expect(err).To(gomega.BeNil())
	type fields struct {
		c *Client
	}
	type args struct {
		queryTemplate string
		label         string
		bounds        pV1.Range
	}

	tests := []struct {
		name   string
		args   args
		fields fields
	}{
		{
			name: "should successfully execute QueryRange",
			fields: fields{
				c: client,
			},
			args: args{
				queryTemplate: "kafka_instance_connection_limit{%s}",
				label:         "",
				bounds:        pV1.Range{},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			g.Expect(tt.fields.c.QueryRange(tt.args.queryTemplate, tt.args.label, tt.args.bounds).Err).ToNot(gomega.HaveOccurred())
		})
	}
}

func Test_all(t *testing.T) {
	testValue := "test"
	testConj := "#"
	type args struct {
		items       []string
		conjunction string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should return an empty string with empty args",
			want: "",
		},
		{
			name: "should return formatted string when passing items of length 1",
			args: args{
				items: []string{testValue},
			},
			want: fmt.Sprintf("'%s'", testValue),
		},
		{
			name: "should return formatted string when passing multiple strings in the items",
			args: args{
				items:       []string{testValue, testValue},
				conjunction: testConj,
			},
			want: fmt.Sprintf("'%s' %s '%s'", testValue, testConj, testValue),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(all(tt.args.items, tt.args.conjunction)).To(gomega.Equal(tt.want))
		})
	}
}
