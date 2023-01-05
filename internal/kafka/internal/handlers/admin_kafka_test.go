package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
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

const (
	kafkaByIdUrl = "/kafkas/{id}"
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
		kafkaConfig    *config.KafkaConfig
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
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig, tt.fields.clusterService, tt.fields.kafkaConfig)
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
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig, tt.fields.clusterService, tt.fields.kafkaConfig)
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
				url: kafkaByIdUrl,
			},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig, tt.fields.clusterService, tt.fields.kafkaConfig)
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
		kafkaConfig    *config.KafkaConfig
	}
	type args struct {
		url  string
		body []byte
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		wantStatusCode  int
		wantKafkaStatus constants.KafkaStatus
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
				url:  kafkaByIdUrl,
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
				url:  kafkaByIdUrl,
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
				url:  kafkaByIdUrl,
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
				url:  kafkaByIdUrl,
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
				url:  kafkaByIdUrl,
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
				url:  kafkaByIdUrl,
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
				url:  kafkaByIdUrl,
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
				url:  kafkaByIdUrl,
				body: []byte(`{"kafka_ibp_version": "2.7"}`),
			},
			wantStatusCode:  http.StatusOK,
			wantKafkaStatus: constants.KafkaRequestStatusPreparing,
		},
		{
			name: "should not set kafka in deprovision state into suspending state",
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
							Status: constants.KafkaRequestStatusDeprovision.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.8",
							DesiredKafkaIBPVersion: "2.8",
							ActualKafkaVersion:     "2.8",
							DesiredKafkaVersion:    "2.8",
							DesiredStrimziVersion:  "2.8",
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
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": true}`),
			},
			wantStatusCode:  http.StatusBadRequest,
			wantKafkaStatus: constants.KafkaRequestStatusDeprovision,
		},
		{
			name: "should fail when setting kafka in deleting state into suspending state",
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
							Status: constants.KafkaRequestStatusDeleting.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.8",
							DesiredKafkaIBPVersion: "2.8",
							ActualKafkaVersion:     "2.8",
							DesiredKafkaVersion:    "2.8",
							DesiredStrimziVersion:  "2.8",
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
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": true}`),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should set not already suspending or suspended kafka into suspending state",
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
							Status: constants.KafkaRequestStatusReady.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.8",
							DesiredKafkaIBPVersion: "2.8",
							ActualKafkaVersion:     "2.8",
							DesiredKafkaVersion:    "2.8",
							DesiredStrimziVersion:  "2.8",
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
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": true}`),
			},
			wantStatusCode:  http.StatusOK,
			wantKafkaStatus: constants.KafkaRequestStatusSuspending,
		},
		{
			name: "should return an error when trying to resume a kafka instance in suspending state",
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
							Status: constants.KafkaRequestStatusSuspending.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.8",
							DesiredKafkaIBPVersion: "2.8",
							ActualKafkaVersion:     "2.8",
							DesiredKafkaVersion:    "2.8",
							DesiredStrimziVersion:  "2.8",
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
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": false}`),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error when trying to resume an instance that is already ready",
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
							Status: constants.KafkaRequestStatusReady.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.8",
							DesiredKafkaIBPVersion: "2.8",
							ActualKafkaVersion:     "2.8",
							DesiredKafkaVersion:    "2.8",
							DesiredStrimziVersion:  "2.8",
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
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": false}`),
			},
			wantStatusCode:  http.StatusBadRequest,
			wantKafkaStatus: constants.KafkaRequestStatusReady,
		},
		{
			name: "should have no effect on a non suspending or suspended kafka instance if suspended param is not provided",
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
							Status: constants.KafkaRequestStatusReady.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.7",
							DesiredKafkaIBPVersion: "2.7",
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
				url:  kafkaByIdUrl,
				body: []byte(`{"kafka_ibp_version": "2.7"}`),
			},
			wantStatusCode:  http.StatusOK,
			wantKafkaStatus: constants.KafkaRequestStatusReady,
		},
		{
			name: "should return an error when trying to suspend a kafka instance that is already suspended when param is set to true",
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
							Status: constants.KafkaRequestStatusSuspended.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.8",
							DesiredKafkaIBPVersion: "2.8",
							ActualKafkaVersion:     "2.8",
							DesiredKafkaVersion:    "2.8",
							DesiredStrimziVersion:  "2.8",
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
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": true}`),
			},
			wantStatusCode:  http.StatusBadRequest,
			wantKafkaStatus: constants.KafkaRequestStatusSuspended,
		},
		{
			name: "should return an error when trying to suspend a kafka in suspending state when suspended param is set to true",
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
							Status: constants.KafkaRequestStatusSuspending.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.8",
							DesiredKafkaIBPVersion: "2.8",
							ActualKafkaVersion:     "2.8",
							DesiredKafkaVersion:    "2.8",
							DesiredStrimziVersion:  "2.8",
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
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": true}`),
			},
			wantStatusCode:  http.StatusBadRequest,
			wantKafkaStatus: constants.KafkaRequestStatusSuspending,
		},
		{
			name: "should resume an instance in suspended state which has no expiration set",
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
							Status: constants.KafkaRequestStatusSuspended.String(),
							Meta: api.Meta{
								ID: "id",
							},
							ClusterID:              "cluster-id",
							ActualKafkaIBPVersion:  "2.8",
							DesiredKafkaIBPVersion: "2.8",
							ActualKafkaVersion:     "2.8",
							DesiredKafkaVersion:    "2.8",
							DesiredStrimziVersion:  "2.8",
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
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": false}`),
			},
			wantStatusCode:  http.StatusOK,
			wantKafkaStatus: constants.KafkaRequestStatusResuming,
		},
		{
			name: "should return an error when trying to resume an instance that is suspended, it has an expiration time set and it is within its grace period",
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
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								config.KafkaInstanceType{
									Id: "myinstancetype",
									SupportedBillingModels: []config.KafkaBillingModel{
										config.KafkaBillingModel{
											ID:              "mybillingmodel",
											GracePeriodDays: 10,
										},
									},
								},
							},
						},
					},
				},
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status: constants.KafkaRequestStatusSuspended.String(),
							Meta: api.Meta{
								ID: "id",
							},
							InstanceType:            "myinstancetype",
							ClusterID:               "cluster-id",
							ActualKafkaIBPVersion:   "2.8",
							DesiredKafkaIBPVersion:  "2.8",
							ActualKafkaVersion:      "2.8",
							DesiredKafkaVersion:     "2.8",
							DesiredStrimziVersion:   "2.8",
							KafkaStorageSize:        "100",
							ActualKafkaBillingModel: "mybillingmodel",
							ExpiresAt:               sql.NullTime{Time: time.Now().Add(48 * time.Hour), Valid: true},
						}, nil
					},
					VerifyAndUpdateKafkaAdminFunc: func(ctx context.Context, kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				accountService: account.NewMockAccountService(),
			},
			args: args{
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": false}`),
			},
			wantStatusCode:  http.StatusBadRequest,
			wantKafkaStatus: constants.KafkaRequestStatusSuspended,
		},
		{
			name: "should resume a suspended instance that is suspended but has no expiration date set, even if grace period is configured",
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
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								config.KafkaInstanceType{
									Id: "myinstancetype",
									SupportedBillingModels: []config.KafkaBillingModel{
										config.KafkaBillingModel{
											ID:              "mybillingmodel",
											GracePeriodDays: 10,
										},
									},
								},
							},
						},
					},
				},
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status: constants.KafkaRequestStatusSuspended.String(),
							Meta: api.Meta{
								ID: "id",
							},
							InstanceType:            "myinstancetype",
							ClusterID:               "cluster-id",
							ActualKafkaIBPVersion:   "2.8",
							DesiredKafkaIBPVersion:  "2.8",
							ActualKafkaVersion:      "2.8",
							DesiredKafkaVersion:     "2.8",
							DesiredStrimziVersion:   "2.8",
							KafkaStorageSize:        "100",
							ActualKafkaBillingModel: "mybillingmodel",
						}, nil
					},
					VerifyAndUpdateKafkaAdminFunc: func(ctx context.Context, kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				accountService: account.NewMockAccountService(),
			},
			args: args{
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": false}`),
			},
			wantStatusCode:  http.StatusOK,
			wantKafkaStatus: constants.KafkaRequestStatusResuming,
		},
		{
			name: "should resume a suspended instance that has an expiration date set but it has not reached that date nor the grace period stage",
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
				kafkaConfig: &config.KafkaConfig{
					SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
						Configuration: config.SupportedKafkaInstanceTypesConfig{
							SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
								config.KafkaInstanceType{
									Id: "myinstancetype",
									SupportedBillingModels: []config.KafkaBillingModel{
										config.KafkaBillingModel{
											ID:              "mybillingmodel",
											GracePeriodDays: 2,
										},
									},
								},
							},
						},
					},
				},
				kafkaService: &services.KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
						return &dbapi.KafkaRequest{
							Status: constants.KafkaRequestStatusSuspended.String(),
							Meta: api.Meta{
								ID: "id",
							},
							InstanceType:            "myinstancetype",
							ClusterID:               "cluster-id",
							ActualKafkaIBPVersion:   "2.8",
							DesiredKafkaIBPVersion:  "2.8",
							ActualKafkaVersion:      "2.8",
							DesiredKafkaVersion:     "2.8",
							DesiredStrimziVersion:   "2.8",
							KafkaStorageSize:        "100",
							ActualKafkaBillingModel: "mybillingmodel",
							ExpiresAt:               sql.NullTime{Time: time.Now().Add(240 * time.Hour), Valid: true}, //expires 10 days from now
						}, nil
					},
					VerifyAndUpdateKafkaAdminFunc: func(ctx context.Context, kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				accountService: account.NewMockAccountService(),
			},
			args: args{
				url:  kafkaByIdUrl,
				body: []byte(`{"suspended": false}`),
			},
			wantStatusCode:  http.StatusOK,
			wantKafkaStatus: constants.KafkaRequestStatusResuming,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewAdminKafkaHandler(tt.fields.kafkaService, tt.fields.accountService, tt.fields.providerConfig, tt.fields.clusterService, tt.fields.kafkaConfig)
			req, rw := GetHandlerParams("PATCH", tt.args.url, bytes.NewBuffer(tt.args.body), t)
			h.Update(rw, req)
			resp := rw.Result()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			if tt.wantStatusCode == http.StatusOK {
				kafka := &private.Kafka{}
				err := json.NewDecoder(resp.Body).Decode(&kafka)
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(kafka.Status).To(gomega.Equal(tt.wantKafkaStatus.String()))
			}
			resp.Body.Close()
		})
	}
}
