package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

type cloudProvidersHandler struct {
	service services.CloudProvidersService
	config  services.ConfigService
}

func NewCloudProviderHandler(service services.CloudProvidersService, configService services.ConfigService) *cloudProvidersHandler {
	return &cloudProvidersHandler{
		service: service,
		config:  configService,
	}
}

func (h cloudProvidersHandler) ListCloudProviderRegions(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	cfg := &handlerConfig{

		Validate: []validate{
			validateLength(&id, "id", &minRequiredFieldLength, nil),
		},

		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			cloudRegions, err := h.service.ListCloudProviderRegions(id)
			if err != nil {
				return nil, err
			}
			regionList := openapi.CloudRegionList{
				Kind:  "CloudRegionList",
				Total: int32(len(cloudRegions)),
				Size:  int32(len(cloudRegions)),
				Page:  int32(1),
			}
			for _, cloudRegion := range cloudRegions {
				cloudRegion.Enabled = h.config.IsRegionSupportedForProvider(cloudRegion.CloudProvider, cloudRegion.Id)
				converted := presenters.PresentCloudRegion(&cloudRegion)
				regionList.Items = append(regionList.Items, converted)
			}
			return regionList, nil
		},
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
}

func (h cloudProvidersHandler) ListCloudProviders(w http.ResponseWriter, r *http.Request) {

	cfg := &handlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			cloudProviders, err := h.service.ListCloudProviders()
			if err != nil {
				return nil, err
			}
			cloudProviderList := openapi.CloudProviderList{
				Kind:  "CloudProviderList",
				Total: int32(len(cloudProviders)),
				Size:  int32(len(cloudProviders)),
				Page:  int32(1),
			}

			for _, cloudProvider := range cloudProviders {
				cloudProvider.Enabled = h.config.IsProviderSupported(cloudProvider.Id)
				converted := presenters.PresentCloudProvider(&cloudProvider)
				cloudProviderList.Items = append(cloudProviderList.Items, converted)
			}

			return cloudProviderList, nil
		},
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
}
