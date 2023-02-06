package config

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/go-playground/validator/v10"
	"github.com/onsi/gomega"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

func Test_amsProductValidator(t *testing.T) {
	testValidator := validator.New()
	g := gomega.NewWithT(t)

	err := testValidator.RegisterValidation("ams_product_validator", amsProductValidator)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	type testInvalidType struct {
		TestField bool `validate:"ams_product_validator"`
	}

	invalidVal := testInvalidType{TestField: true}
	err = testValidator.Struct(invalidVal)
	g.Expect(err).To(gomega.HaveOccurred())

	type testType struct {
		TestField string `validate:"ams_product_validator"`
	}

	type args struct {
		testType testType
	}

	additionalTests := []struct {
		name    string
		wantErr bool
		args    args
	}{
		{
			name: "validation doesn't fail with accepted AMS product (I)",
			args: args{
				testType: testType{
					TestField: string(ocm.RHOSAKProduct),
				},
			},
			wantErr: false,
		},
		{
			name: "validation doesn't fail with accepted AMS product (II)",
			args: args{
				testType: testType{
					TestField: string(ocm.RHOSAKEvalProduct),
				},
			},
			wantErr: false,
		},
		{
			name: "validation doesn't fail with accepted AMS product (III)",
			args: args{
				testType: testType{
					TestField: string(ocm.RHOSAKEvalProduct),
				},
			},
			wantErr: false,
		},
		{
			name: "validation doesn't fail with accepted AMS product (IV)",
			args: args{
				testType: testType{
					TestField: string(ocm.RHOSAKCCProduct),
				},
			},
			wantErr: false,
		},
		{
			name: "validation fails with non accepted AMS product",
			args: args{
				testType: testType{
					TestField: "nonexistingproduct",
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range additionalTests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := testValidator.Struct(tt.args.testType)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

		})
	}
}

func Test_amsResourceValidator(t *testing.T) {
	testValidator := validator.New()
	g := gomega.NewWithT(t)

	err := testValidator.RegisterValidation("ams_resource_validator", amsResourceValidator)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	type testInvalidType struct {
		TestField bool `validate:"ams_resource_validator"`
	}

	invalidVal := testInvalidType{TestField: true}
	err = testValidator.Struct(invalidVal)
	g.Expect(err).To(gomega.HaveOccurred())

	type testType struct {
		TestField string `validate:"ams_resource_validator"`
	}

	type args struct {
		testType testType
	}

	additionalTests := []struct {
		name    string
		wantErr bool
		args    args
	}{
		{
			name: "validation doesn't fail with accepted AMS resource",
			args: args{
				testType: testType{
					TestField: string(ocm.RHOSAKResourceName),
				},
			},
			wantErr: false,
		},
		{
			name: "validation fails with non accepted AMS resource",
			args: args{
				testType: testType{
					TestField: "nonexistingproduct",
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range additionalTests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := testValidator.Struct(tt.args.testType)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

		})
	}
}

func Test_amsBillingModelsValidator(t *testing.T) {
	testValidator := validator.New()
	g := gomega.NewWithT(t)

	err := testValidator.RegisterValidation("ams_billing_models_validator", amsBillingModelsValidator)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	type testInvalidType struct {
		TestField []bool `validate:"ams_billing_models_validator"`
	}

	invalidVal := testInvalidType{TestField: []bool{true, false}}
	err = testValidator.Struct(invalidVal)
	g.Expect(err).To(gomega.HaveOccurred())

	type validTestTypeString struct {
		TestField string `validate:"ams_billing_models_validator"`
	}

	type validTestTypeSliceString struct {
		TestField []string `validate:"ams_billing_models_validator"`
	}

	type args struct {
		inputVal interface{}
	}

	additionalTests := []struct {
		name    string
		wantErr bool
		args    args
	}{
		{
			name: "validation doesn't fail with accepted AMS billing model (I)",
			args: args{
				inputVal: validTestTypeString{
					TestField: string(amsv1.BillingModelMarketplace),
				},
			},
			wantErr: false,
		},
		{
			name: "validation doesn't fail with accepted AMS billing model (II)",
			args: args{
				inputVal: validTestTypeString{
					TestField: string(amsv1.BillingModelMarketplaceRHM),
				},
			},
			wantErr: false,
		},
		{
			name: "validation doesn't fail with accepted AMS billing model (III)",
			args: args{
				inputVal: validTestTypeString{
					TestField: string(amsv1.BillingModelMarketplaceAWS),
				},
			},
			wantErr: false,
		},
		{
			name: "validation doesn't fail with accepted AMS billing model (IV)",
			args: args{
				inputVal: validTestTypeString{
					TestField: string(amsv1.BillingModelStandard),
				},
			},
			wantErr: false,
		},
		{
			name: "validation doesn't fail with accepted AMS billing model (V)",
			args: args{
				inputVal: validTestTypeSliceString{
					TestField: []string{
						string(amsv1.BillingModelMarketplaceAWS),
						string(amsv1.BillingModelMarketplaceRHM),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "validation fails with non accepted AMS billing model (I)",
			args: args{
				inputVal: validTestTypeString{
					TestField: "nonexistingbillingmodel",
				},
			},
			wantErr: true,
		},
		{
			name: "validation fails with non accepted AMS billing model (I)",
			args: args{
				inputVal: validTestTypeSliceString{
					TestField: []string{
						string(amsv1.BillingModelMarketplaceAWS),
						"nonvalidbillingmodel",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range additionalTests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := testValidator.Struct(tt.args.inputVal)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))

		})
	}
}
