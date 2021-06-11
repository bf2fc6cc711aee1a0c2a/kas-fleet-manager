package handlers

import (
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

type cloudProvidersHandler struct {
	service services.CloudProvidersService
	config  services.ConfigService
	cache   *cache.Cache
}

func NewCloudProviderHandler(service services.CloudProvidersService, configService services.ConfigService) *cloudProvidersHandler {
	return &cloudProvidersHandler{
		service: service,
		config:  configService,
		cache:   cache.New(5*time.Minute, 10*time.Minute),
	}
}

func (h cloudProvidersHandler) ListCloudProviderRegions(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	cfg := &HandlerConfig{
		Validate: []Validate{
			ValidateLength(&id, "id", &minRequiredFieldLength, nil),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			cachedRegionList, cached := h.cache.Get(id)
			if cached {
				return cachedRegionList, nil
			}
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
			h.cache.Set(id, regionList, cache.DefaultExpiration)
			return regionList, nil
		},
	}
	HandleGet(w, r, cfg)
}

func (h cloudProvidersHandler) ListCloudProviders(w http.ResponseWriter, r *http.Request) {
	cfg := &HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			cachedCloudProviderList, cached := h.cache.Get("cloudProviderList")
			if cached {
				return cachedCloudProviderList, nil
			}
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
			h.cache.Set("cloudProviderList", cloudProviderList, cache.DefaultExpiration)
			return cloudProviderList, nil
		},
	}
	HandleGet(w, r, cfg)
}
