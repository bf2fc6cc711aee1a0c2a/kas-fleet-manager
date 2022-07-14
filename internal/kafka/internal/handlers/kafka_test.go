package handlers

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	mocksupportedinstancetypes "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/supported_instance_types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	s "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
)

var (
	id  = "test-id"
	ctx = auth.SetTokenInContext(context.TODO(), &jwt.Token{
		Claims: jwt.MapClaims{
			"username":     "test-user",
			"org_id":       mocks.DefaultOrganisationId,
			"is_org_admin": true,
		},
	})
	limit = 2
)

func Test_KafkaHandler_Get(t *testing.T) {
	type fields struct {
		service        services.KafkaService
		providerConfig *config.ProviderConfig
		authService    authorization.Authorization
		kafkaConfig    *config.KafkaConfig
	}

	tests := []struct {
		name           string
		fields         fields
		wantStatusCode int
	}{
		{
			name: "should succeed if kafkaService GET succeeds",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
				},
				kafkaConfig: &fullKafkaConfig,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should fail if kafkaService GET fails",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewKafkaHandler(tt.fields.service, tt.fields.providerConfig, tt.fields.authService, tt.fields.kafkaConfig)
			req, rw := GetHandlerParams("GET", "/{id}", nil, t)
			req = mux.SetURLVars(req, map[string]string{"id": id})
			h.Get(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_KafkaHandler_Delete(t *testing.T) {
	type fields struct {
		service        services.KafkaService
		providerConfig *config.ProviderConfig
		authService    authorization.Authorization
		kafkaConfig    *config.KafkaConfig
	}

	type args struct {
		url string
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name:           "fails if async is not set",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "fails if RegisterKafkaDeprovisionJob fails in kafka service",
			fields: fields{
				service: &services.KafkaServiceMock{
					RegisterKafkaDeprovisionJobFunc: func(ctx context.Context, id string) *errors.ServiceError {
						return errors.GeneralError("register kafka deprovision job failed")
					},
				},
			},
			args: args{
				url: "/kafkas/{id}?async=true",
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "fails if RegisterKafkaDeprovisionJob fails in kafka service",
			fields: fields{
				service: &services.KafkaServiceMock{
					RegisterKafkaDeprovisionJobFunc: func(ctx context.Context, id string) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				url: "/kafkas/{id}?async=true",
			},
			wantStatusCode: http.StatusAccepted,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewKafkaHandler(tt.fields.service, tt.fields.providerConfig, tt.fields.authService, tt.fields.kafkaConfig)
			req, rw := GetHandlerParams("DELETE", tt.args.url, nil, t)
			h.Delete(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_KafkaHandler_List(t *testing.T) {
	type fields struct {
		service        services.KafkaService
		providerConfig *config.ProviderConfig
		authService    authorization.Authorization
		kafkaConfig    *config.KafkaConfig
	}

	type args struct {
		url string
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "fails if GetAcceptedOrderByParams validation fails",
			args: args{
				url: "/kafkas?orderBy=invalidField",
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "fails if List in the kafka service returns an error",
			fields: fields{
				service: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{}, errors.GeneralError("ListFunc returned an error")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "succeeds if List in the kafka service returns an empty list",
			fields: fields{
				service: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{}, &api.PagingMeta{}, nil
					},
				},
				kafkaConfig: &fullKafkaConfig,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "succeeds if List in the kafka service returns a non empty list",
			fields: fields{
				service: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{
							mocks.BuildKafkaRequest(
								mocks.WithPredefinedTestValues(),
							),
						}, &api.PagingMeta{}, nil
					},
				},
				kafkaConfig: &fullKafkaConfig,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "fails if its unable to Present KafkaRequest due to invalid Instance type",
			fields: fields{
				service: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues(), mocks.With(mocks.INSTANCE_TYPE, "invalid"))}, &api.PagingMeta{}, nil
					},
				},
				kafkaConfig: &fullKafkaConfig,
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewKafkaHandler(tt.fields.service, tt.fields.providerConfig, tt.fields.authService, tt.fields.kafkaConfig)
			req, rw := GetHandlerParams("GET", tt.args.url, nil, t)
			h.List(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_KafkaHandler_Update(t *testing.T) {
	type fields struct {
		service        services.KafkaService
		providerConfig *config.ProviderConfig
		authService    authorization.Authorization
		kafkaConfig    *config.KafkaConfig
	}

	type args struct {
		url  string
		body []byte
		ctx  context.Context
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "succeeds if reauthentication is enabled - updated",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						return nil
					},
				},
				kafkaConfig: &fullKafkaConfig,
			},
			args: args{
				body: []byte(`{"reauthentication_enabled": true}`),
				ctx:  ctx,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "succeeds if the owner value is set",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						return nil
					},
				},
				authService: &authorization.AuthorizationMock{
					CheckUserValidFunc: func(username, orgId string) (bool, error) {
						return true, nil
					},
				},
				kafkaConfig: &fullKafkaConfig,
			},
			args: args{
				body: []byte(`{"owner": "owner"}`),
				ctx:  ctx,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "fails if Updates in the kafka service returns an error",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdatesFunc: func(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError {
						return errors.GeneralError("update fail")
					},
				},
				authService: &authorization.AuthorizationMock{
					CheckUserValidFunc: func(username, orgId string) (bool, error) {
						return true, nil
					},
				},
				kafkaConfig: &fullKafkaConfig,
			},
			args: args{
				body: []byte(`{"owner": "owner"}`),
				ctx:  ctx,
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewKafkaHandler(tt.fields.service, tt.fields.providerConfig, tt.fields.authService, tt.fields.kafkaConfig)
			req, rw := GetHandlerParams("PATCH", tt.args.url, bytes.NewBuffer(tt.args.body), t)
			req = req.WithContext(tt.args.ctx)
			h.Update(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_KafkaHandler_Create(t *testing.T) {
	type fields struct {
		service        services.KafkaService
		providerConfig *config.ProviderConfig
		authService    authorization.Authorization
		kafkaConfig    *config.KafkaConfig
	}

	type args struct {
		url  string
		body []byte
		ctx  context.Context
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "succeeds if RegisterKafkaJob in the kafka service succeds",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{}, &api.PagingMeta{}, nil
					},
					RegisterKafkaJobFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						kafkaRequest.KafkaStorageSize = mocksupportedinstancetypes.DefaultMaxDataRetentionSize
						return nil
					},
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.STANDARD, nil
					},
				},
				providerConfig: &supportedProviders,
				kafkaConfig:    &fullKafkaConfig,
			},
			args: args{
				url:  "/kafkas?async=true",
				body: []byte(`{"name": "name", "cloud_provider": "aws", "region": "us-east-1"}`),
				ctx:  ctx,
			},
			wantStatusCode: http.StatusAccepted,
		},
		{
			name: "fails if RegisterKafkaJob in the kafka service retuns an error",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{}, &api.PagingMeta{}, nil
					},
					RegisterKafkaJobFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("create failed")
					},
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.STANDARD, nil
					},
				},
				providerConfig: &supportedProviders,
				kafkaConfig:    &fullKafkaConfig,
			},
			args: args{
				url:  "/kafkas?async=true",
				body: []byte(`{"name": "name", "cloud_provider": "aws", "region": "us-east-1"}`),
				ctx:  ctx,
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "fails if validation fails while async is not enabled",
			args: args{
				ctx: ctx,
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "fails if name validation fails",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
				},
			},
			args: args{
				url:  "/kafkas?async=true",
				body: []byte(`{}`),
				ctx:  ctx,
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "fails if cloud provider validation fails",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{}, &api.PagingMeta{}, nil
					},
				},
				providerConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name:    "aws",
								Default: true,
								Regions: config.RegionList{
									config.Region{
										Name:    "us-west-1",
										Default: true,
										SupportedInstanceTypes: config.InstanceTypeMap{
											"standard": {
												Limit: &limit,
											},
										},
									},
								},
							},
						},
					},
				},
				kafkaConfig: &fullKafkaConfig,
			},
			args: args{
				url:  "/kafkas?async=true",
				body: []byte(`{"name": "name", "cloud_provider": "aws", "region": "us-east-1"}`),
				ctx:  ctx,
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "fails if kafka plan validation fails",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{}, &api.PagingMeta{}, nil
					},
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.KafkaInstanceType("invalid"), errors.GeneralError("Unsupported plan provided")
					},
				},
				providerConfig: &supportedProviders,
				kafkaConfig:    &fullKafkaConfig,
			},
			args: args{
				url:  "/kafkas?async=true",
				body: []byte(`{"name": "name", "cloud_provider": "aws", "plan":"developer.invalidPlan"}`),
				ctx:  ctx,
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "fails if ValidateBillingCloudAccountIdAndMarketplace fails",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{}, &api.PagingMeta{}, nil
					},
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.STANDARD, nil
					},
				},
				providerConfig: &supportedProviders,
				kafkaConfig:    &fullKafkaConfig,
			},
			args: args{
				url:  "/kafkas?async=true",
				body: []byte(`{"name": "name", "cloud_provider": "aws", "marketplace":"redhat"}`),
				ctx:  ctx,
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "fails if BillingModelValidation fails",
			fields: fields{
				service: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(mocks.WithPredefinedTestValues()), nil
					},
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{}, &api.PagingMeta{}, nil
					},
					AssignInstanceTypeFunc: func(owner, organisationID string) (types.KafkaInstanceType, *errors.ServiceError) {
						return types.STANDARD, nil
					},
				},
				providerConfig: &supportedProviders,
				kafkaConfig:    &fullKafkaConfig,
			},
			args: args{
				url:  "/kafkas?async=true",
				body: []byte(`{"name": "name", "cloud_provider": "aws", "billing_model":"invalid"}`),
				ctx:  ctx,
			},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewKafkaHandler(tt.fields.service, tt.fields.providerConfig, tt.fields.authService, tt.fields.kafkaConfig)
			req, rw := GetHandlerParams("CREATE", tt.args.url, bytes.NewBuffer(tt.args.body), t)
			req = req.WithContext(tt.args.ctx)
			h.Create(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}
