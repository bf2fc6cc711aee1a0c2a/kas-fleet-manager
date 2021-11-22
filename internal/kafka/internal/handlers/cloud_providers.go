package handlers

import (
	"net/http"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"

	"github.com/patrickmn/go-cache"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
)

const cloudProvidersCacheKey = "cloudProviderList"

type cloudProvidersHandler struct {
	service            services.CloudProvidersService
	cache              *cache.Cache
	supportedProviders config.ProviderList
}

func NewCloudProviderHandler(service services.CloudProvidersService, providerConfig *config.ProviderConfig) *cloudProvidersHandler {
	return &cloudProvidersHandler{
		service:            service,
		supportedProviders: providerConfig.ProvidersConfig.SupportedProviders,
		cache:              cache.New(5*time.Minute, 10*time.Minute),
	}
}

func (h cloudProvidersHandler) ListCloudProviderRegions(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	query := r.URL.Query()
	instanceTypeFilter := query.Get("instance_type")
	cacheId := id
	if instanceTypeFilter != "" {
		cacheId = cacheId + "-" + instanceTypeFilter
	}

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&id, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			cachedRegionList, cached := h.cache.Get(cacheId)
			if cached {
				return cachedRegionList, nil
			}
			cloudRegions, err := h.service.ListCloudProviderRegions(id)
			if err != nil {
				return nil, err
			}
			regionList := public.CloudRegionList{
				Kind:  "CloudRegionList",
				Total: int32(len(cloudRegions)),
				Size:  int32(len(cloudRegions)),
				Page:  int32(1),
				Items: []public.CloudRegion{},
			}

			provider, _ := h.supportedProviders.GetByName(id)
			for _, cloudRegion := range cloudRegions {
				region, _ := provider.Regions.GetByName(cloudRegion.Id)

				// if instance_type was specified, only set enabled to true for regions that supports the specified instance type. Otherwise,
				// set enable to true for all region that supports any instance types
				if instanceTypeFilter != "" {
					cloudRegion.Enabled = region.IsInstanceTypeSupported(config.InstanceType(instanceTypeFilter))
				} else {
					cloudRegion.Enabled = len(region.SupportedInstanceTypes) > 0
				}

				cloudRegion.SupportedInstanceTypes = region.SupportedInstanceTypes.AsSlice()
				converted := presenters.PresentCloudRegion(&cloudRegion)
				regionList.Items = append(regionList.Items, converted)
			}

			h.cache.Set(cacheId, regionList, cache.DefaultExpiration)
			return regionList, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h cloudProvidersHandler) ListCloudProviders(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			cachedCloudProviderList, cached := h.cache.Get(cloudProvidersCacheKey)
			if cached {
				return cachedCloudProviderList, nil
			}
			cloudProviders, err := h.service.ListCloudProviders()
			if err != nil {
				return nil, err
			}
			cloudProviderList := public.CloudProviderList{
				Kind:  "CloudProviderList",
				Total: int32(len(cloudProviders)),
				Size:  int32(len(cloudProviders)),
				Page:  int32(1),
				Items: []public.CloudProvider{},
			}

			for _, cloudProvider := range cloudProviders {
				_, cloudProvider.Enabled = h.supportedProviders.GetByName(cloudProvider.Id)
				converted := presenters.PresentCloudProvider(&cloudProvider)
				cloudProviderList.Items = append(cloudProviderList.Items, converted)
			}
			h.cache.Set(cloudProvidersCacheKey, cloudProviderList, cache.DefaultExpiration)
			return cloudProviderList, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}
