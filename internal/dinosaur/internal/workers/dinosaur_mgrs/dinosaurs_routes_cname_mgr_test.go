package dinosaur_mgrs

import (
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"testing"
)

func TestDinosaurRoutesCNAMEManager(t *testing.T) {
	testChangeID := "1234"
	testChangeINSYNC := route53.ChangeStatusInsync

	type fields struct {
		dinosaurService services.DinosaurService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should success when there are no errors",
			fields: fields{dinosaurService: &services.DinosaurServiceMock{
				ListDinosaursWithRoutesNotCreatedFunc: func() ([]*dbapi.DinosaurRequest, *errors.ServiceError) {
					dinosaur := &dbapi.DinosaurRequest{
						Name:          "test",
						RoutesCreated: false,
					}
					return []*dbapi.DinosaurRequest{
						dinosaur,
					}, nil
				},
				ChangeDinosaurCNAMErecordsFunc: func(dinosaurRequest *dbapi.DinosaurRequest, action services.DinosaurRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
					return &route53.ChangeResourceRecordSetsOutput{
						ChangeInfo: &route53.ChangeInfo{
							Id:     &testChangeID,
							Status: &testChangeINSYNC,
						},
					}, nil
				},
				UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
					if !dinosaurRequest.RoutesCreated {
						return errors.GeneralError("RoutesCreated is set to true")
					}
					return nil
				},
			}},
			wantErr: false,
		},
		{
			name: "should return error when list dinosaurs failed",
			fields: fields{dinosaurService: &services.DinosaurServiceMock{
				ListDinosaursWithRoutesNotCreatedFunc: func() ([]*dbapi.DinosaurRequest, *errors.ServiceError) {
					return nil, errors.GeneralError("failed to list dinosaurs")
				},
			}},
			wantErr: true,
		},
		{
			name: "should return error when creating CNAME failed",
			fields: fields{dinosaurService: &services.DinosaurServiceMock{
				ListDinosaursWithRoutesNotCreatedFunc: func() ([]*dbapi.DinosaurRequest, *errors.ServiceError) {
					dinosaur := &dbapi.DinosaurRequest{
						Name:          "test",
						RoutesCreated: false,
					}
					return []*dbapi.DinosaurRequest{
						dinosaur,
					}, nil
				},
				ChangeDinosaurCNAMErecordsFunc: func(dinosaurRequest *dbapi.DinosaurRequest, action services.DinosaurRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
					return nil, errors.GeneralError("failed to create CNAME")
				},
			}},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k := &DinosaurRoutesCNAMEManager{
				dinosaurService: test.fields.dinosaurService,
				dinosaurConfig:  &config.DinosaurConfig{EnableDinosaurExternalCertificate: true},
			}

			errs := k.Reconcile()
			if len(errs) > 0 && !test.wantErr {
				t.Errorf("unexpected error when reconcile dinosaur routes: %v", errs)
			}
		})
	}
}
