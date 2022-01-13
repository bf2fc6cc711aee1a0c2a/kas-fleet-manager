package dinosaur_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
)

func TestReadyDinosaurManager_reconcileCanaryServiceAccount(t *testing.T) {
	serviceAccount := &api.ServiceAccount{
		ClientID:     "some-client-id",
		ClientSecret: "some-client-secret",
	}

	type fields struct {
		dinosaurService    services.DinosaurService
		keycloakService coreServices.KeycloakService
	}
	type args struct {
		dinosaur *dbapi.DinosaurRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Successful reconcile dinosaur that have canary service account already created ",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					UpdateFunc: nil, // set to nil as it should not be called
				},
				keycloakService: &coreServices.KeycloakServiceMock{
					CreateServiceAccountInternalFunc: nil, // set to nil as it should not be called,
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{
					CanaryServiceAccountClientID:     "some-client-id",
					CanaryServiceAccountClientSecret: "some-client-secret",
				},
			},
			wantErr: false,
		},
		{
			name: "returns an error when service account creation fails",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				},
				keycloakService: &coreServices.KeycloakServiceMock{
					CreateServiceAccountInternalFunc: func(request coreServices.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return nil, &errors.ServiceError{}
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			wantErr: true,
		},
		{
			name: "Successful reconcile",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				},
				keycloakService: &coreServices.KeycloakServiceMock{
					CreateServiceAccountInternalFunc: func(request coreServices.CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
						return serviceAccount, nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &ReadyDinosaurManager{
				dinosaurService:    tt.fields.dinosaurService,
				keycloakService: tt.fields.keycloakService,
			}

			if err := k.reconcileCanaryServiceAccount(tt.args.dinosaur); (err != nil) != tt.wantErr {
				t.Errorf("reconcilePreparingDinosaur() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				gomega.Expect(tt.args.dinosaur.CanaryServiceAccountClientID).NotTo(gomega.BeEmpty())
				gomega.Expect(tt.args.dinosaur.CanaryServiceAccountClientSecret).NotTo(gomega.BeEmpty())
			}
		})
	}
}
