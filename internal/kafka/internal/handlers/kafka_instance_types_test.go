package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"

	"testing"
)

var (
	cloudProvider = "aws"
	cloudRegion   = "us-east-1"
)

func Test_ListSupportedKafkaInstanceTypes(t *testing.T) {
	type fields struct {
		supportedKafkaInstanceTypesService services.SupportedKafkaInstanceTypesService
	}

	type args struct {
		cloudProvider string
		cloudRegion   string
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should fail if neither cloud_provider nor cloud_region is provided in the request",
			fields: fields{
				supportedKafkaInstanceTypesService: &services.SupportedKafkaInstanceTypesServiceMock{},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should fail if cloud_region is not provided in the request",
			args: args{
				cloudProvider: cloudProvider,
			},
			fields: fields{
				supportedKafkaInstanceTypesService: &services.SupportedKafkaInstanceTypesServiceMock{},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should fail if cloud_provider is not provided in the request",
			args: args{
				cloudRegion: cloudRegion,
			},
			fields: fields{
				supportedKafkaInstanceTypesService: &services.SupportedKafkaInstanceTypesServiceMock{},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should fail if GetSupportedKafkaInstanceTypesByRegion fails with unsupported instance type error",
			args: args{
				cloudRegion:   cloudRegion,
				cloudProvider: cloudProvider,
			},
			fields: fields{
				supportedKafkaInstanceTypesService: &services.SupportedKafkaInstanceTypesServiceMock{
					GetSupportedKafkaInstanceTypesByRegionFunc: func(providerId, regionId string) ([]config.KafkaInstanceType, *errors.ServiceError) {
						return nil, errors.InstanceTypeNotSupported("instance type not supported")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should fail if GetSupportedKafkaInstanceTypesByRegion fails with error other than unsupported instance type error",
			args: args{
				cloudRegion:   cloudRegion,
				cloudProvider: cloudProvider,
			},
			fields: fields{
				supportedKafkaInstanceTypesService: &services.SupportedKafkaInstanceTypesServiceMock{
					GetSupportedKafkaInstanceTypesByRegionFunc: func(providerId, regionId string) ([]config.KafkaInstanceType, *errors.ServiceError) {
						return nil, errors.GeneralError("error occurred")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return the slice of KafkaInstanceTypes",
			args: args{
				cloudRegion:   cloudRegion,
				cloudProvider: cloudProvider,
			},
			fields: fields{
				supportedKafkaInstanceTypesService: &services.SupportedKafkaInstanceTypesServiceMock{
					GetSupportedKafkaInstanceTypesByRegionFunc: func(providerId, regionId string) ([]config.KafkaInstanceType, *errors.ServiceError) {
						return []config.KafkaInstanceType{
							{
								Id:          "developer",
								DisplayName: "Trial",
								Sizes: []config.KafkaInstanceSize{
									{
										Id:          "x1",
										DisplayName: "1",
									},
								},
							},
						}, nil
					},
				},
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewSupportedKafkaInstanceTypesHandler(tt.fields.supportedKafkaInstanceTypesService)
			req, rw := GetHandlerParams("GET", "/cloud_provider=aws", nil, t)
			muxVars := map[string]string{
				"cloud_provider": tt.args.cloudProvider,
				"cloud_region":   tt.args.cloudRegion,
			}
			req = mux.SetURLVars(req, muxVars)
			h.ListSupportedKafkaInstanceTypes(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}
