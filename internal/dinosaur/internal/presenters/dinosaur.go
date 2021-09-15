package presenters

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/account"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
)

// ConvertDinosaurRequest from payload to DinosaurRequest
func ConvertDinosaurRequest(dinosaurRequest public.DinosaurRequestPayload) *dbapi.DinosaurRequest {
	return &dbapi.DinosaurRequest{
		Region:        dinosaurRequest.Region,
		Name:          dinosaurRequest.Name,
		CloudProvider: dinosaurRequest.CloudProvider,
		MultiAZ:       dinosaurRequest.MultiAz,
	}
}

// PresentDinosaurRequest - create DinosaurRequest in an appropriate format ready to be returned by the API
func PresentDinosaurRequest(dinosaurRequest *dbapi.DinosaurRequest, accountService account.AccountService) (*public.DinosaurRequest, *errors.ServiceError) {
	reference := PresentReference(dinosaurRequest.ID, dinosaurRequest)

	org, err := accountService.GetOrganization(fmt.Sprintf("external_id='%s'", dinosaurRequest.OrganisationId))

	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "error presenting the request")
	}

	if org == nil {
		return nil, errors.New(errors.ErrorGeneral, "unable to find an organisation for external_id '%s'", dinosaurRequest.OrganisationId)
	}

	if org == nil {

	}

	return &public.DinosaurRequest{
		Id:            reference.Id,
		Kind:          reference.Kind,
		Href:          reference.Href,
		Region:        dinosaurRequest.Region,
		Name:          dinosaurRequest.Name,
		CloudProvider: dinosaurRequest.CloudProvider,
		MultiAz:       dinosaurRequest.MultiAZ,
		Owner:         dinosaurRequest.Owner,
		AccountNumber: org.AccountNumber,
		Status:        dinosaurRequest.Status,
		CreatedAt:     dinosaurRequest.CreatedAt,
		UpdatedAt:     dinosaurRequest.UpdatedAt,
		FailedReason:  dinosaurRequest.FailedReason,
		Version:       dinosaurRequest.ActualDinosaurVersion,
		InstanceType:  dinosaurRequest.InstanceType,
	}, nil
}

func setBootstrapServerHost(bootstrapServerHost string) string {
	if bootstrapServerHost != "" {
		return fmt.Sprintf("%s:443", bootstrapServerHost)
	}
	return bootstrapServerHost
}
