package kafka_mgrs

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	mockClusters "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/clusters"
	mockKafkas "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	svcErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	w "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/onsi/gomega"
)

func Test_ProvisioningKafkaManager_Reconcile(t *testing.T) {
	type fields struct {
		kafkaService             services.KafkaService
		clusterPlacementStrategy services.ClusterPlacementStrategy
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should throw an error if listing kafkas fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants.KafkaStatus) ([]*dbapi.KafkaRequest, *svcErrors.ServiceError) {
						return nil, svcErrors.GeneralError("failed to list kafka requests")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should throw an error when for reassigning kafka returns an error",
			fields: fields{
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{
							ClusterID:                "",
							AvailableStrimziVersions: mockClusters.AvailableStrimziVersions,
						}, fmt.Errorf("some error")
					},
				},
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants.KafkaStatus) ([]*dbapi.KafkaRequest, *svcErrors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
								kafkaRequest.ClusterID = ""
								kafkaRequest.Status = constants.KafkaRequestStatusProvisioning.String()
							}),
						}, nil
					},
					AssignBootstrapServerHostFunc: func(kafkaRequest *dbapi.KafkaRequest) error {
						return svcErrors.GeneralError("test")
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *svcErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should not throw an error if listing kafkas returns an empty list",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants.KafkaStatus) ([]*dbapi.KafkaRequest, *svcErrors.ServiceError) {
						return []*dbapi.KafkaRequest{}, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Should not throw an error when reconciling the provisioning kafka is successful",
			fields: fields{
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{
							ClusterID:                "",
							AvailableStrimziVersions: mockClusters.AvailableStrimziVersions,
						}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					ListByStatusFunc: func(status ...constants.KafkaStatus) ([]*dbapi.KafkaRequest, *svcErrors.ServiceError) {
						return []*dbapi.KafkaRequest{
							mockKafkas.BuildKafkaRequest(func(kafkaRequest *dbapi.KafkaRequest) {
								kafkaRequest.ClusterID = ""
								kafkaRequest.Status = constants.KafkaRequestStatusProvisioning.String()
							}),
						}, nil
					},
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *svcErrors.ServiceError {
						return nil
					},
					AssignBootstrapServerHostFunc: func(kafkaRequest *dbapi.KafkaRequest) error {
						return nil
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			k := &ProvisioningKafkaManager{
				kafkaService:             tt.fields.kafkaService,
				clusterPlacementStrategy: tt.fields.clusterPlacementStrategy,
			}
			g.Expect(len(NewProvisioningKafkaManager(tt.fields.kafkaService, w.Reconciler{}, tt.fields.clusterPlacementStrategy).Reconcile()) > 0).To(gomega.Equal(tt.wantErr))

			got := k.Reconcile()
			g.Expect(got != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_ProvisioningKafkaManager_reassignProvisioningKafka(t *testing.T) {
	strimziOperatorVersion := "strimzi-cluster-operator.from-cluster"
	allStrimziVersions, err := json.Marshal([]api.StrimziVersion{
		{
			Version: strimziOperatorVersion,
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{
					Version: "2.7.0",
				},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{
					Version: "2.7",
				},
			},
		},
	})
	if err != nil {
		t.Fatal("failed to convert available strimzi versions to json")
	}

	versionsWOkafkaVersion, err := json.Marshal([]api.StrimziVersion{
		{
			Version:       strimziOperatorVersion,
			Ready:         true,
			KafkaVersions: nil,
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{
					Version: "2.7",
				},
			},
		},
	})
	if err != nil {
		t.Fatal("failed to convert available strimzi versions to json")
	}
	versionsWOIBPVersion, err := json.Marshal([]api.StrimziVersion{
		{
			Version: strimziOperatorVersion,
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{
					Version: "2.7.0",
				},
			},
			KafkaIBPVersions: nil,
		},
	})
	if err != nil {
		t.Fatal("failed to convert available strimzi versions to json")
	}

	produceErrorForStrimziVersion := append(allStrimziVersions[:1], allStrimziVersions[7:]...)

	type fields struct {
		kafkaService             services.KafkaService
		clusterPlacementStrategy *services.ClusterPlacementStrategyMock
	}
	type args struct {
		kafka *dbapi.KafkaRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "should return error if no available cluster can be found",
			fields: fields{
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					ClusterID:           "",
					BootstrapServerHost: "",
				},
			},
			want: true,
		},
		{
			name: "should return error if ClusterPlacementStrategy returns error",
			fields: fields{
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return nil, errors.New("test")
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					ClusterID:           "",
					BootstrapServerHost: "",
				},
			},
			want: true,
		},
		{
			name: "should return error if there is an error finding new kafka version",
			fields: fields{
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{
							ClusterID:                "test-id",
							AvailableStrimziVersions: versionsWOkafkaVersion,
						}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					AssignBootstrapServerHostFunc: func(kafkaRequest *dbapi.KafkaRequest) error {
						return nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					ClusterID:           "",
					BootstrapServerHost: "",
				},
			},
			want: true,
		},
		{
			name: "should return error if there is an error finding new strimzi versions",
			fields: fields{
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{
							ClusterID:                "test-id",
							AvailableStrimziVersions: produceErrorForStrimziVersion,
						}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					AssignBootstrapServerHostFunc: func(kafkaRequest *dbapi.KafkaRequest) error {
						return nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					ClusterID:           "",
					BootstrapServerHost: "",
				},
			},
			want: true,
		},
		{
			name: "should return an error if available strimzi versions is nil",
			fields: fields{
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{
							ClusterID:                "test-id",
							AvailableStrimziVersions: nil,
						}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *svcErrors.ServiceError {
						return nil
					},
					AssignBootstrapServerHostFunc: func(kafkaRequest *dbapi.KafkaRequest) error {
						return nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					ClusterID:           "",
					BootstrapServerHost: "",
				},
			},
			want: true,
		},
		{
			name: "should return error if there is an error finding new kafka IBP version",
			fields: fields{
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{
							ClusterID:                "test-id",
							AvailableStrimziVersions: versionsWOIBPVersion,
						}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					AssignBootstrapServerHostFunc: func(kafkaRequest *dbapi.KafkaRequest) error {
						return nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					ClusterID:           "",
					BootstrapServerHost: "",
				},
			},
			want: true,
		},
		{
			name: "should return an error if kafka values fail to update.",
			fields: fields{
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{
							ClusterID:                "test-id",
							AvailableStrimziVersions: mockClusters.AvailableStrimziVersions,
						}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *svcErrors.ServiceError {
						return svcErrors.GeneralError("test")
					},
					AssignBootstrapServerHostFunc: func(kafkaRequest *dbapi.KafkaRequest) error {
						return nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					ClusterID:           "",
					BootstrapServerHost: "",
				},
			},
			want: true,
		},
		{
			name: "should successfully assign new field values to kafka",
			fields: fields{
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{
							ClusterID:                "test-id",
							AvailableStrimziVersions: mockClusters.AvailableStrimziVersions,
						}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *svcErrors.ServiceError {
						return nil
					},
					AssignBootstrapServerHostFunc: func(kafkaRequest *dbapi.KafkaRequest) error {
						return nil
					},
				},
			},
			args: args{
				kafka: &dbapi.KafkaRequest{
					ClusterID:           "",
					BootstrapServerHost: "",
				},
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			k := &ProvisioningKafkaManager{
				kafkaService:             test.fields.kafkaService,
				clusterPlacementStrategy: test.fields.clusterPlacementStrategy,
			}
			err := k.reassignProvisioningKafka(test.args.kafka)
			g.Expect(err != nil).To(gomega.Equal(test.want))
		})
	}
}
