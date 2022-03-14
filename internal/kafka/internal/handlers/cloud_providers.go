package handlers

import (
	"net/http"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"

	"github.com/patrickmn/go-cache"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
)

const cloudProvidersCacheKey = "cloudProviderList"

type cloudProvidersHandler struct {
	service                  services.CloudProvidersService
	cache                    *cache.Cache
	supportedProviders       config.ProviderList
	kafkaService             services.KafkaService
	clusterPlacementStrategy services.ClusterPlacementStrategy
	kafkaConfig              *config.KafkaConfig
}

func NewCloudProviderHandler(service services.CloudProvidersService, providerConfig *config.ProviderConfig, kafkaService services.KafkaService, clusterPlacementStrategy services.ClusterPlacementStrategy, kafkaConfig *config.KafkaConfig) *cloudProvidersHandler {
	return &cloudProvidersHandler{
		service:                  service,
		supportedProviders:       providerConfig.ProvidersConfig.SupportedProviders,
		cache:                    cache.New(5*time.Minute, 10*time.Minute),
		kafkaService:             kafkaService,
		clusterPlacementStrategy: clusterPlacementStrategy,
		kafkaConfig:              kafkaConfig,
	}
}

func (h cloudProvidersHandler) ListCloudProviderRegions(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	query := r.URL.Query()
	instanceTypeFilter := query.Get("instance_type")

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&id, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			cloudRegions, err := h.service.ListCachedCloudProviderRegions(id)
			if err != nil {
				return nil, err
			}
			regionList := public.CloudRegionList{
				Kind:  "CloudRegionList",
				Page:  int32(1),
				Items: []public.CloudRegion{},
			}

			provider, _ := h.supportedProviders.GetByName(id)
			for _, cloudRegion := range cloudRegions {
				region, _ := provider.Regions.GetByName(cloudRegion.Id)

				// skip any regions that do not support the specified instance type so its not included in the response
				if instanceTypeFilter != "" && !region.IsInstanceTypeSupported(config.InstanceType(instanceTypeFilter)) {
					continue
				}

				kafka := &dbapi.KafkaRequest{}

				// Enabled set to true and Capacity set only if at least one instance type is supported by the region
				if region.SupportedInstanceTypes != nil && len(region.SupportedInstanceTypes) > 0 {
					capacities := []api.RegionCapacityListItem{}
					cloudRegion.Enabled = true
					cloudRegion.SupportedInstanceTypes = region.SupportedInstanceTypes.AsSlice()
					for _, instType := range cloudRegion.SupportedInstanceTypes {
						maxCapacityReached := true
						kafka.InstanceType = instType
						kafka.Region = cloudRegion.Id
						size, e := h.kafkaConfig.GetFirstAvailableSize(instType)
						if e != nil {
							return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to list cloud provider regions")
						}
						kafka.SizeId = size
						kafka.CloudProvider = cloudRegion.CloudProvider
						hasCapacity, err := h.kafkaService.HasAvailableCapacityInRegion(kafka)
						if err == nil && hasCapacity {
							cluster, err := h.clusterPlacementStrategy.FindCluster(kafka)
							if err == nil && cluster != nil {
								maxCapacityReached = false
							}
						}
						capacity := api.RegionCapacityListItem{InstanceType: instType, DeprecatedMaxCapacityReached: maxCapacityReached}
						capacities = append(capacities, capacity)
					}
					cloudRegion.Capacity = capacities
				}
				converted := presenters.PresentCloudRegion(&cloudRegion)
				regionList.Items = append(regionList.Items, converted)
			}

			regionList.Total = int32(len(regionList.Items))
			regionList.Size = int32(len(regionList.Items))

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
