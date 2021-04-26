package workers

import (
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	constants "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
)

func TestKafkaManager_reconcilePreparingKafka(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}
	type args struct {
		kafka *api.KafkaRequest
	}
	tests := []struct {
		name                string
		fields              fields
		args                args
		wantErr             bool
		wantErrMsg          string
		expectedKafkaStatus constants.KafkaStatus
	}{
		{
			name: "Encounter a 5xx error Kafka preparation and performed the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("simulate 5xx error")
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(time.Minute * time.Duration(30)),
					},
					Status: string(constants.KafkaRequestStatusPreparing),
				},
			},
			wantErr:             true,
			wantErrMsg:          "",
			expectedKafkaStatus: constants.KafkaRequestStatusPreparing,
		},
		{
			name: "Encounter a 5xx error Kafka preparation and skipped the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.GeneralError("simulate 5xx error")
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(time.Minute * time.Duration(-30)),
					},
					Status: string(constants.KafkaRequestStatusPreparing),
				},
			},
			wantErr:             true,
			wantErrMsg:          "simulate 5xx error",
			expectedKafkaStatus: constants.KafkaRequestStatusFailed,
		},
		{
			name: "Encounter a Client error (4xx) in Kafka preparation",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.NotFound("simulate a 4xx error")
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{
					Status: string(constants.KafkaRequestStatusPreparing),
				},
			},
			wantErr:             true,
			wantErrMsg:          "simulate a 4xx error",
			expectedKafkaStatus: constants.KafkaRequestStatusFailed,
		},
		{
			name: "Encounter an SSO Client internal error in Kafka creation and performed the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.FailedToCreateSSOClient("ErrorFailedToCreateSSOClientReason")
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(30))}},
			},
			wantErr:    true,
			wantErrMsg: "",
		},
		{
			name: "Encounter an SSO Client internal error in Kafka creation and skipped the retry",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return errors.FailedToCreateSSOClient("ErrorFailedToCreateSSOClientReason")
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(-30))}},
			},
			wantErr:             true,
			wantErrMsg:          "ErrorFailedToCreateSSOClientReason",
			expectedKafkaStatus: constants.KafkaRequestStatusFailed,
		},
		{
			name: "Successful reconcile",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					PrepareKafkaRequestFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
					UpdateFunc: func(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				kafka: &api.KafkaRequest{},
			},
			wantErr:    false,
			wantErrMsg: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &KafkaManager{
				kafkaService: tt.fields.kafkaService,
			}

			if err := k.reconcilePreparingKafka(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcilePreparingKafka() error = %v, wantErr %v", err, tt.wantErr)
			}

			gomega.Expect(tt.expectedKafkaStatus.String()).Should(gomega.Equal(tt.args.kafka.Status))
			gomega.Expect(tt.args.kafka.FailedReason).Should(gomega.Equal(tt.wantErrMsg))
		})
	}
}

func TestKafkaManager_reconcileDeletedKafkas(t *testing.T) {
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
			if err := k.reconcileDeletedKafkas(tt.args.kafka); (err != nil) != tt.wantErr {
				t.Errorf("reconcileDeletedKafkas() error = %v, wantErr %v", err, tt.wantErr)
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
