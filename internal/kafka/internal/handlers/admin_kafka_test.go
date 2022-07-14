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
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	s "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account"
	"github.com/onsi/gomega"
)

func GetHandlerParams(method string, url string, body io.Reader, t *testing.T) (*http.Request, *httptest.ResponseRecorder) {
	g := gomega.NewWithT(t)
	req, err := http.NewRequest(method, url, body)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	return req, httptest.NewRecorder()
}

func Test_Get(t *testing.T) {
	type fields struct {
		kafkaService   services.KafkaService
		accountService account.AccountService
		providerConfig *config.ProviderConfig
		clusterService services.ClusterService
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
						return mocks.BuildKafkaRequest(
							mocks.WithPredefinedTestValues(),
						), nil
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig, tt.fields.clusterService)
			req, rw := GetHandlerParams("GET", "/{id}", nil, t)
			h.Get(rw, req)
			resp := rw.Result()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}

func Test_List(t *testing.T) {
	type fields struct {
		kafkaService   services.KafkaService
		accountService account.AccountService
		providerConfig *config.ProviderConfig
		clusterService services.ClusterService
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
							mocks.BuildKafkaRequest(
								mocks.WithPredefinedTestValues(),
							),
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig, tt.fields.clusterService)
			req, rw := GetHandlerParams("GET", tt.args.url, nil, t)
			h.List(rw, req)
			resp := rw.Result()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}

func Test_Delete(t *testing.T) {
	type fields struct {
		kafkaService   services.KafkaService
		accountService account.AccountService
		providerConfig *config.ProviderConfig
		clusterService services.ClusterService
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig, tt.fields.clusterService)
			req, rw := GetHandlerParams("DELETE", tt.args.url, nil, t)
			h.Delete(rw, req)
			resp := rw.Result()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}

func Test_adminKafkaHandler_Update(t *testing.T) {
	type fields struct {
		kafkaService   services.KafkaService
		accountService account.AccountService
		providerConfig *config.ProviderConfig
		clusterService services.ClusterService
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
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
					IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
						return true, nil
					},
					CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
						return true, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
					VerifyAndUpdateKafkaAdminFunc: func(ctx context.Context, kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				accountService: account.NewMockAccountService(),
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
			name: "should return an error if kafka's status is not upgradable",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID: clusterID,
						}, nil
					},
					IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
						return true, nil
					},
					CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
						return true, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return mocks.BuildKafkaRequest(
							mocks.With(mocks.STATUS, constants.KafkaRequestStatusAccepted.String()),
							mocks.With(mocks.ID, "id"),
							mocks.With(mocks.CLUSTER_ID, "cluster-id"),
							mocks.With(mocks.ACTUAL_KAFKA_IBP_VERSION, "2.7"),
							mocks.With(mocks.DESIRED_KAFKA_IBP_VERSION, "2.7"),
							mocks.With(mocks.ACTUAL_KAFKA_VERSION, "2.7"),
							mocks.With(mocks.DESIRED_KAFKA_VERSION, "2.7"),
							mocks.With(mocks.DESIRED_STRIMZI_VERSION, "2.7"),
							mocks.With(mocks.STORAGE_SIZE, "100"),
						), nil
					},
					VerifyAndUpdateKafkaAdminFunc: func(ctx context.Context, kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				accountService: account.NewMockAccountService(),
			},
			args: args{
				url:  "/kafkas/{id}",
				body: []byte(`{"kafka_ibp_version": "2.7"}`),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return error if VerifyAndUpdateKafkaAdmin returns error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID: clusterID,
						}, nil
					},
					IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
						return true, nil
					},
					CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
						return true, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status: constants.KafkaRequestStatusPreparing.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.7",
							DesiredKafkaIBPVersion: "2.8",
							ActualKafkaVersion:     "2.7",
							DesiredKafkaVersion:    "2.7",
							DesiredStrimziVersion:  "2.7",
							KafkaStorageSize:       "100",
						}, nil
					},
					VerifyAndUpdateKafkaAdminFunc: func(ctx context.Context, kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
				accountService: account.NewMockAccountService(),
			},
			args: args{
				url:  "/kafkas/{id}",
				body: []byte(`{"kafka_ibp_version": "2.7"}`),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should successfully upgrade kafka",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID: clusterID,
						}, nil
					},
					IsStrimziKafkaVersionAvailableInClusterFunc: func(cluster *api.Cluster, strimziVersion, kafkaVersion, ibpVersion string) (bool, error) {
						return true, nil
					},
					CheckStrimziVersionReadyFunc: func(cluster *api.Cluster, strimziVersion string) (bool, error) {
						return true, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status: constants.KafkaRequestStatusPreparing.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.7",
							DesiredKafkaIBPVersion: "2.8",
							ActualKafkaVersion:     "2.7",
							DesiredKafkaVersion:    "2.7",
							DesiredStrimziVersion:  "2.7",
							KafkaStorageSize:       "100",
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
				body: []byte(`{"kafka_ibp_version": "2.7"}`),
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig, tt.fields.clusterService)
			req, rw := GetHandlerParams("PATCH", tt.args.url, bytes.NewBuffer(tt.args.body), t)
			h.Update(rw, req)
			resp := rw.Result()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}
