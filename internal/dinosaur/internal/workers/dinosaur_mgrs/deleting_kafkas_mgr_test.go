package dinosaur_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
)

func TestDeletingDinosaurManager(t *testing.T) {
	type fields struct {
		dinosaurService services.DinosaurService
		quotaService    services.QuotaService
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
			name: "successful reconcile",
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					DeleteFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				},
				quotaService: &services.QuotaServiceMock{
					DeleteQuotaFunc: func(id string) *errors.ServiceError {
						return nil
					},
				},
			},
		},
		{
			name: "failed reconcile",
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					DeleteFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return errors.GeneralError("test")
					},
				},
				quotaService: &services.QuotaServiceMock{
					DeleteQuotaFunc: func(id string) *errors.ServiceError {
						return nil
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &DeletingDinosaurManager{
				dinosaurService: tt.fields.dinosaurService,
				quotaServiceFactory: &services.QuotaServiceFactoryMock{
					GetQuotaServiceFunc: func(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
						return tt.fields.quotaService, nil
					},
				},
			}
			if err := k.reconcileDeletingDinosaurs(tt.args.dinosaur); (err != nil) != tt.wantErr {
				t.Errorf("reconcileDeletingDinosaurs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
