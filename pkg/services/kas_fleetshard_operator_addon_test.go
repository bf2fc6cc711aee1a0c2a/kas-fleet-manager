package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"testing"
)

func TestAgentOperatorAddon_Provision(t *testing.T) {
	addon, _ := clustersmgmtv1.NewAddOnInstallation().ID("test-id").Build()
	type fields struct {
		ocm        ocm.Client
		ssoService KeycloakService
	}
	tests := []struct {
		name    string
		fields  fields
		result  bool
		wantErr bool
	}{
		{
			name: "provision is finished successfully",
			fields: fields{
				ssoService: &KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
				ocm: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return addon, nil
					},
					CreateAddonWithParamsFunc: func(clusterId string, addonId string, parameters []ocm.AddonParameter) (*clustersmgmtv1.AddOnInstallation, error) {
						return &clustersmgmtv1.AddOnInstallation{}, nil
					},
				},
			},
			// we can't change the state of AddOnInstallation to be ready as the field is private
			result:  false,
			wantErr: false,
		},
		{
			name: "provision is failed",
			fields: fields{
				ssoService: &KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, errors.GeneralError("error")
					},
				},
				ocm: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return nil, nil
					},
				},
			},
			result:  false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			agentOperatorAddon := &kasFleetshardOperatorAddon{
				ssoService: tt.fields.ssoService,
				ocm:        tt.fields.ocm,
				configService: NewConfigService(config.ApplicationConfig{
					Server: &config.ServerConfig{},
					Keycloak: &config.KeycloakConfig{
						KafkaRealm: &config.KeycloakRealmConfig{},
					},
					ClusterCreationConfig:      &config.ClusterCreationConfig{},
					KasFleetShardConfig: &config.KasFleetshardConfig{},
				}),
			}
			ready, err := agentOperatorAddon.Provision(api.Cluster{
				ClusterID: "test-cluster-id",
			})
			if err != nil && !tt.wantErr {
				t.Errorf("Provision() error = %v, want = %v", err, tt.wantErr)
			}
			Expect(ready).To(Equal(tt.result))
		})
	}
}

func TestAgentOperatorAddon_RemoveServiceAccount(t *testing.T) {
	type fields struct {
		ssoService                KeycloakService
		fleetshardOperatorEnabled bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "does not error when fleetshard operator feature is turned off",
			fields: fields{
				ssoService: &KeycloakServiceMock{
					DeRegisterKasFleetshardOperatorServiceAccountFunc: nil, // set to nil as it not called
				},
				fleetshardOperatorEnabled: false,
			},
			wantErr: false,
		},
		{
			name: "receives error during removal of the service account fails when fleetshard operator is turned on",
			fields: fields{
				ssoService: &KeycloakServiceMock{
					DeRegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string) *errors.ServiceError {
						return &errors.ServiceError{} // an error is returned
					},
				},
				fleetshardOperatorEnabled: true,
			},
			wantErr: true,
		},
		{
			name: "succesful removes the service account when fleetshard operator is turned on",
			fields: fields{
				ssoService: &KeycloakServiceMock{
					DeRegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string) *errors.ServiceError {
						return nil
					},
				},
				fleetshardOperatorEnabled: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			agentOperatorAddon := &kasFleetshardOperatorAddon{
				ssoService: tt.fields.ssoService,
				configService: NewConfigService(config.ApplicationConfig{
					Kafka: &config.KafkaConfig{
						EnableKasFleetshardSync: tt.fields.fleetshardOperatorEnabled,
						EnableManagedKafkaCR:    tt.fields.fleetshardOperatorEnabled,
					},
				}),
			}
			err := agentOperatorAddon.RemoveServiceAccount(api.Cluster{
				ClusterID: "test-cluster-id",
			})
			gomega.Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestKasFleetshardOperatorAddon_ReconcileParameters(t *testing.T) {
	addon, _ := clustersmgmtv1.NewAddOnInstallation().ID("test-id").Build()
	type fields struct {
		ocm        ocm.Client
		ssoService KeycloakService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "ReconcileParameters is finished successfully",
			fields: fields{
				ssoService: &KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
				ocm: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return addon, nil
					},
					UpdateAddonParametersFunc: func(clusterId string, addonId string, parameters []ocm.AddonParameter) (*clustersmgmtv1.AddOnInstallation, error) {
						return &clustersmgmtv1.AddOnInstallation{}, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ReconcileParameters is failed because addon id is not valid",
			fields: fields{
				ssoService: &KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
				ocm: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return nil, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "ReconcileParameters is failed because UpdateAddonParameters failed",
			fields: fields{
				ssoService: &KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
				ocm: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return addon, nil
					},
					UpdateAddonParametersFunc: func(clusterId string, addonId string, parameters []ocm.AddonParameter) (*clustersmgmtv1.AddOnInstallation, error) {
						return nil, errors.GeneralError("test error")
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			agentOperatorAddon := &kasFleetshardOperatorAddon{
				ssoService: tt.fields.ssoService,
				ocm:        tt.fields.ocm,
				configService: NewConfigService(config.ApplicationConfig{
					Server: &config.ServerConfig{},
					Keycloak: &config.KeycloakConfig{
						KafkaRealm: &config.KeycloakRealmConfig{},
					},
					ObservabilityConfiguration: &config.ObservabilityConfiguration{},
					ClusterCreationConfig:      &config.ClusterCreationConfig{},
					KasFleetShardConfig: &config.KasFleetshardConfig{},
				}),
			}
			err := agentOperatorAddon.ReconcileParameters(api.Cluster{
				ClusterID: "test-cluster-id",
			})
			if err != nil && !tt.wantErr {
				t.Errorf("Provision() error = %v, want = %v", err, tt.wantErr)
			}
		})
	}
}
