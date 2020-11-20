package handlers

import (
	"testing"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"

	. "github.com/onsi/gomega"
)

func Test_Validation_Provider(t *testing.T) {
	type args struct {
		kafkaRequest  openapi.KafkaRequest
		configService services.ConfigService
	}

	type result struct {
		wantErr      bool
		reason       string
		kafkaRequest openapi.KafkaRequest
	}

	tests := []struct {
		name string
		arg  args
		want result
	}{
		{
			name: "do not throw an error when default provider and region are picked",
			arg: args{
				kafkaRequest: openapi.KafkaRequest{},
				configService: services.NewConfigService(config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name:    "aws",
							Default: true,
							Regions: config.RegionList{
								config.Region{
									Name:    "us-east-1",
									Default: true,
								},
							},
						},
					},
				}),
			},
			want: result{
				wantErr: false,
				kafkaRequest: openapi.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
			},
		},
		{
			name: "do not throw an error when cloud provider and region matches",
			arg: args{
				kafkaRequest: openapi.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
				configService: services.NewConfigService(config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name: "gcp",
							Regions: config.RegionList{
								config.Region{
									Name: "eu-east-1",
								},
							},
						},
						config.Provider{
							Name: "aws",
							Regions: config.RegionList{
								config.Region{
									Name: "us-east-1",
								},
							},
						},
					},
				}),
			},
			want: result{
				wantErr: false,
				kafkaRequest: openapi.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
			},
		},
		{
			name: "throws an error when cloud provider and region do not match",
			arg: args{
				kafkaRequest: openapi.KafkaRequest{
					CloudProvider: "aws",
					Region:        "us-east",
				},
				configService: services.NewConfigService(config.ProviderConfiguration{
					SupportedProviders: config.ProviderList{
						config.Provider{
							Name: "aws",
							Regions: config.RegionList{
								config.Region{
									Name: "us-east-1",
								},
							},
						},
					},
				}),
			},
			want: result{
				wantErr: true,
				reason:  "region us-east is not supported for aws, supported regions are: [us-east-1]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			validateFn := validateCloudProvider(&tt.arg.kafkaRequest, tt.arg.configService, "creating-kafka")
			err := validateFn()
			if !tt.want.wantErr && err != nil {
				t.Errorf("validatedCloudProvider() expected not to throw error but threw %v", err)
			} else if tt.want.wantErr {
				Expect(err.Reason).To(Equal(tt.want.reason))
				return
			}

			Expect(tt.want.wantErr).To(Equal(err != nil))

			if !tt.want.wantErr {
				Expect(tt.arg.kafkaRequest.CloudProvider).To(Equal(tt.want.kafkaRequest.CloudProvider))
				Expect(tt.arg.kafkaRequest.Region).To(Equal(tt.want.kafkaRequest.Region))
			}

		})
	}
}
