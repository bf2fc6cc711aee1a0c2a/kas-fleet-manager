package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	serviceError "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	pkgErr "github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/onsi/gomega"
)

const (
	resourceType           = "sampleResource"
	mockKafkaRequestID     = "9bsv0s3fd06g02i2be9g" // sample kafka request ID
	mockIDWithInvalidChars = "vp&xG^nl9MStC@SI*#c$6V^TKq0"
)

func buildKafkaDBApiRequest() dbapi.KafkaRequest {
	route := []byte("unitesthost.com/v1/test-route")
	return dbapi.KafkaRequest{
		Region:                           "us-east-1",
		ClusterID:                        uuid.NewString(),
		CloudProvider:                    "AWS",
		MultiAZ:                          false,
		Name:                             "test-cluster",
		Status:                           "Creating",
		CanaryServiceAccountClientID:     uuid.NewString(),
		CanaryServiceAccountClientSecret: uuid.NewString(),
		SubscriptionId:                   "test",
		Owner:                            "unit=test-user",
		OwnerAccountId:                   uuid.NewString(),
		BootstrapServerHost:              "unittesthost.com",
		OrganisationId:                   "unit-test-org",
		FailedReason:                     "",
		PlacementId:                      "unit-test",
		DesiredKafkaVersion:              "3",
		ActualKafkaVersion:               "3",
		DesiredStrimziVersion:            "0.27",
		ActualStrimziVersion:             "0.27",
		DesiredKafkaIBPVersion:           "v1",
		ActualKafkaIBPVersion:            "v1",
		KafkaUpgrading:                   false,
		StrimziUpgrading:                 false,
		KafkaIBPUpgrading:                false,
		KafkaStorageSize:                 "5gi",
		InstanceType:                     "developer",
		QuotaType:                        "ams",
		Routes:                           route,
		RoutesCreated:                    true,
		Namespace:                        "unit-test",
		ReauthenticationEnabled:          true,
		RoutesCreationId:                 uuid.NewString(),
	}
}

func Test_HandleGetError(t *testing.T) {
	cause := pkgErr.WithStack(gorm.ErrInvalidData)
	type args struct {
		resourceType string
		field        string
		value        interface{}
		err          error
	}
	tests := []struct {
		name string
		args args
		want *serviceError.ServiceError
	}{
		{
			name: "Handler should return a general error for any errors other than record not found",
			args: args{
				resourceType: resourceType,
				field:        "id",
				value:        "sample-id",
				err:          cause,
			},
			want: serviceError.NewWithCause(serviceError.ErrorGeneral, cause, "Unable to find %s with id='sample-id'", resourceType),
		},
		{
			name: "Handler should return a not found error if record was not found in the database",
			args: args{
				resourceType: resourceType,
				field:        "id",
				value:        "sample-id",
				err:          gorm.ErrRecordNotFound,
			},
			want: serviceError.NotFound("%s with id='sample-id' not found", resourceType),
		},
		{
			name: "Handler should redact sensitive fields from the error message",
			args: args{
				resourceType: resourceType,
				field:        "email",
				value:        "sample@example.com",
				err:          gorm.ErrRecordNotFound,
			},
			want: serviceError.NotFound("%s with email='<redacted>' not found", resourceType),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got := services.HandleGetError(tt.args.resourceType, tt.args.field, tt.args.value, tt.args.err)
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_handleCreateError(t *testing.T) {
	type args struct {
		resourceType string
		err          error
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "Handler should return a general error for any other errors than violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          gorm.ErrInvalidField,
			},
			want: errors.GeneralError("Unable to create %s: %s", resourceType, gorm.ErrInvalidField.Error()),
		},
		{
			name: "Handler should return a conflict error if creation error is due to violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          fmt.Errorf("transaction violates unique constraints"),
			},
			want: errors.Conflict("This %s already exists", resourceType),
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got := services.HandleCreateError(tt.args.resourceType, tt.args.err)
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_handleUpdateError(t *testing.T) {
	type args struct {
		resourceType string
		err          error
	}
	tests := []struct {
		name string
		args args
		want *errors.ServiceError
	}{
		{
			name: "Handler should return a general error for any other errors than violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          gorm.ErrInvalidData,
			},
			want: errors.GeneralError("Unable to update %s: %s", resourceType, gorm.ErrInvalidData.Error()),
		},
		{
			name: "Handler should return a conflict error if update error is due to violating unique constraints",
			args: args{
				resourceType: resourceType,
				err:          fmt.Errorf("transaction violates unique constraints"),
			},
			want: errors.Conflict("Changes to %s conflict with existing records", resourceType),
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got := services.HandleUpdateError(tt.args.resourceType, tt.args.err)
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_TruncateString(t *testing.T) {
	exampleString := "example-string"
	type args struct {
		str string
		num int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should truncate string successfully",
			args: args{
				str: exampleString,
				num: 10,
			},
			want: exampleString[0:10],
		},
		{
			name: "should not truncate string if wanted length is less than given string length",
			args: args{
				str: exampleString,
				num: 15,
			},
			want: exampleString,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got := TruncateString(tt.args.str, tt.args.num)
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_buildTruncateKafkaIdentifier(t *testing.T) {
	mockShortKafkaName := "kafka"
	mockLongKafkaName := "sample-kafka-name-long"

	type args struct {
		kafkaRequest *dbapi.KafkaRequest
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "build kafka identifier with a short name successfully",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Name: mockShortKafkaName,
				},
			},
			want: fmt.Sprintf("%s-%s", mockShortKafkaName, strings.ToLower(mockKafkaRequestID)),
		},
		{
			name: "build kafka identifier with a long name successfully",
			args: args{
				kafkaRequest: &dbapi.KafkaRequest{
					Name: mockLongKafkaName,
				},
			},
			want: fmt.Sprintf("%s-%s", mockLongKafkaName[0:truncatedNameLen], strings.ToLower(mockKafkaRequestID)),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.args.kafkaRequest.ID = mockKafkaRequestID
			got := buildTruncateKafkaIdentifier(tt.args.kafkaRequest)
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_MaskProceedingandTrailingDash(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should replace '-' prefix and suffix with a subdomain safe value",
			args: args{
				name: "-example-name-",
			},
			want: fmt.Sprintf("%[1]sexample-name%[1]s", appendChar),
		},
		{
			name: "should not replace '-' if its not a prefix or suffix of the given string",
			args: args{
				name: "example-name",
			},
			want: "example-name",
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got := MaskProceedingandTrailingDash(tt.args.name)
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_replaceHostSpecialChar(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "replace all invalid characters in an invalid host name",
			args: args{
				name: fmt.Sprintf("-host-%s", mockIDWithInvalidChars),
			},
			want: "ahost-vp-xg-nl-mstc-si-c--v-tkqa",
		},
		{
			name: "valid hostname should not be modified",
			args: args{
				name: "sample-host-name",
			},
			want: "sample-host-name",
		},
		{
			name: "should return an error if given host name is an empty string",
			args: args{
				name: "",
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got, err := replaceHostSpecialChar(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("replaceHostSpecialChar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_contains(t *testing.T) {
	searchedString := "findMe"
	someSlice := []string{"some", "string", "values"}
	sliceWithFindMe := []string{"some", "string", "values", "findMe"}
	var emptySlice []string
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check for a string in an empty slice",
			args: args{
				s:     searchedString,
				slice: emptySlice,
			},
			want: false,
		},
		{
			name: "Check for a string in a non-empty slice that doesn't contain the string",
			args: args{
				s:     searchedString,
				slice: someSlice,
			},
			want: false,
		},
		{
			name: "Check for a string in a non-empty slice that contains that string",
			args: args{
				s:     searchedString,
				slice: sliceWithFindMe,
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got := arrays.Contains(tt.args.slice, tt.args.s)
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_BuildCustomClaimCheck(t *testing.T) {
	type args struct {
		kafkaRequest      *dbapi.KafkaRequest
		ssoconfigProvider string
	}
	kafkaRequest := buildKafkaDBApiRequest()
	tests := []struct {
		name                string
		args                args
		expectedCustomClaim string
	}{
		{
			name: "Customclaimcheck without canary service account client ID - uses MAS SSO",
			args: args{
				kafkaRequest:      &kafkaRequest,
				ssoconfigProvider: keycloak.MAS_SSO,
			},
			expectedCustomClaim: fmt.Sprintf("@.rh-org-id == '%s'|| @.org_id == '%s'", kafkaRequest.OrganisationId, kafkaRequest.OrganisationId),
		},
		{
			name: "Customclaimcheck with canary service account client ID - uses REDHAT SSO",
			args: args{
				kafkaRequest:      &kafkaRequest,
				ssoconfigProvider: keycloak.REDHAT_SSO,
			},
			expectedCustomClaim: fmt.Sprintf("@.rh-org-id == '%s'|| @.org_id == '%s' || @.clientId == '%s'", kafkaRequest.OrganisationId, kafkaRequest.OrganisationId, kafkaRequest.CanaryServiceAccountClientID),
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			receivedBuiltCustomClaimCheck := BuildCustomClaimCheck(tt.args.kafkaRequest, tt.args.ssoconfigProvider)
			g.Expect(receivedBuiltCustomClaimCheck).To(gomega.Equal(tt.expectedCustomClaim))
		})
	}
}
