package handlers

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/cloud_providers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
)

var (
	supportedProviders = config.ProviderConfig{
		ProvidersConfig: config.ProviderConfiguration{
			SupportedProviders: config.ProviderList{
				config.Provider{
					Name: "aws",
					Regions: config.RegionList{
						config.Region{
							Name: "us-east-1",
							SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
								"standard":  {Limit: nil},
								"developer": {Limit: nil},
							},
						},
					},
				},
			},
		},
	}
	fullKafkaConfig = config.KafkaConfig{
		SupportedInstanceTypes: &config.KafkaSupportedInstanceTypesConfig{
			Configuration: config.SupportedKafkaInstanceTypesConfig{
				SupportedKafkaInstanceTypes: []config.KafkaInstanceType{
					{
						Id:          "standard",
						DisplayName: "Standard",
						Sizes: []config.KafkaInstanceSize{
							{
								Id:                          "x1",
								IngressThroughputPerSec:     "30Mi",
								EgressThroughputPerSec:      "30Mi",
								TotalMaxConnections:         1000,
								MaxDataRetentionSize:        "100Gi",
								MaxPartitions:               1000,
								MaxDataRetentionPeriod:      "P14D",
								MaxConnectionAttemptsPerSec: 100,
								QuotaConsumed:               1,
								CapacityConsumed:            0,
								MaxMessageSize:              "1Mi",
								MinInSyncReplicas:           2,
								ReplicationFactor:           3,
							},
						},
					},
					{
						Id:          "developer",
						DisplayName: "Developer",
						Sizes: []config.KafkaInstanceSize{
							{
								Id:                          "x1",
								IngressThroughputPerSec:     "30Mi",
								EgressThroughputPerSec:      "30Mi",
								TotalMaxConnections:         1000,
								MaxDataRetentionSize:        "100Gi",
								MaxPartitions:               1000,
								MaxDataRetentionPeriod:      "P14D",
								MaxConnectionAttemptsPerSec: 100,
								QuotaConsumed:               1,
								CapacityConsumed:            0,
								MaxMessageSize:              "1Mi",
								MinInSyncReplicas:           2,
								ReplicationFactor:           3,
							},
						},
					},
				},
			},
		},
	}
)

func Test_ListCloudProviderRegions(t *testing.T) {
	type fields struct {
		cloudProvidersService    services.CloudProvidersService
		providerConfig           *config.ProviderConfig
		kafkaService             services.KafkaService
		clusterPlacementStrategy services.ClusterPlacementStrategy
		kafkaConfig              *config.KafkaConfig
	}

	type args struct {
		url string
		id  string
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return an error if id param is empty",
			args: args{
				url: "/",
			},
			fields: fields{
				providerConfig: &supportedProviders,
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if cloudProvidersService returns an error when listing cached provider regions",
			args: args{
				url: "/",
				id:  "aws",
			},
			fields: fields{
				cloudProvidersService: &services.CloudProvidersServiceMock{
					ListCachedCloudProviderRegionsFunc: func(id string) ([]api.CloudRegion, *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
				},
				providerConfig: &supportedProviders,
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return an error if cloudProvidersService returns an error when listing cached provider regions",
			args: args{
				url: "/",
				id:  "aws",
			},
			fields: fields{
				cloudProvidersService: &services.CloudProvidersServiceMock{
					ListCachedCloudProviderRegionsFunc: func(id string) ([]api.CloudRegion, *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
				},
				providerConfig: &supportedProviders,
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return no error when cloudProvidersService returns empty regions list",
			args: args{
				url: "/",
				id:  "aws",
			},
			fields: fields{
				cloudProvidersService: &services.CloudProvidersServiceMock{
					ListCachedCloudProviderRegionsFunc: func(id string) ([]api.CloudRegion, *errors.ServiceError) {
						return []api.CloudRegion{}, nil
					},
				},
				providerConfig: &supportedProviders,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return no error when cloudProvidersService returns non-empty regions list",
			args: args{
				url: "/",
				id:  "aws",
			},
			fields: fields{
				cloudProvidersService: &services.CloudProvidersServiceMock{
					ListCachedCloudProviderRegionsFunc: func(id string) ([]api.CloudRegion, *errors.ServiceError) {
						return []api.CloudRegion{
							*mocks.BuildApiCloudRegion(nil),
						}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					HasAvailableCapacityInRegionFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, *errors.ServiceError) {
						return true, nil
					},
					GetAvailableSizesInRegionFunc: func(criteria *services.FindClusterCriteria) ([]string, *errors.ServiceError) {
						return []string{}, nil
					},
				},
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							ClusterID: "test",
						}, nil
					},
				},
				providerConfig: &supportedProviders,
				kafkaConfig:    &fullKafkaConfig,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return an error when GetAvailableSizesInRegion fails",
			args: args{
				url: "/",
				id:  "aws",
			},
			fields: fields{
				cloudProvidersService: &services.CloudProvidersServiceMock{
					ListCachedCloudProviderRegionsFunc: func(id string) ([]api.CloudRegion, *errors.ServiceError) {
						return []api.CloudRegion{
							*mocks.BuildApiCloudRegion(nil),
						}, nil
					},
				},
				kafkaService: &services.KafkaServiceMock{
					HasAvailableCapacityInRegionFunc: func(kafkaRequest *dbapi.KafkaRequest) (bool, *errors.ServiceError) {
						return true, nil
					},
					GetAvailableSizesInRegionFunc: func(criteria *services.FindClusterCriteria) ([]string, *errors.ServiceError) {
						return []string{}, errors.GeneralError("test")
					},
				},
				clusterPlacementStrategy: &services.ClusterPlacementStrategyMock{
					FindClusterFunc: func(kafka *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
						return &api.Cluster{
							ClusterID: "test",
						}, nil
					},
				},
				providerConfig: &supportedProviders,
				kafkaConfig:    &fullKafkaConfig,
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewCloudProviderHandler(tt.fields.cloudProvidersService, tt.fields.providerConfig, tt.fields.kafkaService,
				tt.fields.clusterPlacementStrategy, tt.fields.kafkaConfig)

			req, rw := GetHandlerParams("GET", tt.args.url, nil, t)
			if tt.args.id != "" {
				req = mux.SetURLVars(req, map[string]string{"id": tt.args.id})
			}
			h.ListCloudProviderRegions(rw, req)
			resp := rw.Result()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}

func Test_ListCloudProviders(t *testing.T) {
	type fields struct {
		cloudProvidersService services.CloudProvidersService
		providerConfig        *config.ProviderConfig
		kafkaService          services.KafkaService
		kafkaConfig           *config.KafkaConfig
	}

	tests := []struct {
		name           string
		fields         fields
		wantStatusCode int
	}{
		{
			name: "should return empty cloud providers list",
			fields: fields{
				providerConfig: &supportedProviders,
				cloudProvidersService: &services.CloudProvidersServiceMock{
					ListCloudProvidersFunc: func() ([]api.CloudProvider, *errors.ServiceError) {
						return []api.CloudProvider{}, nil
					},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should fail if listing cloud providers by the service fails",
			fields: fields{
				providerConfig: &supportedProviders,
				cloudProvidersService: &services.CloudProvidersServiceMock{
					ListCloudProvidersFunc: func() ([]api.CloudProvider, *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return a non-empty cloud providers list",
			fields: fields{
				providerConfig: &supportedProviders,
				cloudProvidersService: &services.CloudProvidersServiceMock{
					ListCloudProvidersFunc: func() ([]api.CloudProvider, *errors.ServiceError) {
						return []api.CloudProvider{
							*mocks.BuildApiCloudProvider(nil),
						}, nil
					},
				},
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewCloudProviderHandler(tt.fields.cloudProvidersService, tt.fields.providerConfig, tt.fields.kafkaService,
				nil, tt.fields.kafkaConfig)

			req, rw := GetHandlerParams("GET", "/", nil, t)
			h.ListCloudProviders(rw, req)
			resp := rw.Result()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}
