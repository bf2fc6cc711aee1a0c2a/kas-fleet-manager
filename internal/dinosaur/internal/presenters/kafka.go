package presenters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/api/public"
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
func PresentDinosaurRequest(dinosaurRequest *dbapi.DinosaurRequest) public.DinosaurRequest {
	reference := PresentReference(dinosaurRequest.ID, dinosaurRequest)

	return public.DinosaurRequest{
		Id:            reference.Id,
		Kind:          reference.Kind,
		Href:          reference.Href,
		Region:        dinosaurRequest.Region,
		Name:          dinosaurRequest.Name,
		CloudProvider: dinosaurRequest.CloudProvider,
		MultiAz:       dinosaurRequest.MultiAZ,
		Owner:         dinosaurRequest.Owner,
		Status:        dinosaurRequest.Status,
		CreatedAt:     dinosaurRequest.CreatedAt,
		UpdatedAt:     dinosaurRequest.UpdatedAt,
		FailedReason:  dinosaurRequest.FailedReason,
		Version:       dinosaurRequest.ActualDinosaurVersion,
		InstanceType:  dinosaurRequest.InstanceType,
	}
}

func setBootstrapServerHost(bootstrapServerHost string) string {
	if bootstrapServerHost != "" {
		return fmt.Sprintf("%s:443", bootstrapServerHost)
	}
	return bootstrapServerHost
}
