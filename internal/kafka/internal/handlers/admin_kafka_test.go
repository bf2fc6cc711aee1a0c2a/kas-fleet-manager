package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	s "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account"
	. "github.com/onsi/gomega"
)

func GetHandlerParams(method string, url string, body io.Reader) (*http.Request, *httptest.ResponseRecorder) {
	req, err := http.NewRequest(method, url, body)
	Expect(err).NotTo(HaveOccurred())

	return req, httptest.NewRecorder()
}

func Test_Get(t *testing.T) {
	type fields struct {
		kafkaService   services.KafkaService
		accountService account.AccountService
		providerConfig *config.ProviderConfig
	}

	tests := []struct {
		name           string
		fields         fields
		wantStatusCode int
	}{
		{
			name: "should successfully execute GET",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{}, nil
					},
				},
				accountService: account.NewMockAccountService(),
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return an error if kafkaService Get returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
				},
				accountService: account.NewMockAccountService(),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig)
			req, rw := GetHandlerParams("GET", "/{id}", nil)
			h.Get(rw, req)
			resp := rw.Result()
			Expect(resp.StatusCode).To(Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}

func Test_List(t *testing.T) {
	type fields struct {
		kafkaService   services.KafkaService
		accountService account.AccountService
		providerConfig *config.ProviderConfig
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
			name: "should successfully return empty kafkas list",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{}, &api.PagingMeta{}, nil
					},
				},
				accountService: account.NewMockAccountService(),
			},
			args: args{
				url: "/kafkas",
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should successfully return a non-empty kafkas list",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.KafkaList{
							{
								Name: "test",
							},
						}, &api.PagingMeta{}, nil
					},
				},
				accountService: account.NewMockAccountService(),
			},
			args: args{
				url: "/kafkas",
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return an error if kafkaService List returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListFunc: func(ctx context.Context, listArgs *s.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{}, errors.GeneralError("test")
					},
				},
			},
			args: args{
				url: "/kafkas",
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return an error if orderBy param is invalid",
			args: args{
				url: "/kafkas?orderBy=invalidField",
			},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig)
			req, rw := GetHandlerParams("GET", tt.args.url, nil)
			h.List(rw, req)
			resp := rw.Result()
			Expect(resp.StatusCode).To(Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}

func Test_Delete(t *testing.T) {
	type fields struct {
		kafkaService   services.KafkaService
		accountService account.AccountService
		providerConfig *config.ProviderConfig
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
			name: "should successfully accept kafka deletion request",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
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
		{
			name: "should return an error if async flag is not set to true when deleting kafka",
			args: args{
				url: "/kafkas/{id}",
			},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig)
			req, rw := GetHandlerParams("DELETE", tt.args.url, nil)
			h.Delete(rw, req)
			resp := rw.Result()
			Expect(resp.StatusCode).To(Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}

func Test_Update(t *testing.T) {
	type fields struct {
		kafkaService   services.KafkaService
		accountService account.AccountService
		providerConfig *config.ProviderConfig
	}

	type args struct {
		url  string
		body []byte
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return an error if retrieving kafka to update fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
				},
			},
			args: args{
				url:  "/kafkas/{id}",
				body: []byte(`{}`),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return an error if kafka to update can't be found",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			args: args{
				url:  "/kafkas/{id}",
				body: []byte(`{}`),
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "should return an error if kafka's status is not upgradable",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status: constants.KafkaRequestStatusAccepted.String(),
						}, nil
					},
				},
			},
			args: args{
				url:  "/kafkas/{id}",
				body: []byte(`{"kafka_version": "2.8.0"}`),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if kafka version is already being upgraded",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status:         constants.KafkaRequestStatusPreparing.String(),
							KafkaUpgrading: true,
						}, nil
					},
				},
			},
			args: args{
				url:  "/kafkas/{id}",
				body: []byte(`{"kafka_version": "2.8.0"}`),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if strimzi version is already being upgraded",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status:           constants.KafkaRequestStatusPreparing.String(),
							StrimziUpgrading: true,
						}, nil
					},
				},
			},
			args: args{
				url:  "/kafkas/{id}",
				body: []byte(`{"strimzi_version": "2.8.0"}`),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if ibp version is already being upgraded",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status:            constants.KafkaRequestStatusPreparing.String(),
							KafkaIBPUpgrading: true,
						}, nil
					},
				},
			},
			args: args{
				url:  "/kafkas/{id}",
				body: []byte(`{"kafka_ibp_version": "2.8.0"}`),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if kafka update by kafkaService fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status: constants.KafkaRequestStatusPreparing.String(),
						}, nil
					},
					VerifyAndUpdateKafkaAdminFunc: func(ctx context.Context, kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
			},
			args: args{
				url:  "/kafkas/{id}",
				body: []byte(`{"kafka_ibp_version": "2.8.0"}`),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should successfully upgrade kafka",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status: constants.KafkaRequestStatusPreparing.String(),
						}, nil
					},
					VerifyAndUpdateKafkaAdminFunc: func(ctx context.Context, kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				accountService: account.NewMockAccountService(),
			},
			args: args{
				url:  "/kafkas/{id}",
				body: []byte(`{"kafka_ibp_version": "2.8.0"}`),
			},
			wantStatusCode: http.StatusOK,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig)
			req, rw := GetHandlerParams("PATCH", tt.args.url, bytes.NewBuffer(tt.args.body))
			h.Update(rw, req)
			resp := rw.Result()
			Expect(resp.StatusCode).To(Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}
