package services

import (
	clusters2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters"
	types2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	config2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
)

func TestAgentOperatorAddon_Provision(t *testing.T) {
	addonId := "test-id"
	type fields struct {
		providerFactory clusters2.ProviderFactory
		ssoService      services.KeycloakService
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
				ssoService: &services.KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
				providerFactory: &clusters2.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters2.Provider, error) {
					return &clusters2.ProviderMock{
						InstallKasFleetshardFunc: func(clusterSpec *types2.ClusterSpec, params []types2.Parameter) (bool, error) {
							return false, nil
						},
					}, nil
				}},
			},
			// we can't change the state of AddOnInstallation to be ready as the field is private
			result:  false,
			wantErr: false,
		},
		{
			name: "provision is failed",
			fields: fields{
				ssoService: &services.KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, errors.GeneralError("error")
					},
				},
				providerFactory: &clusters2.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters2.Provider, error) {
					return &clusters2.ProviderMock{
						InstallKasFleetshardFunc: func(clusterSpec *types2.ClusterSpec, params []types2.Parameter) (bool, error) {
							return false, errors.GeneralError("error")
						},
					}, nil
				}},
			},
			result:  false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			agentOperatorAddon := &kasFleetshardOperatorAddon{
				SsoService:          tt.fields.ssoService,
				ProviderFactory:     tt.fields.providerFactory,
				ServerConfig:        &config.ServerConfig{},
				KasFleetShardConfig: &config2.KasFleetshardConfig{},
				OCMConfig:           &config.OCMConfig{KasFleetshardAddonID: addonId},
				KeycloakConfig: &config.KeycloakConfig{
					KafkaRealm: &config.KeycloakRealmConfig{},
				},
			}
			ready, err := agentOperatorAddon.Provision(api.Cluster{
				ClusterID:    "test-cluster-id",
				ProviderType: api.ClusterProviderOCM,
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
		ssoService services.KeycloakService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "receives error during removal of the service account fails when fleetshard operator is turned on",
			fields: fields{
				ssoService: &services.KeycloakServiceMock{
					DeRegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string) *errors.ServiceError {
						return &errors.ServiceError{} // an error is returned
					},
				},
			},
			wantErr: true,
		},
		{
			name: "succesful removes the service account when fleetshard operator is turned on",
			fields: fields{
				ssoService: &services.KeycloakServiceMock{
					DeRegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string) *errors.ServiceError {
						return nil
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			agentOperatorAddon := &kasFleetshardOperatorAddon{
				SsoService: tt.fields.ssoService,
			}
			err := agentOperatorAddon.RemoveServiceAccount(api.Cluster{
				ClusterID:    "test-cluster-id",
				ProviderType: api.ClusterProviderOCM,
			})
			gomega.Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestKasFleetshardOperatorAddon_ReconcileParameters(t *testing.T) {
	type fields struct {
		providerFactory clusters2.ProviderFactory
		ssoService      services.KeycloakService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "ReconcileParameters is finished successfully",
			fields: fields{
				ssoService: &services.KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
				providerFactory: &clusters2.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters2.Provider, error) {
					return &clusters2.ProviderMock{
						InstallKasFleetshardFunc: func(clusterSpec *types2.ClusterSpec, params []types2.Parameter) (bool, error) {
							return true, nil
						},
					}, nil
				}},
			},
			wantErr: false,
		},
		{
			name: "ReconcileParameters is failed because UpdateAddonParameters failed",
			fields: fields{
				ssoService: &services.KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
				providerFactory: &clusters2.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters2.Provider, error) {
					return &clusters2.ProviderMock{
						InstallKasFleetshardFunc: func(clusterSpec *types2.ClusterSpec, params []types2.Parameter) (bool, error) {
							return false, errors.GeneralError("test error")
						},
					}, nil
				}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			agentOperatorAddon := &kasFleetshardOperatorAddon{
				SsoService:          tt.fields.ssoService,
				ProviderFactory:     tt.fields.providerFactory,
				ServerConfig:        &config.ServerConfig{},
				KasFleetShardConfig: &config2.KasFleetshardConfig{},
				OCMConfig:           &config.OCMConfig{KasFleetshardAddonID: "kas-fleetshard"},
				KeycloakConfig: &config.KeycloakConfig{
					KafkaRealm: &config.KeycloakRealmConfig{},
				},
			}
			err := agentOperatorAddon.ReconcileParameters(api.Cluster{
				ClusterID:    "test-cluster-id",
				ProviderType: api.ClusterProviderOCM,
			})
			if err != nil && !tt.wantErr {
				t.Errorf("Provision() error = %v, want = %v", err, tt.wantErr)
			}
		})
	}
}
