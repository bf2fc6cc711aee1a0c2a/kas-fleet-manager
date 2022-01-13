package presenters

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/compat"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/handlers"
)

const (
	// KindDinosaur is a string identifier for the type api.DinosaurRequest
	KindDinosaur = "Dinosaur"
	// CloudRegion is a string identifier for the type api.CloudRegion
	KindCloudRegion = "CloudRegion"
	// KindCloudProvider is a string identifier for the type api.CloudProvider
	KindCloudProvider = "CloudProvider"
	// KindError is a string identifier for the type api.ServiceError
	KindError = "Error"

	BasePath = "/api/dinosaurs_mgmt/v1"
)

func PresentReference(id, obj interface{}) compat.ObjectReference {
	return handlers.PresentReferenceWith(id, obj, objectKind, objectPath)
}

func objectKind(i interface{}) string {
	switch i.(type) {
	case dbapi.DinosaurRequest, *dbapi.DinosaurRequest:
		return KindDinosaur
	case api.CloudRegion, *api.CloudRegion:
		return KindCloudRegion
	case api.CloudProvider, *api.CloudProvider:
		return KindCloudProvider
	case errors.ServiceError, *errors.ServiceError:
		return KindError
	default:
		return ""
	}
}

func objectPath(id string, obj interface{}) string {
	switch obj.(type) {
	case dbapi.DinosaurRequest, *dbapi.DinosaurRequest:
		return fmt.Sprintf("%s/dinosaurs/%s", BasePath, id)
	case errors.ServiceError, *errors.ServiceError:
		return fmt.Sprintf("%s/errors/%s", BasePath, id)
	case api.ServiceAccount, *api.ServiceAccount:
		return fmt.Sprintf("%s/service_accounts/%s", BasePath, id)
	default:
		return ""
	}
}
