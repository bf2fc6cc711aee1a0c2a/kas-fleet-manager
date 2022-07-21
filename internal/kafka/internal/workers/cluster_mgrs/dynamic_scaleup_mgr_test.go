package cluster_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"

	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func Test_DynamicScaleUpManager_reconcileClustersForRegions(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		clusterProvidersConfig config.ProviderConfig
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	testAutoScalingDataPlaneConfig := &config.DataplaneClusterConfig{
		DataPlaneClusterScalingType: config.AutoScaling,
	}
	testProvider := "aws"
	testRegion := "us-east-1"
	testClusterProvidersConfig := config.ProviderConfig{
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

	tests := []struct {
		name    string
		wantErr bool
		fields  fields
	}{
		{
			name: "creates a missing OSD cluster request automatically when autoscaling is enabled",
			fields: fields{
				dataplaneClusterConfig: testAutoScalingDataPlaneConfig,
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *apiErrors.ServiceError) {
						res := []*services.ResGroupCPRegion{
							{
								Provider: testProvider,
								Region:   testRegion,
								Count:    1,
							},
						}
						return res, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				clusterProvidersConfig: testClusterProvidersConfig,
			},
			wantErr: false,
		},
		// {
		// 	name: "skips reconciliation if autoscaling is disabled",
		// 	fields: fields{
		// 		dataplaneClusterConfig: config.NewDataplaneClusterConfig(),
		// 	},
		// 	wantErr: false,
		// },
		{
			name: "should return an error if ListGroupByProviderAndRegion fails",
			fields: fields{
				dataplaneClusterConfig: testAutoScalingDataPlaneConfig,
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *apiErrors.ServiceError) {
						res := []*services.ResGroupCPRegion{}
						return res, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to register cluster job")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "failed to create OSD request with empty services.ResGroupCPRegion",
			fields: fields{
				dataplaneClusterConfig: testAutoScalingDataPlaneConfig,
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *apiErrors.ServiceError) {
						var res []*services.ResGroupCPRegion
						return res, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				clusterProvidersConfig: testClusterProvidersConfig,
			},
			wantErr: false,
		},
		{
			name: "should create OSD request",
			fields: fields{
				dataplaneClusterConfig: testAutoScalingDataPlaneConfig,
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *apiErrors.ServiceError) {
						var res []*services.ResGroupCPRegion
						return res, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to create cluster request")
					},
				},
				clusterProvidersConfig: testClusterProvidersConfig,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			m := DynamicScaleUpManager{
				ClusterService:         tt.fields.clusterService,
				ClusterProvidersConfig: &tt.fields.clusterProvidersConfig,
				DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
			}
			err := m.reconcileClustersForRegions()
			g.Expect(err != nil && !tt.wantErr).To(gomega.BeFalse())
		})
	}
}
