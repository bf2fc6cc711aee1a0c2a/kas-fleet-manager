package dinosaur_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/dinosaurs/types"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
)

func TestAcceptedDinosaurManager(t *testing.T) {
	testConfig := config.NewDataplaneClusterConfig()
	testConfig.StrimziOperatorVersion = "strimzi-cluster-operator.v0.23.0-0"
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
		name       string
		fields     fields
		args       args
		wantErr    bool
		wantStatus string
	}{
		{
			name: "error when finding cluster fails",
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
			name: "error when dinosaur service update fails",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
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
			wantErr:    true,
			wantStatus: constants2.DinosaurRequestStatusPreparing.String(),
		},
		{
			name: "error when dataPlaneClusterConfig doesn't contain strimzi version",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
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
				dinosaur: &dbapi.DinosaurRequest{},
			},
			wantErr:    true,
			wantStatus: constants2.DinosaurRequestStatusFailed.String(),
		},
		{
			name: "successful reconcile",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(dinosaur *dbapi.DinosaurRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
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
				dataPlaneClusterConfig: testConfig,
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			wantStatus: constants2.DinosaurRequestStatusPreparing.String(),
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
		})
	}
}
