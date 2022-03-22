package services

import (
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func Test_KafkaInstanceTypes_GetSupportedKafkaInstanceTypesByRegion(t *testing.T) {
	type fields struct {
		cloudProvidersService CloudProvidersService
	}

	type args struct {
		cloudProvider string
		cloudRegion   string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    []api.SupportedKafkaInstanceType
	}{
		{
			name: "success when instance type list",
			fields: fields{
				cloudProvidersService: &CloudProvidersServiceMock{
					ListCachedCloudProviderRegionsFunc: func(id string) ([]api.CloudRegion, *errors.ServiceError) {
						return []api.CloudRegion{
							{
								Id:                     "us-east-1",
								CloudProvider:          "aws",
								SupportedInstanceTypes: []string{"standard", "eval"},
							},
						}, nil
					},
				},
			},
			args: args{
				cloudProvider: "aws",
				cloudRegion:   "us-east-1",
			},
			wantErr: false,
			want: []api.SupportedKafkaInstanceType{
				{
					Id: "standard",
				},
				{
					Id: "eval",
				},
			},
		},
		{
			name: "successfully return an empty list if region not available",
			fields: fields{
				cloudProvidersService: &CloudProvidersServiceMock{
					ListCachedCloudProviderRegionsFunc: func(id string) ([]api.CloudRegion, *errors.ServiceError) {
						return []api.CloudRegion{
							{
								Id:                     "eu-west-1",
								CloudProvider:          "aws",
								SupportedInstanceTypes: []string{"standard", "eval"},
							},
						}, nil
					},
				},
			},
			args: args{
				cloudProvider: "aws",
				cloudRegion:   "us-east-1",
			},
			wantErr: false,
			want:    []api.SupportedKafkaInstanceType{},
		},
		{
			name: "fail when failure to get cloud regions",
			fields: fields{
				cloudProvidersService: &CloudProvidersServiceMock{
					ListCachedCloudProviderRegionsFunc: func(id string) ([]api.CloudRegion, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to get cloud regions")
					},
				},
			},
			args: args{
				cloudProvider: "aws",
				cloudRegion:   "us-east-1",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := supportedKafkaInstanceTypesService{
				providerService: tt.fields.cloudProvidersService,
			}
			got, err := k.GetSupportedKafkaInstanceTypesByRegion(tt.args.cloudProvider, tt.args.cloudRegion)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("supported Kafka instance type lists dont match. want: %v got: %v", tt.want, got)
			}
		})
	}
}
