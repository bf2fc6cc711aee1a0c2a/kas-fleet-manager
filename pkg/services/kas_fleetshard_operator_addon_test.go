package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"testing"
)

func TestAgentOperatorAddon_Provision(t *testing.T) {
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
						return &clustersmgmtv1.AddOnInstallation{}, nil
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
					Server:                     &config.ServerConfig{},
					Keycloak:                   &config.KeycloakConfig{},
					ObservabilityConfiguration: &config.ObservabilityConfiguration{},
					ClusterCreationConfig:      &config.ClusterCreationConfig{},
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
