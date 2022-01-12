package presenters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
)

// ConvertDinosaurRequest from payload to DinosaurRequest
func ConvertDinosaurRequest(dinosaurRequestPayload public.DinosaurRequestPayload, dbDinosaurrequest ...*dbapi.DinosaurRequest) *dbapi.DinosaurRequest {
	var dinosaur *dbapi.DinosaurRequest
	if len(dbDinosaurrequest) == 0 {
		dinosaur = &dbapi.DinosaurRequest{}
	} else {
		dinosaur = dbDinosaurrequest[0]
	}

	dinosaur.Region = dinosaurRequestPayload.Region
	dinosaur.Name = dinosaurRequestPayload.Name
	dinosaur.CloudProvider = dinosaurRequestPayload.CloudProvider
	dinosaur.MultiAZ = dinosaurRequestPayload.MultiAz

	if dinosaurRequestPayload.ReauthenticationEnabled != nil {
		dinosaur.ReauthenticationEnabled = *dinosaurRequestPayload.ReauthenticationEnabled
	} else {
		dinosaur.ReauthenticationEnabled = true // true by default
	}

	return dinosaur
}

// PresentDinosaurRequest - create DinosaurRequest in an appropriate format ready to be returned by the API
func PresentDinosaurRequest(dinosaurRequest *dbapi.DinosaurRequest) public.DinosaurRequest {
	reference := PresentReference(dinosaurRequest.ID, dinosaurRequest)

	return public.DinosaurRequest{
		Id:                      reference.Id,
		Kind:                    reference.Kind,
		Href:                    reference.Href,
		Region:                  dinosaurRequest.Region,
		Name:                    dinosaurRequest.Name,
		CloudProvider:           dinosaurRequest.CloudProvider,
		MultiAz:                 dinosaurRequest.MultiAZ,
		Owner:                   dinosaurRequest.Owner,
		BootstrapServerHost:     setBootstrapServerHost(dinosaurRequest.BootstrapServerHost),
		Status:                  dinosaurRequest.Status,
		CreatedAt:               dinosaurRequest.CreatedAt,
		UpdatedAt:               dinosaurRequest.UpdatedAt,
		FailedReason:            dinosaurRequest.FailedReason,
		Version:                 dinosaurRequest.ActualDinosaurVersion,
		InstanceType:            dinosaurRequest.InstanceType,
		ReauthenticationEnabled: dinosaurRequest.ReauthenticationEnabled,
	}
}

func setBootstrapServerHost(bootstrapServerHost string) string {
	if bootstrapServerHost != "" {
		return fmt.Sprintf("%s:443", bootstrapServerHost)
	}
	return bootstrapServerHost
}
