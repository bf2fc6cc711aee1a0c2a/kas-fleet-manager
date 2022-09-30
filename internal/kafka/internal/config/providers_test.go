package config

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"
)

var (
	providerName = "aws"
	regionName   = "us-east-1"
	limit        = 1
	instType     = types.STANDARD.String()
	instTypeMap  = InstanceTypeMap{
		instType: InstanceTypeConfig{
			Limit: &limit,
		},
	}
	region = Region{
		Default:                true,
		Name:                   regionName,
		SupportedInstanceTypes: instTypeMap,
	}
	regionList = RegionList{
		region,
	}
	provider = Provider{
		Name:    providerName,
		Default: true,
		Regions: regionList,
	}
	providerList = ProviderList{
		provider,
	}
	providerConfig = ProviderConfig{
		ProvidersConfig: ProviderConfiguration{
			SupportedProviders: providerList,
		},
	}
)

func Test_InstanceTypeMapAsSlice(t *testing.T) {
	g := gomega.NewWithT(t)
	g.Expect(instTypeMap.AsSlice()).To(gomega.Equal([]string{"standard"}))
}

func Test_IsInstanceTypeSupported(t *testing.T) {
	type args struct {
		instanceType string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should return true for supported instance type",
			args: args{
				instanceType: types.STANDARD.String(),
			},
			want: true,
		},
		{
			name: "should return false for unsupported instance type",
			args: args{
				instanceType: "unsupported",
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(region.IsInstanceTypeSupported(InstanceType(tt.args.instanceType))).To(gomega.Equal(tt.want))
		})
	}
}

func Test_getLimitSetForInstanceTypeInRegion(t *testing.T) {
	type args struct {
		instanceType string
	}

	tests := []struct {
		name    string
		args    args
		want    *int
		wantErr bool
	}{
		{
			name: "should return limit for supported instance type",
			args: args{
				instanceType: types.STANDARD.String(),
			},
			want:    &limit,
			wantErr: false,
		},
		{
			name: "should return nil for limit and an error for unsupported instance type",
			args: args{
				instanceType: "unsupported",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			limit, err := region.getLimitSetForInstanceTypeInRegion(tt.args.instanceType)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(limit).To(gomega.Equal(tt.want))
		})
	}
}

func Test_GetByName(t *testing.T) {
	type fields struct {
		regionName string
	}

	tests := []struct {
		name   string
		fields fields
		want   Region
		found  bool
	}{
		{
			name: "should return region found",
			fields: fields{
				regionName: region.Name,
			},
			want:  region,
			found: true,
		},
		{
			name: "should return empty region and false if not found",
			fields: fields{
				regionName: "unsupported",
			},
			want:  Region{},
			found: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			region, found := regionList.GetByName(tt.fields.regionName)
			g.Expect(region).To(gomega.Equal(tt.want))
			g.Expect(found).To(gomega.Equal(tt.found))
		})
	}
}

func Test_String_RegionList(t *testing.T) {
	type fields struct {
		regionList RegionList
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return the slice of region list names as a string separeated by commas for non-empty regionList",
			fields: fields{
				regionList: regionList,
			},
			want: fmt.Sprintf("[%s]", regionList[0].Name),
		},
		{
			name: "should return empty string for empty regionList",
			fields: fields{
				regionList: RegionList{},
			},
			want: "[]",
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.regionList.String()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_String_ProviderList(t *testing.T) {
	type fields struct {
		providerList ProviderList
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return the slice of provider list names as a string separeated by commas for non-empty ProviderList",
			fields: fields{
				providerList: providerList,
			},
			want: fmt.Sprintf("[%s]", providerList[0].Name),
		},
		{
			name: "should return empty string for empty ProviderList",
			fields: fields{
				providerList: ProviderList{},
			},
			want: "[]",
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.providerList.String()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_NewSupportedProvidersConfig(t *testing.T) {
	tests := []struct {
		name string
		want *ProviderConfig
	}{
		{
			name: "should return NewSupportedProvidersConfig",
			want: &ProviderConfig{
				ProvidersConfigFile: "config/provider-configuration.yaml",
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewSupportedProvidersConfig()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ReadFilesProviderConfig(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "should return no error when running ReadFiles for ProviderConfig",
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewSupportedProvidersConfig().ReadFiles() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_GetInstanceLimit(t *testing.T) {
	type args struct {
		region       string
		providerName string
		InstanceType string
	}
	type fields struct {
		providerConfig ProviderConfig
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *int
		wantErr bool
	}{
		{
			name: "should return the limit for provided region, provider and instance type from providerConfig",
			fields: fields{
				providerConfig: providerConfig,
			},
			args: args{
				region:       regionName,
				providerName: providerName,
				InstanceType: instType,
			},
			want:    &limit,
			wantErr: false,
		},
		{
			name: "should return an error and no limit for not supported provider from providerConfig",
			fields: fields{
				providerConfig: providerConfig,
			},
			args: args{
				providerName: "invalid",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should return an error and no limit for not supported region from providerConfig",
			fields: fields{
				providerConfig: providerConfig,
			},
			args: args{
				region:       "invalid",
				providerName: providerName,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should return an error and no limit for not supported instanceType from providerConfig",
			fields: fields{
				providerConfig: providerConfig,
			},
			args: args{
				region:       regionName,
				providerName: providerName,
				InstanceType: "invalid",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			limit, err := tt.fields.providerConfig.GetInstanceLimit(tt.args.region, tt.args.providerName, tt.args.InstanceType)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(limit).To(gomega.Equal(tt.want))
		})
	}
}

func Test_readFileProvidersConfig(t *testing.T) {
	type args struct {
		file string
		val  *ProviderConfiguration
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "should return no error when reading ProviderConfig file with valid params",
			args: args{
				file: "config/provider-configuration.yaml",
				val:  &providerConfig.ProvidersConfig,
			},
			wantErr: false,
		},
		{
			name: "should return no error when reading ProviderConfig file with valid params",
			args: args{
				file: "invalid",
				val:  &providerConfig.ProvidersConfig,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := readFileProvidersConfig(tt.args.file, tt.args.val)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_IsRegionSupported(t *testing.T) {
	type args struct {
		region string
	}
	type fields struct {
		provider Provider
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "should return the limit for provided region, provider and instance type from providerConfig",
			fields: fields{
				provider: provider,
			},
			args: args{
				region: regionName,
			},
			want: true,
		},
		{
			name: "should return an error and no limit for not supported provider from providerConfig",
			fields: fields{
				provider: provider,
			},
			args: args{
				region: "invalid",
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.provider.IsRegionSupported(tt.args.region)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Region_Validate(t *testing.T) {
	t.Parallel()
	type args struct {
		dataplaneClusterConfig *DataplaneClusterConfig
	}
	type fields struct {
		region Region
	}

	instTypeStandard := types.STANDARD.String()

	testClusterList := ClusterList{
		ManualCluster{
			Name:                  "cluster-1",
			SupportedInstanceType: "standard",
			Region:                "us-east-1",
			CloudProvider:         "aws",
			Schedulable:           true,
			MultiAZ:               true,
			Status:                "ready",
			ClusterId:             "cluster-id-1",
			KafkaInstanceLimit:    10,
			ProviderType:          api.ClusterProviderOCM,
		},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "It should succeed when cluster-level limits match with region-level limits",
			fields: fields{
				region: Region{
					Name:    regionName,
					Default: true,
					SupportedInstanceTypes: InstanceTypeMap{
						instTypeStandard: InstanceTypeConfig{
							Limit: &[]int{10}[0],
						},
					},
				},
			},
			args: args{
				dataplaneClusterConfig: &DataplaneClusterConfig{
					DataPlaneClusterScalingType: ManualScaling,
					ClusterConfig:               NewClusterConfig(testClusterList),
				},
			},
			wantErr: false,
		},
		{
			name: "It should return an error when cluster-level limit is not equal than the region-level limit",
			fields: fields{
				region: Region{
					Name:    regionName,
					Default: true,
					SupportedInstanceTypes: InstanceTypeMap{
						instTypeStandard: InstanceTypeConfig{
							Limit: &[]int{11}[0],
						},
					},
				},
			},
			args: args{
				dataplaneClusterConfig: &DataplaneClusterConfig{
					DataPlaneClusterScalingType: ManualScaling,
					ClusterConfig:               NewClusterConfig(testClusterList),
				},
			},
			wantErr: true,
		},
		{
			name: "When dynamic scaling is enabled the manual data plane cluster level limits are not evaluated",
			fields: fields{
				region: Region{
					Name:    regionName,
					Default: true,
					SupportedInstanceTypes: InstanceTypeMap{
						instTypeStandard: InstanceTypeConfig{
							Limit: &[]int{11}[0],
						},
					},
				},
			},
			args: args{
				dataplaneClusterConfig: &DataplaneClusterConfig{
					DataPlaneClusterScalingType: AutoScaling,
					ClusterConfig:               NewClusterConfig(testClusterList),
				},
			},
			wantErr: false,
		},
		{
			name: "It should return an error when minimum available capacity slack is greater than the region limit",
			fields: fields{
				region: Region{
					Name:    regionName,
					Default: true,
					SupportedInstanceTypes: InstanceTypeMap{
						instTypeStandard: InstanceTypeConfig{
							Limit:                                   &[]int{10}[0],
							MinAvailableCapacitySlackStreamingUnits: 11,
						},
					},
				},
			},
			args: args{
				dataplaneClusterConfig: &DataplaneClusterConfig{
					DataPlaneClusterScalingType: AutoScaling,
					ClusterConfig:               NewClusterConfig(testClusterList),
				},
			},
			wantErr: true,
		},
		{
			name: "It should return an error when minimum available capacity slack is negative",
			fields: fields{
				region: Region{
					Name:    regionName,
					Default: true,
					SupportedInstanceTypes: InstanceTypeMap{
						instTypeStandard: InstanceTypeConfig{
							Limit:                                   &[]int{10}[0],
							MinAvailableCapacitySlackStreamingUnits: -1,
						},
					},
				},
			},
			args: args{
				dataplaneClusterConfig: &DataplaneClusterConfig{
					DataPlaneClusterScalingType: AutoScaling,
					ClusterConfig:               NewClusterConfig(testClusterList),
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			err := tt.fields.region.Validate(tt.args.dataplaneClusterConfig)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestProvider_Validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		Name    string
		Regions RegionList
	}
	type args struct {
		dataplaneClusterConfig *DataplaneClusterConfig
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should not return an error if provider is valid when scaling type is manual",
			fields: fields{
				Name: "aws",
				Regions: RegionList{
					{
						Name:    "us-east-1",
						Default: true,
						SupportedInstanceTypes: InstanceTypeMap{
							"instance-type": InstanceTypeConfig{
								Limit: nil,
							},
						},
					},
				},
			},
			args: args{
				dataplaneClusterConfig: &DataplaneClusterConfig{
					DataPlaneClusterScalingType: ManualScaling,
					ClusterConfig: NewClusterConfig(ClusterList{
						ManualCluster{
							Name:                  "cluster-1",
							SupportedInstanceType: "standard",
							Region:                "us-east-1",
							CloudProvider:         "aws",
							Schedulable:           true,
							MultiAZ:               true,
							Status:                "ready",
							ClusterId:             "cluster-id-1",
							KafkaInstanceLimit:    10,
							ProviderType:          api.ClusterProviderOCM,
						},
					}),
				},
			},
			wantErr: false,
		},
		{
			name: "should return an error if no default region",
			fields: fields{
				Name: "aws",
				Regions: RegionList{
					{
						Name:    "us-east-1",
						Default: false,
						SupportedInstanceTypes: InstanceTypeMap{
							"instance-type": InstanceTypeConfig{
								Limit: nil,
							},
						},
					},
				},
			},
			args: args{
				dataplaneClusterConfig: &DataplaneClusterConfig{
					DataPlaneClusterScalingType: ManualScaling,
					ClusterConfig: NewClusterConfig(ClusterList{
						ManualCluster{
							Name:                  "cluster-1",
							SupportedInstanceType: "standard",
							Region:                "us-east-1",
							CloudProvider:         "aws",
							Schedulable:           true,
							MultiAZ:               true,
							Status:                "ready",
							ClusterId:             "cluster-id-1",
							KafkaInstanceLimit:    10,
							ProviderType:          api.ClusterProviderOCM,
						},
					}),
				},
			},
			wantErr: true,
		},
		{
			name: "should not return an error if provider is valid when scaling type is auto",
			fields: fields{
				Name: "aws",
				Regions: RegionList{
					{
						Name:    "us-east-1",
						Default: true,
						SupportedInstanceTypes: InstanceTypeMap{
							"instance-type": InstanceTypeConfig{
								Limit: nil,
							},
						},
					},
				},
			},
			args: args{
				dataplaneClusterConfig: &DataplaneClusterConfig{
					DataPlaneClusterScalingType: AutoScaling,
					ClusterConfig:               &ClusterConfig{},
					DynamicScalingConfig: DynamicScalingConfig{
						ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]ComputeMachinesConfig{
							cloudproviders.AWS: {
								ClusterWideWorkload: &ComputeMachineConfig{
									ComputeMachineType: "some-machine-type",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should return an error if provider machine type configuration are missing when scaling type is auto",
			fields: fields{
				Name: "aws",
				Regions: RegionList{
					{
						Name:    "us-east-1",
						Default: true,
						SupportedInstanceTypes: InstanceTypeMap{
							"instance-type": InstanceTypeConfig{
								Limit: nil,
							},
						},
					},
				},
			},
			args: args{
				dataplaneClusterConfig: &DataplaneClusterConfig{
					DataPlaneClusterScalingType: AutoScaling,
					ClusterConfig:               &ClusterConfig{},
					DynamicScalingConfig: DynamicScalingConfig{
						ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]ComputeMachinesConfig{},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			provider := Provider{
				Name:    testcase.fields.Name,
				Regions: testcase.fields.Regions,
			}

			err := provider.Validate(testcase.args.dataplaneClusterConfig)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}
