package observatorium

import (
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	pAPI "github.com/prometheus/client_golang/api"
	pV1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var (
	observabilityConfiguration = NewObservabilityConfigurationConfig()
	configuration              = Configuration{
		Cookie:   observabilityConfiguration.Cookie,
		Timeout:  observabilityConfiguration.Timeout,
		Debug:    observabilityConfiguration.Debug,
		Insecure: observabilityConfiguration.Insecure,
		AuthType: "test",
		BaseURL:  "",
	}
	testValue = "test"
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
			name: "should not return an error when providing default ObservabilityConfiguration with AuthType = AuthTypeSso",
			args: args{
				c: NewObservabilityConfigurationConfig(),
			},
			modifyFn: func(config *ObservabilityConfiguration) {
				config.AuthType = AuthTypeSso
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
	}
	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.args.c
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			_, err := NewObservatoriumClient(config)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_NewClient(t *testing.T) {
	type args struct {
		c *Configuration
	}

	tests := []struct {
		name     string
		args     args
		modifyFn func(config *Configuration)
		wantErr  bool
	}{
		// only one test case required. the rest is covered in the Test_NewObservatoriumClient
		{
			name: "should return an error when creating a client with invalid url",
			args: args{
				c: &configuration,
			},
			modifyFn: func(config *Configuration) {
				config.BaseURL = ":::" // invalid url
			},
			wantErr: true,
		},
	}
	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.args.c
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			_, err := NewClient(config)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_RoundTrip(t *testing.T) {
	RegisterTestingT(t)

	req, err := http.NewRequest(http.MethodGet, "https://httpbin.org/get", nil)
	Expect(err).To(BeNil())

	config := ClientConfiguration{
		Timeout:    configuration.Timeout,
		AuthToken:  configuration.AuthToken,
		Cookie:     configuration.Cookie,
		Debug:      configuration.Debug,
		EnableMock: false,
		Insecure:   configuration.Insecure,
		AuthType:   configuration.AuthType,
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
		{
			name: "should return no error with roundTripper's AuthType set to AuthTypeDex and sample valid request and non-empty AuthToken",
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
			modifyFn: func(config *observatoriumRoundTripper) {
				config.config.AuthType = AuthTypeDex
				config.config.AuthToken = testValue
			},
			wantErr: false,
		},
		{
			name: "should return no error with roundTripper's AuthType set to AuthTypeDex and sample valid request and non-empty Cookie",
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
			modifyFn: func(config *observatoriumRoundTripper) {
				config.config.AuthType = AuthTypeDex
				config.config.Cookie = testValue
			},
			wantErr: false,
		},
		{
			name: "should return an error with roundTripper's AuthType set to AuthTypeDex empty auth fields",
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
			modifyFn: func(config *observatoriumRoundTripper) {
				config.config.AuthType = AuthTypeDex
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.fields.p
			if tt.modifyFn != nil {
				tt.modifyFn(&p)
			}
			_, err := p.RoundTrip(tt.args.request)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_Query(t *testing.T) {
	c := observabilityConfiguration
	c.EnableMock = true
	RegisterTestingT(t)

	client, err := NewObservatoriumClient(c)
	Expect(err).To(BeNil())
	type fields struct {
		c *Client
	}
	type args struct {
		queryTemplate string
		label         string
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
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
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.c.Query(tt.args.queryTemplate, tt.args.label).Err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_QueryRange(t *testing.T) {
	c := observabilityConfiguration
	c.EnableMock = true
	RegisterTestingT(t)

	client, err := NewObservatoriumClient(c)
	Expect(err).To(BeNil())
	type fields struct {
		c *Client
	}
	type args struct {
		queryTemplate string
		label         string
		bounds        pV1.Range
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
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
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.c.QueryRange(tt.args.queryTemplate, tt.args.label, tt.args.bounds).Err != nil).To(Equal(tt.wantErr))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(all(tt.args.items, tt.args.conjunction)).To(Equal(tt.want))
		})
	}
}
