package errors

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/compat"
	. "github.com/onsi/gomega"
	ocmErrors "github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/pkg/errors"
)

var (
	e                   = ServiceError{}
	testCause           = "failed to do something"
	testError           = New(http.StatusBadRequest, testCause)
	errorWithCause      = NewWithCause(ErrorBadRequest, testError, "Unable to list kafka requests: %s", testError.Error())
	genericErrorMessage = "something went wrong"
)

func TestErrorFormatting(t *testing.T) {
	RegisterTestingT(t)
	err := New(ErrorGeneral, "test %s, %d", "errors", 1)
	Expect(err.Reason).To(Equal("test errors, 1"))
}

func TestErrorFind(t *testing.T) {
	RegisterTestingT(t)
	exists, err := Find(ErrorNotFound)
	Expect(exists).To(Equal(true))
	Expect(err.Code).To(Equal(ErrorNotFound))

	// Hopefully we never reach 91,823,719 error codes or this test will fail
	exists, err = Find(ServiceErrorCode(91823719))
	Expect(exists).To(Equal(false))
	Expect(err).To(BeNil())
}

func Test_NewErrorFromHTTPStatusCode(t *testing.T) {
	badRequestReason := "bad request reason"
	unauthorizedReason := "unauthorised reason"
	forbiddenReason := "forbidden reason"
	notFoundReason := "not found reason"
	notImplementedReason := "not implemented reason"
	conflictReason := "conflict reason"
	generalReason := "general reason"
	type args struct {
		httpCode int
		reason   string
	}

	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return bad request error",
			args: args{
				httpCode: http.StatusBadRequest,
				reason:   badRequestReason,
			},
			want: BadRequest(badRequestReason),
		},
		{
			name: "should return unauthorised error",
			args: args{
				httpCode: http.StatusUnauthorized,
				reason:   unauthorizedReason,
			},
			want: Unauthorized(unauthorizedReason),
		},
		{
			name: "should return forbidden error",
			args: args{
				httpCode: http.StatusForbidden,
				reason:   forbiddenReason,
			},
			want: Forbidden(forbiddenReason),
		},
		{
			name: "should return not found error",
			args: args{
				httpCode: http.StatusNotFound,
				reason:   notFoundReason,
			},
			want: NotFound(notFoundReason),
		},
		{
			name: "should return not implemented error",
			args: args{
				httpCode: http.StatusMethodNotAllowed,
				reason:   notImplementedReason,
			},
			want: NotImplemented(notImplementedReason),
		},
		{
			name: "should return conflict error",
			args: args{
				httpCode: http.StatusConflict,
				reason:   conflictReason,
			},
			want: Conflict(conflictReason),
		},
		{
			name: "should return general error",
			args: args{
				httpCode: http.StatusInternalServerError,
				reason:   generalReason,
			},
			want: GeneralError(generalReason),
		},
		{
			name: "general case, should return general error",
			want: GeneralError("Unspecified error"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(NewErrorFromHTTPStatusCode(tt.args.httpCode, tt.args.reason)).To(Equal(tt.want))
		})
	}
}

type errorWithoutStackTrace struct {
}

func (e *errorWithoutStackTrace) Error() string {
	return "Error"
}
func Test_NewWithCause(t *testing.T) {
	internalServerCause := "Unspecified error"
	type args struct {
		code   ServiceErrorCode
		cause  error
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a service error with a nil cause",
			args: args{
				reason: internalServerCause,
			},
			want: &ServiceError{ErrorGeneral, "Unspecified error", http.StatusInternalServerError, nil},
		},
		{
			name: "should return a service error if the cause is not nil",
			args: args{
				reason: internalServerCause,
				cause:  GeneralError(genericErrorMessage),
			},
			want: &ServiceError{ErrorGeneral, "Unspecified error", http.StatusInternalServerError, GeneralError("")},
		},
		{
			name: "should return a service error where there is no stack trace",
			args: args{
				reason: internalServerCause,
				cause:  &errorWithoutStackTrace{},
			},
			want: &ServiceError{ErrorGeneral, "Unspecified error", http.StatusInternalServerError, errors.WithStack(&errorWithoutStackTrace{})},
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := NewWithCause(tt.args.code, tt.args.cause, tt.args.reason)
			Expect(err.Code).To(Equal(tt.want.Code))
			Expect(err.Reason).To(Equal(tt.want.Reason))
			Expect(err.HttpCode).To(Equal(tt.want.HttpCode))
			if err.cause != nil {
				_, ok := err.cause.(stackTracer)
				Expect((ok)).To(BeTrue())
			}
		})
	}
}

func Test_FailedToCreateSSOClient(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorFailedToCreateSSOClient",
			args: args{
				reason: "Failed to create kafka client in the mas sso",
			},
			want: New(ErrorFailedToCreateSSOClient, "Failed to create kafka client in the mas sso"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FailedToCreateSSOClient(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_FailedToGetSSOClientSecret(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorFailedToGetSSOClientSecret",
			args: args{
				reason: "Failed to get kafka client secret from the mas sso",
			},
			want: New(ErrorFailedToGetSSOClientSecret, "Failed to get kafka client secret from the mas sso"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FailedToGetSSOClientSecret(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_FailedToGetSSOClient(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorFailedToGetSSOClient",
			args: args{
				reason: "Failed to get kafka client from the mas sso",
			},
			want: New(ErrorFailedToGetSSOClient, "Failed to get kafka client from the mas sso"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FailedToGetSSOClient(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_FailedToDeleteSSOClient(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorFailedToDeleteSSOClient",
			args: args{
				reason: "Failed to delete kafka client from the mas sso",
			},
			want: New(ErrorFailedToDeleteSSOClient, "Failed to delete kafka client from the mas sso"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FailedToDeleteSSOClient(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_FailedToCreateServiceAccount(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorFailedToCreateServiceAccount",
			args: args{
				reason: "Failed to create service account",
			},
			want: New(ErrorFailedToCreateServiceAccount, "Failed to create service account"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FailedToCreateServiceAccount(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_FailedToDeleteServiceAccount(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorFailedToDeleteServiceAccount",
			args: args{
				reason: "Failed to delete service account",
			},
			want: New(ErrorFailedToDeleteServiceAccount, "Failed to delete service account"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FailedToDeleteServiceAccount(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_MaxLimitForServiceAccountReached(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorMaxLimitForServiceAccountsReached",
			args: args{
				reason: "Max limit for the service account creation has reached",
			},
			want: New(ErrorMaxLimitForServiceAccountsReached, "Max limit for the service account creation has reached"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(MaxLimitForServiceAccountReached(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_FailedToGetServiceAccount(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorFailedToGetServiceAccount",
			args: args{
				reason: "Failed to get service account",
			},
			want: New(ErrorFailedToGetServiceAccount, "Failed to get service account"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FailedToGetServiceAccount(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_ServiceAccountNotFound(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorServiceAccountNotFound",
			args: args{
				reason: "Failed to find service account",
			},
			want: New(ErrorServiceAccountNotFound, "Failed to find service account"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(ServiceAccountNotFound(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_RegionNotSupported(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorRegionNotSupported",
			args: args{
				reason: "Region not supported",
			},
			want: New(ErrorRegionNotSupported, "Region not supported"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(RegionNotSupported(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_InstanceTypeNotSupported(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorInstanceTypeNotSupported",
			args: args{
				reason: "Instance Type not supported",
			},
			want: New(ErrorInstanceTypeNotSupported, "Instance Type not supported"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(InstanceTypeNotSupported(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_ProviderNotSupported(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorProviderNotSupported",
			args: args{
				reason: "Provider not supported",
			},
			want: New(ErrorProviderNotSupported, "Provider not supported"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(ProviderNotSupported(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_InstancePlanNotSupported(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorInstancePlanNotSupported",
			args: args{
				reason: "Instance plan not supported",
			},
			want: New(ErrorInstancePlanNotSupported, "Instance plan not supported"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(InstancePlanNotSupported(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_MalformedKafkaClusterName(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorMalformedKafkaClusterName",
			args: args{
				reason: "Kafka cluster name is invalid",
			},
			want: New(ErrorMalformedKafkaClusterName, "Kafka cluster name is invalid"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(MalformedKafkaClusterName(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_MalformedServiceAccountName(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorMalformedServiceAccountName",
			args: args{
				reason: "Service account name is invalid",
			},
			want: New(ErrorMalformedServiceAccountName, "Service account name is invalid"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(MalformedServiceAccountName(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_MalformedServiceAccountDesc(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorMalformedServiceAccountDesc",
			args: args{
				reason: "Service account desc is invalid",
			},
			want: New(ErrorMalformedServiceAccountDesc, "Service account desc is invalid"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(MalformedServiceAccountDesc(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_MalformedServiceAccountId(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorMalformedServiceAccountId",
			args: args{
				reason: "Service account id is invalid",
			},
			want: New(ErrorMalformedServiceAccountId, "Service account id is invalid"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(MalformedServiceAccountId(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_DuplicateKafkaClusterName(t *testing.T) {
	tests := []struct {
		name string
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorDuplicateKafkaClusterName",
			want: New(ErrorDuplicateKafkaClusterName, ErrorDuplicateKafkaClusterNameReason),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(DuplicateKafkaClusterName()).To(Equal(tt.want))
		})
	}
}

func Test_MinimumFieldLengthNotReached(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorMinimumFieldLength",
			args: args{
				reason: "Minimum field length not reached",
			},
			want: New(ErrorMinimumFieldLength, "Minimum field length not reached"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(MinimumFieldLengthNotReached(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_MaximumFieldLengthExceeded(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorMaximumFieldLength",
			args: args{
				reason: "Maximum field length has been depassed",
			},
			want: New(ErrorMaximumFieldLength, "Maximum field length has been depassed"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(MaximumFieldLengthExceeded(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_UnableToSendErrorResponse(t *testing.T) {
	tests := []struct {
		name string
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorUnableToSendErrorResponse",
			want: New(ErrorUnableToSendErrorResponse, ErrorUnableToSendErrorResponseReason),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(UnableToSendErrorResponse()).To(Equal(tt.want))
		})
	}
}

func Test_FailedToParseQueryParms(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new bad request error",
			args: args{
				reason: "Bad request",
			},
			want: New(ErrorBadRequest, "Bad request"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FailedToParseQueryParms(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_FieldValidationError(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorFieldValidationError",
			args: args{
				reason: "",
			},
			want: New(ErrorFieldValidationError, "Field validation failed: "),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FieldValidationError(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_InsufficientQuotaError(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorInsufficientQuota",
			args: args{
				reason: "",
			},
			want: New(ErrorInsufficientQuota, "Insufficient quota: "),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(InsufficientQuotaError(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_FailedToCheckQuota(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorFailedToCheckQuota",
			args: args{
				reason: "",
			},
			want: New(ErrorFailedToCheckQuota, "Failed to check quota: "),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FailedToCheckQuota(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_InvalidBillingAccount(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorBillingAccountInvalid",
			args: args{
				reason: "",
			},
			want: New(ErrorBillingAccountInvalid, "Billing account id missing or invalid: "),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(InvalidBillingAccount(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_TermsNotAccepted(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorTermsNotAccepted",
			args: args{
				reason: "Required terms have not been accepted",
			},
			want: New(ErrorTermsNotAccepted, "Required terms have not been accepted"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(TermsNotAccepted(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_Unauthenticated(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorUnauthenticated",
			args: args{
				reason: "Account authentication could not be verified",
			},
			want: New(ErrorUnauthenticated, "Account authentication could not be verified"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(Unauthenticated(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_Maintenance(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorServiceIsUnderMaintenance",
			args: args{
				reason: "Unable to perform this action, as the service is currently under maintenance",
			},
			want: New(ErrorServiceIsUnderMaintenance, "Unable to perform this action, as the service is currently under maintenance"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(Maintenance(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_MaximumAllowedInstanceReached(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorMaxAllowedInstanceReached",
			args: args{
				reason: "Forbidden to create more instances than the maximum allowed",
			},
			want: New(ErrorMaxAllowedInstanceReached, "Forbidden to create more instances than the maximum allowed"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(MaximumAllowedInstanceReached(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_TooManyKafkaInstancesReached(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorTooManyKafkaInstancesReached",
			args: args{
				reason: "The maximum number of allowed kafka instances has been reached",
			},
			want: New(ErrorTooManyKafkaInstancesReached, "The maximum number of allowed kafka instances has been reached"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(TooManyKafkaInstancesReached(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_Validation(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorValidation",
			args: args{
				reason: "General validation failure",
			},
			want: New(ErrorValidation, "General validation failure"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(Validation(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_MalformedRequest(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorMalformedRequest",
			args: args{
				reason: "Unable to read request body",
			},
			want: New(ErrorMalformedRequest, "Unable to read request body"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(MalformedRequest(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_FailedToParseSearch(t *testing.T) {
	type args struct {
		reason string
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorFailedToParseSearch",
			args: args{
				reason: "",
			},
			want: New(ErrorFailedToParseSearch, "Failed to parse search query: "),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(FailedToParseSearch(tt.args.reason)).To(Equal(tt.want))
		})
	}
}

func Test_SyncActionNotSupported(t *testing.T) {
	tests := []struct {
		name string
		want *ServiceError
	}{
		{
			name: "should return a new service error if ErrorSyncActionNotSupported",
			want: New(ErrorSyncActionNotSupported, ErrorSyncActionNotSupportedReason),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(SyncActionNotSupported()).To(Equal(tt.want))
		})
	}
}

func Test_CodeStr(t *testing.T) {
	type args struct {
		code ServiceErrorCode
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should return a formatted string for CodeStr()",
			args: args{
				code: http.StatusBadRequest,
			},
			want: fmt.Sprintf("%s-%d", ERROR_CODE_PREFIX, http.StatusBadRequest),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(CodeStr(tt.args.code)).To(Equal(tt.want))
		})
	}
}

func Test_Href(t *testing.T) {
	type args struct {
		code ServiceErrorCode
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should return a formatted string for Href()",
			args: args{
				code: http.StatusBadRequest,
			},
			want: fmt.Sprintf("%s%d", ERROR_HREF, http.StatusBadRequest),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(Href(tt.args.code)).To(Equal(tt.want))
		})
	}
}

func Test_ToServiceError(t *testing.T) {
	RegisterTestingT(t)
	sampleNonServiceError, err := ocmErrors.NewError().Reason("Unspecified error").Build()
	Expect(err).To(BeNil())
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a service error if a service error occured",
			args: args{
				err: BadRequest(""),
			},
			want: BadRequest(""),
		},
		{
			name: "should retun a servis error if a on servce error occured",
			args: args{
				err: sampleNonServiceError,
			},
			want: GeneralError(""),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(ToServiceError(tt.args.err)).To(Equal(tt.want))
		})
	}
}

func Test_Is404(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error Is404()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == NotFound(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.Is404()).To(Equal(tt.want))
		})
	}
}

func Test_IsConflict(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsConflict()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == Conflict(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsConflict()).To(Equal(tt.want))
		})
	}
}

func Test_IsForbidden(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsForbidden()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == Forbidden(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsForbidden()).To(Equal(tt.want))
		})
	}
}

func Test_IsFailedToCreateSSOClient(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsFailedToCreateSSOClient()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == FailedToCreateSSOClient(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsFailedToCreateSSOClient()).To(Equal(tt.want))
		})
	}
}

func Test_IsFailedToGetSSOClientSecret(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsFailedToGetSSOClientSecret()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == FailedToGetSSOClientSecret(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsFailedToGetSSOClientSecret()).To(Equal(tt.want))
		})
	}
}

func Test_IsFailedToGetSSOClient(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsFailedToGetSSOClient()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == FailedToGetSSOClient(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsFailedToGetSSOClient()).To(Equal(tt.want))
		})
	}
}

func Test_IsFailedToDeleteSSOClient(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsFailedToDeleteSSOClient()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == FailedToDeleteSSOClient(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsFailedToDeleteSSOClient()).To(Equal(tt.want))
		})
	}
}

func Test_IsFailedToCreateServiceAccount(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsFailedToCreateServiceAccount()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == FailedToCreateServiceAccount(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsFailedToCreateServiceAccount()).To(Equal(tt.want))
		})
	}
}

func Test_IsFailedToGetServiceAccount(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsFailedToGetServiceAccount()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == FailedToGetServiceAccount(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsFailedToGetServiceAccount()).To(Equal(tt.want))
		})
	}
}

func Test_IsFailedToDeleteServiceAccount(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsFailedToDeleteServiceAccount()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == FailedToDeleteServiceAccount(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsFailedToDeleteServiceAccount()).To(Equal(tt.want))
		})
	}
}

func Test_IsServiceAccountNotFound(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsServiceAccountNotFound()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == ServiceAccountNotFound(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsServiceAccountNotFound()).To(Equal(tt.want))
		})
	}
}

func Test_IsMaxLimitForServiceAccountReached(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsMaxLimitForServiceAccountReached()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == MaxLimitForServiceAccountReached(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsMaxLimitForServiceAccountReached()).To(Equal(tt.want))
		})
	}
}

func Test_IsBadRequest(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsBadRequest()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == BadRequest(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsBadRequest()).To(Equal(tt.want))
		})
	}
}

func Test_InSufficientQuota(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error is InSufficientQuota()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == InsufficientQuotaError(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.InSufficientQuota()).To(Equal(tt.want))
		})
	}
}

func Test_IsFailedToCheckQuota(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsFailedToCheckQuota()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == FailedToCheckQuota(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsFailedToCheckQuota()).To(Equal(tt.want))
		})
	}
}

func Test_IsInstanceTypeNotSupported(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsInstanceTypeNotSupported()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.Code == InstanceTypeNotSupported(genericErrorMessage).Code,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsInstanceTypeNotSupported()).To(Equal(tt.want))
		})
	}
}

func Test_IsClientErrorClass(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsClientErrorClass()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.HttpCode >= http.StatusBadRequest && e.HttpCode < http.StatusInternalServerError,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsClientErrorClass()).To(Equal(tt.want))
		})
	}
}

func Test_IsServerErrorClass(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return a boolean value true if the error IsServerErrorClass()",
			fields: fields{
				err: &ServiceError{},
			},
			want: e.HttpCode >= http.StatusInternalServerError,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.IsServerErrorClass()).To(Equal(tt.want))
		})
	}
}

func Test_AsOpenapiError(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	type args struct {
		operationID string
		basePath    string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   compat.Error
	}{
		{
			name: "should return compat.Error",
			fields: fields{
				err: &ServiceError{},
			},
			want: compat.Error{
				Id:          strconv.Itoa(int(e.Code)),
				Kind:        "Error",
				Href:        Href(e.Code),
				Code:        CodeStr(e.Code),
				Reason:      e.Reason,
				OperationId: "",
			},
		},
		{
			name: "should return compat.Error",
			fields: fields{
				err: &ServiceError{},
			},
			args: args{
				basePath: "/api/connector_mgmt/",
			},
			want: compat.Error{
				Id:          strconv.Itoa(int(e.Code)),
				Kind:        "Error",
				Href:        strings.Replace(Href(e.Code), ERROR_HREF, CONNECTOR_MGMT_ERROR_HREF, 1),
				Code:        strings.Replace(CodeStr(e.Code), ERROR_CODE_PREFIX, CONNECTOR_MGMT_ERROR_CODE_PREFIX, 1),
				Reason:      e.Reason,
				OperationId: "",
			},
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.AsOpenapiError(tt.args.operationID, tt.args.basePath)).To(Equal(tt.want))
		})
	}
}

func Test_StackTrace(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   errors.StackTrace
	}{
		{
			name: "should return error stacktrace if error cause is nil",
			fields: fields{
				err: &ServiceError{},
			},
			want: nil,
		},
		{
			name: "should return error stacktrace if cause defined",
			fields: fields{
				err: &ServiceError{
					cause: errorWithCause.cause,
				},
			},
			want: errorWithCause.StackTrace(),
		},
		{
			name: "should return nil if the cause doesn't have stacktrace",
			fields: fields{
				err: &ServiceError{
					cause: &errorWithoutStackTrace{},
				},
			},
			want: nil,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.StackTrace()).To(Equal(tt.want))
		})
	}
}

func Test_AsError(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   error
	}{
		{
			name: "should return a formatted error",
			fields: fields{
				err: &ServiceError{
					Code:     ServiceErrorCode(e.HttpCode),
					Reason:   e.Reason,
					HttpCode: e.HttpCode,
					cause:    e.cause,
				},
			},
			want: fmt.Errorf(e.Error()),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.AsError()).To(Equal(tt.want))
		})
	}
}

func Test_Error(t *testing.T) {
	errList := make(ErrorList, 10)
	var res string
	for _, err := range errList {
		res = res + fmt.Sprintf(";%s", err)
	}
	type fields struct {
		err ErrorList
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return a formatted error list",
			fields: fields{
				err: make(ErrorList, 10),
			},
			want: fmt.Sprintf("[%s]", res),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.Error()).To(Equal(tt.want))
		})
	}
}

func Test_Unwrap(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   error
	}{
		{
			name: "should return the original error that caused the ServiceError",
			fields: fields{
				err: errorWithCause,
			},
			want: errorWithCause.cause,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.Unwrap()).To(BeEquivalentTo(tt.want))
		})
	}
}

func Test_Error2(t *testing.T) {
	type fields struct {
		err *ServiceError
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return formatted error details",
			fields: fields{
				err: &ServiceError{},
			},
			want: fmt.Sprintf("%s: %s", CodeStr(e.Code), e.Reason),
		},
		{
			name: "should return formatted error details if the cause is not nil",
			fields: fields{
				err: errorWithCause,
			},
			want: fmt.Sprintf("%s: %s\n caused by: %s", CodeStr(errorWithCause.Code), errorWithCause.Reason, errorWithCause.cause.Error()),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Expect(tt.fields.err.Error()).To(BeEquivalentTo(tt.want))
		})
	}
}
