package dinosaur_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/acl"
	"testing"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
)

func TestDinosaurManager_reconcileDeniedDinosaurOwners(t *testing.T) {
	type fields struct {
		dinosaurService services.DinosaurService
	}
	type args struct {
		deniedAccounts acl.DeniedUsers
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
				dinosaurService: &services.DinosaurServiceMock{
					DeprovisionDinosaurForUsersFunc: nil, // set to nil as it should not be called
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{},
			},
			wantErr: false,
		},
		{
			name: "should receive error when update in deprovisioning in database returns an error",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					DeprovisionDinosaurForUsersFunc: func(users []string) *errors.ServiceError {
						return &errors.ServiceError{}
					},
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{"some user"},
			},
			wantErr: true,
		},
		{
			name: "should not receive error when update in deprovisioning in database succeed",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					DeprovisionDinosaurForUsersFunc: func(users []string) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{"some user"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &DinosaurManager{
				dinosaurService: tt.fields.dinosaurService,
			}
			err := k.reconcileDeniedDinosaurOwners(tt.args.deniedAccounts)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
