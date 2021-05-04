package workers

import (
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	constants "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

func TestKafkaManager_reconcileProvisionedKafka(t *testing.T) {
	type fields struct {
		ocmClient            ocm.Client
		kafkaService         services.KafkaService
		timer                *time.Timer
		keycloakService      services.KeycloakService
		observatoriumService services.ObservatoriumService
		configService        services.ConfigService
	}
	type args struct {
		kafka *api.KafkaRequest
	}
	tests := []struct {
		name                string
		fields              fields
		args                args
		wantErr             bool
		expectedKafkaStatus constants.KafkaStatus
	}{
		{
			name: "kafka request marked as failed when kafka creation fails with 5XX and maximum time limit has been reached",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					IsKafkaClientExistFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return config.NewKeycloakConfig()
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(-(constants.KafkaMaxDurationWithProvisioningErrs + 1)),
					},
				},
			},
			wantErr:             true,
			expectedKafkaStatus: constants.KafkaRequestStatusFailed,
		},
		{
			name: "kafka request marked as failed when kafka creation fails with 4XX error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.BadRequest("test badrequest")
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					IsKafkaClientExistFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return config.NewKeycloakConfig()
					},
				},
				observatoriumService: &services.ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{}, nil
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr:             true,
			expectedKafkaStatus: constants.KafkaRequestStatusFailed,
		},
		{
			name: "kafka creation returns error without marking kafka request as failed when kafka creation fails with 5XX error and no time limit is reached",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					IsKafkaClientExistFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
					GetConfigFunc: func() *config.KeycloakConfig {
						return config.NewKeycloakConfig()
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now(),
					},
					Status: string(constants.KafkaRequestStatusPreparing),
				},
			},
			wantErr:             true,
			expectedKafkaStatus: constants.KafkaRequestStatusPreparing,
		},
		{
			name: "successful reconcile",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return false, nil
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
				observatoriumService: &services.ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{}, nil
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					GetConfigFunc: func() *config.KeycloakConfig {
						return config.NewKeycloakConfig()
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &KafkaManager{
				ocmClient:            tt.fields.ocmClient,
				kafkaService:         tt.fields.kafkaService,
				timer:                tt.fields.timer,
				keycloakService:      tt.fields.keycloakService,
				observatoriumService: tt.fields.observatoriumService,
				configService:        tt.fields.configService,
			}
			if err := k.reconcilePreparedKafka(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcilePreparedKafka() error = %v, wantErr %v", err, tt.wantErr)
			}
			if string(tt.expectedKafkaStatus) != tt.args.kafka.Status {
				t.Errorf("reconcilePreparedKafka() kafka status = %v, expectedKafkaStatus :%v", tt.args.kafka.Status, tt.expectedKafkaStatus)
			}
		})
	}
}

func TestKafkaManager_reconcileDeprovisioningRequest(t *testing.T) {
	type fields struct {
		ocmClient            ocm.Client
		kafkaService         services.KafkaService
		timer                *time.Timer
		keycloakService      services.KeycloakService
		observatoriumService services.ObservatoriumService
		quotaService         services.QuotaService
		configService        services.ConfigService
	}
	type args struct {
		kafka *api.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "successful reconcile",
			args: args{
				kafka: &api.KafkaRequest{},
			},
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeleteFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						//return errors.GeneralError("test")
						return nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					DeleteQuotaFunc: func(id string) *errors.ServiceError {
						return nil
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
		},
		{
			name: "failed reconcile",
			args: args{
				kafka: &api.KafkaRequest{},
			},
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeleteFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
				quotaService: &services.QuotaServiceMock{
					DeleteQuotaFunc: func(id string) *errors.ServiceError {
						return nil
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &KafkaManager{
				ocmClient:            tt.fields.ocmClient,
				kafkaService:         tt.fields.kafkaService,
				timer:                tt.fields.timer,
				keycloakService:      tt.fields.keycloakService,
				observatoriumService: tt.fields.observatoriumService,
				quotaService:         tt.fields.quotaService,
				configService:        tt.fields.configService,
			}
			if err := k.reconcileDeprovisioningRequest(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcileDeprovisioningRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKafkaManager_reconcileAcceptedKafka(t *testing.T) {
	type fields struct {
		ocmClient            ocm.Client
		kafkaService         services.KafkaService
		timer                *time.Timer
		keycloakService      services.KeycloakService
		observatoriumService services.ObservatoriumService
		configService        services.ConfigService
		clusterPlmtStrategy  services.ClusterPlacementStrategy
	}
	type args struct {
		kafka *api.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "error when finding cluster fails",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *api.KafkaRequest) (*api.Cluster, error) {
						return nil, errors.GeneralError("test")
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					GetConfigFunc: func() *config.KeycloakConfig {
						return config.NewKeycloakConfig()
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "error when kafka service update fails",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *api.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					GetConfigFunc: func() *config.KeycloakConfig {
						return config.NewKeycloakConfig()
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "successful reconcile",
			fields: fields{
				clusterPlmtStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *api.KafkaRequest) (*api.Cluster, error) {
						return &api.Cluster{}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					GetConfigFunc: func() *config.KeycloakConfig {
						return config.NewKeycloakConfig()
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &KafkaManager{
				ocmClient:            tt.fields.ocmClient,
				kafkaService:         tt.fields.kafkaService,
				timer:                tt.fields.timer,
				keycloakService:      tt.fields.keycloakService,
				observatoriumService: tt.fields.observatoriumService,
				configService:        tt.fields.configService,
				clusterPlmtStrategy:  tt.fields.clusterPlmtStrategy,
			}
			if err := k.reconcileAcceptedKafka(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcileAcceptedKafka() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKafkaManager_reconcileProvisioningKafka(t *testing.T) {
	type fields struct {
		ocmClient            ocm.Client
		kafkaService         services.KafkaService
		timer                *time.Timer
		keycloakService      services.KeycloakService
		observatoriumService services.ObservatoriumService
		configService        services.ConfigService
	}
	type args struct {
		kafka *api.KafkaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "error when creating kafka fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					IsKafkaClientExistFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
				},
				observatoriumService: &services.ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{}, errors.NotFound("Not Found")
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "error when updating kafka status fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return false, errors.GeneralError("test")
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
				keycloakService: &services.KeycloakServiceMock{
					IsKafkaClientExistFunc: func(clientId string) *errors.ServiceError {
						return nil
					},
				},
				observatoriumService: &services.ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{}, errors.NotFound("Not Found")
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "error when kafka resource_creating status does not exist in the DB before creation",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
						return false, nil
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, errors.NotFound("Not Found")
					},
				},
				observatoriumService: &services.ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{}, errors.NotFound("Not Found")
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr: true,
		},
		{
			name: "successful reconcile",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					CreateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					GetByIdFunc: func(id string) (*api.KafkaRequest, *errors.ServiceError) {
						return &api.KafkaRequest{}, nil
					},
				},
				observatoriumService: &services.ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{State: "ready"}, nil
					},
				},
				configService: services.NewConfigService(config.ApplicationConfig{
					Kafka: config.NewKafkaConfig(),
				}),
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &KafkaManager{
				ocmClient:            tt.fields.ocmClient,
				kafkaService:         tt.fields.kafkaService,
				timer:                tt.fields.timer,
				keycloakService:      tt.fields.keycloakService,
				observatoriumService: tt.fields.observatoriumService,
				configService:        tt.fields.configService,
			}
			if err := k.reconcileProvisioningKafka(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcileProvisioningKafka() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKafkaManager_reconcileDeniedKafkaOwners(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}
	type args struct {
		deniedAccounts config.DeniedUsers
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "do not reconcile when denied accounts list is empty",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil, // set to nil as it should not be called
				},
			},
			args: args{
				deniedAccounts: config.DeniedUsers{},
			},
			wantErr: false,
		},
		{
			name: "should receive error when update in deprovisioning in database returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: func(users []string) *errors.ServiceError {
						return &errors.ServiceError{}
					},
				},
			},
			args: args{
				deniedAccounts: config.DeniedUsers{"some user"},
			},
			wantErr: true,
		},
		{
			name: "should not receive error when update in deprovisioning in database succeed",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: func(users []string) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				deniedAccounts: config.DeniedUsers{"some user"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &KafkaManager{
				kafkaService: tt.fields.kafkaService,
			}
			err := k.reconcileDeniedKafkaOwners(tt.args.deniedAccounts)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
