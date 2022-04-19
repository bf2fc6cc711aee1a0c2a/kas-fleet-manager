package config

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	. "github.com/onsi/gomega"
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
	RegisterTestingT(t)
	Expect(instTypeMap.AsSlice()).To(Equal([]string{"standard"}))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(region.IsInstanceTypeSupported(InstanceType(tt.args.instanceType))).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, err := region.getLimitSetForInstanceTypeInRegion(tt.args.instanceType)
			Expect(limit).To(Equal(tt.want))
			Expect(err != nil).To(Equal(tt.wantErr))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region, found := regionList.GetByName(tt.fields.regionName)
			Expect(region).To(Equal(tt.want))
			Expect(found).To(Equal(tt.found))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.regionList.String()).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.providerList.String()).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(NewSupportedProvidersConfig()).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(NewSupportedProvidersConfig().ReadFiles() != nil).To(Equal(tt.wantErr))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, err := tt.fields.providerConfig.GetInstanceLimit(tt.args.region, tt.args.providerName, tt.args.InstanceType)
			Expect(limit).To(Equal(tt.want))
			Expect(err != nil).To(Equal(tt.wantErr))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := readFileProvidersConfig(tt.args.file, tt.args.val)
			Expect(err != nil).To(Equal(tt.wantErr))
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

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.fields.provider.IsRegionSupported(tt.args.region)).To(Equal(tt.want))
		})
	}
}
