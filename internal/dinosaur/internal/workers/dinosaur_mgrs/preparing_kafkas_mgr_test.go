package dinosaur_mgrs

import (
	"testing"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/services"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestPreparingDinosaurManager(t *testing.T) {
	type fields struct {
		dinosaurService services.DinosaurService
	}
	type args struct {
		dinosaur *dbapi.DinosaurRequest
	}
	tests := []struct {
		name                   string
		fields                 fields
		args                   args
		wantErr                bool
		wantErrMsg             string
		expectedDinosaurStatus constants2.DinosaurStatus
	}{
		{
			name: "Encounter a 5xx error Dinosaur preparation and performed the retry",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					PrepareDinosaurRequestFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return errors.GeneralError("simulate 5xx error")
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(time.Minute * time.Duration(30)),
					},
					Status: string(constants2.DinosaurRequestStatusPreparing),
				},
			},
			wantErr:                true,
			wantErrMsg:             "",
			expectedDinosaurStatus: constants2.DinosaurRequestStatusPreparing,
		},
		{
			name: "Encounter a 5xx error Dinosaur preparation and skipped the retry",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					PrepareDinosaurRequestFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return errors.GeneralError("simulate 5xx error")
					},
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{
					Meta: api.Meta{
						CreatedAt: time.Now().Add(time.Minute * time.Duration(-30)),
					},
					Status: string(constants2.DinosaurRequestStatusPreparing),
				},
			},
			wantErr:                true,
			wantErrMsg:             "simulate 5xx error",
			expectedDinosaurStatus: constants2.DinosaurRequestStatusFailed,
		},
		{
			name: "Encounter a Client error (4xx) in Dinosaur preparation",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					PrepareDinosaurRequestFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return errors.NotFound("simulate a 4xx error")
					},
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{
					Status: string(constants2.DinosaurRequestStatusPreparing),
				},
			},
			wantErr:                true,
			wantErrMsg:             "simulate a 4xx error",
			expectedDinosaurStatus: constants2.DinosaurRequestStatusFailed,
		},
		{
			name: "Encounter an SSO Client internal error in Dinosaur creation and performed the retry",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					PrepareDinosaurRequestFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return errors.FailedToCreateSSOClient("ErrorFailedToCreateSSOClientReason")
					},
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(30))}},
			},
			wantErr:    true,
			wantErrMsg: "",
		},
		{
			name: "Encounter an SSO Client internal error in Dinosaur creation and skipped the retry",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					PrepareDinosaurRequestFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return errors.FailedToCreateSSOClient("ErrorFailedToCreateSSOClientReason")
					},
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{Meta: api.Meta{CreatedAt: time.Now().Add(time.Minute * time.Duration(-30))}},
			},
			wantErr:                true,
			wantErrMsg:             "ErrorFailedToCreateSSOClientReason",
			expectedDinosaurStatus: constants2.DinosaurRequestStatusFailed,
		},
		{
			name: "Successful reconcile",
			fields: fields{
				dinosaurService: &services.DinosaurServiceMock{
					PrepareDinosaurRequestFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
					UpdateFunc: func(dinosaurRequest *dbapi.DinosaurRequest) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				dinosaur: &dbapi.DinosaurRequest{},
			},
			wantErr:    false,
			wantErrMsg: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &PreparingDinosaurManager{
				dinosaurService: tt.fields.dinosaurService,
			}

			if err := k.reconcilePreparingDinosaur(tt.args.dinosaur); (err != nil) != tt.wantErr {
				t.Errorf("reconcilePreparingDinosaur() error = %v, wantErr %v", err, tt.wantErr)
			}

			gomega.Expect(tt.expectedDinosaurStatus.String()).Should(gomega.Equal(tt.args.dinosaur.Status))
			gomega.Expect(tt.args.dinosaur.FailedReason).Should(gomega.Equal(tt.wantErrMsg))
		})
	}
}
