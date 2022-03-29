package kafka_mgrs

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestKafkaManager_reconcileDeniedKafkaOwners(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}
	type args struct {
		deniedAccounts acl.DeniedUsers
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "do not reconcile when denied accounts list is empty",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil, // set to nil as it should not be called
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{},
			},
			wantErr: false,
		},
		{
			name: "should receive error when update in deprovisioning in database returns an error",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: func(users []string) *errors.ServiceError {
						return &errors.ServiceError{}
					},
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{"some user"},
			},
			wantErr: true,
		},
		{
			name: "should not receive error when update in deprovisioning in database succeed",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: func(users []string) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				deniedAccounts: acl.DeniedUsers{"some user"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &KafkaManager{
				kafkaService: tt.fields.kafkaService,
			}
			err := k.reconcileDeniedKafkaOwners(tt.args.deniedAccounts)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

var (
	cloudProviderStandardLimit = 5
)

func TestKafkaManager_capacityMetrics(t *testing.T) {
	type fields struct {
		kafkaService           services.KafkaService
		dataplaneClusterConfig config.DataplaneClusterConfig
		cloudProviders         config.ProviderConfig
		existingKafkas         []services.KafkaRegionCount
	}
	tests := []struct {
		name     string
		fields   fields
		expected map[string]map[string]float64
	}{
		{
			name: "expected available capacity with no cloud provider limit",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						{
							Name:                  "a",
							ClusterId:             "a",
							CloudProvider:         "aws",
							Region:                "us-east-1",
							MultiAZ:               true,
							Schedulable:           true,
							KafkaInstanceLimit:    10,
							Status:                "ready",
							SupportedInstanceType: "standard",
						},
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				existingKafkas: []services.KafkaRegionCount{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "a",
						Count:         5,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// no cloud provider limits
			// 5 standard instances used, 3 developer instances used
			// ==> expect 5 more standard instances to be available
			expected: map[string]map[string]float64{
				"us-east-1": {
					"standard": 5,
				},
			},
		},
		{
			name: "expected available capacity with no cloud provider limit and existing developer instances",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						{
							Name:                  "a",
							ClusterId:             "a",
							CloudProvider:         "aws",
							Region:                "us-east-1",
							MultiAZ:               true,
							Schedulable:           true,
							KafkaInstanceLimit:    10,
							Status:                "ready",
							SupportedInstanceType: "standard,developer",
						},
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: nil},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				existingKafkas: []services.KafkaRegionCount{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "a",
						Count:         5,
						CloudProvider: "aws",
					},
					{
						Region:        "us-east-1",
						InstanceType:  "developer",
						ClusterId:     "a",
						Count:         3,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// no cloud provider limits
			// 5 standard instances used
			// 3 developer instances used
			// ==> expect 2 more standard instances to be available
			// ==> expect 2 more developer instances to be available
			expected: map[string]map[string]float64{
				"us-east-1": {
					"standard":  2,
					"developer": 2,
				},
			},
		},
		{
			name: "expected available capacity with cloud provider limits and existing instances",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						{
							Name:                  "a",
							ClusterId:             "a",
							CloudProvider:         "aws",
							Region:                "us-east-1",
							MultiAZ:               true,
							Schedulable:           true,
							KafkaInstanceLimit:    10,
							Status:                "ready",
							SupportedInstanceType: "standard",
						},
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard": {Limit: &cloudProviderStandardLimit},
										},
									},
								},
							},
						},
					},
				},
				existingKafkas: []services.KafkaRegionCount{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "a",
						Count:         4,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// cloud provider limit of 5 standard instances
			// 4 standard instances used
			// ==> expect 1 more standard instance to be available
			expected: map[string]map[string]float64{
				"us-east-1": {
					"standard": 1,
				},
			},
		},
		{
			name: "expected available capacity with cloud provider limits and existing mixed instances",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					DeprovisionKafkaForUsersFunc: nil,
				},
				dataplaneClusterConfig: config.DataplaneClusterConfig{
					ClusterConfig: config.NewClusterConfig([]config.ManualCluster{
						{
							Name:                  "a",
							ClusterId:             "a",
							CloudProvider:         "aws",
							Region:                "us-east-1",
							MultiAZ:               true,
							Schedulable:           true,
							KafkaInstanceLimit:    10,
							Status:                "ready",
							SupportedInstanceType: "developer,standard",
						},
					}),
				},
				cloudProviders: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: []config.Provider{
							{
								Name:    "aws",
								Default: true,
								Regions: []config.Region{
									{
										Name:    "us-east-1",
										Default: true,
										SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
											"standard":  {Limit: &cloudProviderStandardLimit},
											"developer": {Limit: nil},
										},
									},
								},
							},
						},
					},
				},
				existingKafkas: []services.KafkaRegionCount{
					{
						Region:        "us-east-1",
						InstanceType:  "standard",
						ClusterId:     "a",
						Count:         4,
						CloudProvider: "aws",
					},
					{
						Region:        "us-east-1",
						InstanceType:  "developer",
						ClusterId:     "a",
						Count:         4,
						CloudProvider: "aws",
					},
				},
			},
			// 10 total capacity
			// cloud provider limit of 5 standard instances
			// 4 standard instances used
			// 4 developer instances used
			// ==> expect 1 more standard instance to be available
			// ==> expect 2 more developer instance to be available
			expected: map[string]map[string]float64{
				"us-east-1": {
					"standard":  1,
					"developer": 2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			k := &KafkaManager{
				kafkaService:           tt.fields.kafkaService,
				dataplaneClusterConfig: &tt.fields.dataplaneClusterConfig,
				cloudProviders:         &tt.fields.cloudProviders,
			}

			results, err := k.calculateAvailableCapacityByRegionAndInstanceType(tt.fields.existingKafkas)
			gomega.Expect(err).To(gomega.BeNil())

			for _, result := range results {
				gomega.Expect(result.Count).To(gomega.Equal(tt.expected[result.Region][result.InstanceType]))
			}
		})
	}
}
