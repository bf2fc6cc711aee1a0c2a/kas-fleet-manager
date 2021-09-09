package dinosaur_mgrs

import (
	"encoding/json"
	"testing"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/dinosaurs/types"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"

	"github.com/onsi/gomega"
)

func TestAcceptedDinosaurManager(t *testing.T) {
	testConfig := config.NewDataplaneClusterConfig()

	strimziOperatorVersion := "strimzi-cluster-operator.from-cluster"
	availableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
		{
			Version: strimziOperatorVersion,
			Ready:   true,
			DinosaurVersions: []api.DinosaurVersion{
				api.DinosaurVersion{
					Version: "2.7.0",
				},
				api.DinosaurVersion{
					Version: "2.8.0",
				},
			},
			DinosaurIBPVersions: []api.DinosaurIBPVersion{
				api.DinosaurIBPVersion{
					Version: "2.7",
				},
				api.DinosaurIBPVersion{
					Version: "2.8",
				},
			},
		},
	})
	if err != nil {
		t.Fatal("failed to convert available strimzi versions to json")
	}

	noAvailableStrimziVersions, err := json.Marshal([]api.StrimziVersion{
		{
			Version: strimziOperatorVersion,
			Ready:   false,
			DinosaurVersions: []api.DinosaurVersion{
				api.DinosaurVersion{
					Version: "2.7.0",
				},
				api.DinosaurVersion{
					Version: "2.8.0",
				},
			},
			DinosaurIBPVersions: []api.DinosaurIBPVersion{
				api.DinosaurIBPVersion{
					Version: "2.7",
				},
				api.DinosaurIBPVersion{
					Version: "2.8",
				},
			},
		},
	})
	if err != nil {
		t.Fatal("failed to convert available strimzi versions to json")
	}

	mockCluster := &api.Cluster{
		Meta: api.Meta{
			ID:        "id",
			CreatedAt: time.Now(),
		},
		ClusterID:                "cluster-id",
		MultiAZ:                  true,
		Region:                   "us-east-1",
		Status:                   "ready",
		AvailableStrimziVersions: availableStrimziVersions,
	}

	mockClusterWithoutAvailableStrimziVersion := *mockCluster
	mockClusterWithoutAvailableStrimziVersion.AvailableStrimziVersions = noAvailableStrimziVersions

	type fields struct {
		dinosaurService        services.DinosaurService
		clusterPlmtStrategy    services.ClusterPlacementStrategy
		quotaService           services.QuotaService
		dataPlaneClusterConfig *config.DataplaneClusterConfig
	}
	type args struct {
		dinosaur *dbapi.DinosaurRequest
	}
	tests := []struct {
		name                       string
		fields                     fields
		args                       args
		wantErr                    bool
		wantStatus                 string
		wantStrimziOperatorVersion string
	}{
		{
			name: "should return an error when finding cluster fails",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
						return nil, errors.GeneralError("test")
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
						return "", nil
					},
				},
				dataPlaneClusterConfig: testConfig,
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			wantErr: true,
		},
		{
			name: "should not return an error if no available cluster is found",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
						return "", nil
					},
				},
				dataPlaneClusterConfig: testConfig,
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			wantErr: false,
		},
		{
			name: "should return an error when dinosaur service update fails",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
						return mockCluster, nil
					},
				},
				dinosaurService: &services.DinosaurServiceMock{
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
						return "some-subscription", nil
					},
				},
				dataPlaneClusterConfig: testConfig,
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			wantErr:                    true,
			wantStatus:                 constants2.DinosaurRequestStatusPreparing.String(),
			wantStrimziOperatorVersion: strimziOperatorVersion,
		},
		{
			name: "should get desired strimzi version from cluster if the StrimziOperatorVersion is not set in the data plane config",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
						return mockCluster, nil
					},
				},
				dinosaurService: &services.DinosaurServiceMock{
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{}, nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
						return "sub-scription", nil
					},
				},
				dataPlaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{
					Meta: api.Meta{
						CreatedAt: time.Now(),
					},
				},
			},
			wantErr:                    false,
			wantStatus:                 constants2.DinosaurRequestStatusPreparing.String(),
			wantStrimziOperatorVersion: strimziOperatorVersion,
		},
		{
			name: "should keep dinosaur status as accepted if no strimzi operator version is available when retry period has not expired",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
						return &mockClusterWithoutAvailableStrimziVersion, nil
					},
				},
				dinosaurService: &services.DinosaurServiceMock{
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{}, nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
						return "sub-scription", nil
					},
				},
				dataPlaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{
					Meta: api.Meta{
						CreatedAt: time.Now(),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should set dinosaur status to failed if no strimzi operator version is available after retry period has expired",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
						return &mockClusterWithoutAvailableStrimziVersion, nil
					},
				},
				dinosaurService: &services.DinosaurServiceMock{
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
					GetByIdFunc: func(id string) (*dbapi.DinosaurRequest, *errors.ServiceError) {
						return &dbapi.DinosaurRequest{}, nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					ReserveQuotaFunc: func(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
						return "sub-scription", nil
					},
				},
				dataPlaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(time.Duration(-constants2.AcceptedDinosaurMaxRetryDuration)),
					},
				},
			},
			wantErr:    true,
			wantStatus: constants2.DinosaurRequestStatusFailed.String(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &AcceptedDinosaurManager{
				dinosaurService:     tt.fields.dinosaurService,
				clusterPlmtStrategy: tt.fields.clusterPlmtStrategy,
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
				dataPlaneClusterConfig: tt.fields.dataPlaneClusterConfig,
			}
			err := k.reconcileAcceptedDinosaur(tt.args.dinosaur)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			gomega.Expect(tt.args.dinosaur.Status).To(gomega.Equal(tt.wantStatus))
			gomega.Expect(tt.args.dinosaur.DesiredStrimziVersion).To(gomega.Equal(tt.wantStrimziOperatorVersion))
		})
	}
}
