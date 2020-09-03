package presenters

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
)

func ConvertDinosaur(org openapi.Dinosaur) *api.Dinosaur {
	return &api.Dinosaur{
		Meta: api.Meta{
			ID: org.Id,
		},
		Species: org.Species,
	}
}

func PresentDinosaur(dinosaur *api.Dinosaur) openapi.Dinosaur {
	reference := PresentReference(dinosaur.ID, dinosaur)
	return openapi.Dinosaur{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		Species:   dinosaur.Species,
		CreatedAt: dinosaur.CreatedAt,
		UpdatedAt: dinosaur.UpdatedAt,
	}
}
