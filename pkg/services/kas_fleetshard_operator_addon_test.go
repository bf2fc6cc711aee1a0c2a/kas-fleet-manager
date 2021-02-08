package services

import (
	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
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
			name: "provision is ready",
			fields: fields{
				ssoService: &KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return &api.ServiceAccount{}, nil
					},
				},
				ocm: &ocm.ClientMock{},
			},
			result:  true,
			wantErr: false,
		},
		{
			name: "provision is not ready",
			fields: fields{
				ssoService: &KeycloakServiceMock{
					RegisterKasFleetshardOperatorServiceAccountFunc: func(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, nil
					},
				},
				ocm: &ocm.ClientMock{},
			},
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
				ocm: &ocm.ClientMock{},
			},
			result:  false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			agentOperatorAddon := &agentOperatorAddon{
				ssoService: tt.fields.ssoService,
				ocm:        tt.fields.ocm,
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
