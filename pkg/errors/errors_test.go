package errors

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/compat"
	"github.com/onsi/gomega"
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
	g := gomega.NewWithT(t)
	err := New(ErrorGeneral, "test %s, %d", "errors", 1)
	g.Expect(err.Reason).To(gomega.Equal("test errors, 1"))
}

func TestErrorFind(t *testing.T) {
	g := gomega.NewWithT(t)
	exists, err := Find(ErrorNotFound)
	g.Expect(exists).To(gomega.Equal(true))
	g.Expect(err.Code).To(gomega.Equal(ErrorNotFound))

	// Hopefully we never reach 91,823,719 error codes or this test will fail
	exists, err = Find(ServiceErrorCode(91823719))
	g.Expect(exists).To(gomega.Equal(false))
	g.Expect(err).To(gomega.BeNil())
}

func Test_NewErrorFromHTTPStatusCode(t *testing.T) {
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
				reason:   genericErrorMessage,
			},
			want: BadRequest(genericErrorMessage),
		},
		{
			name: "should return unauthorised error",
			args: args{
				httpCode: http.StatusUnauthorized,
				reason:   genericErrorMessage,
			},
			want: Unauthorized(genericErrorMessage),
		},
		{
			name: "should return forbidden error",
			args: args{
				httpCode: http.StatusForbidden,
				reason:   genericErrorMessage,
			},
			want: Forbidden(genericErrorMessage),
		},
		{
			name: "should return not found error",
			args: args{
				httpCode: http.StatusNotFound,
				reason:   genericErrorMessage,
			},
			want: NotFound(genericErrorMessage),
		},
		{
			name: "should return not implemented error",
			args: args{
				httpCode: http.StatusMethodNotAllowed,
				reason:   genericErrorMessage,
			},
			want: NotImplemented(genericErrorMessage),
		},
		{
			name: "should return conflict error",
			args: args{
				httpCode: http.StatusConflict,
				reason:   genericErrorMessage,
			},
			want: Conflict(genericErrorMessage),
		},
		{
			name: "should return general error",
			args: args{
				httpCode: http.StatusInternalServerError,
				reason:   genericErrorMessage,
			},
			want: GeneralError(genericErrorMessage),
		},
		{
			name: "general case, should return general error",
			want: GeneralError("Unspecified error"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(NewErrorFromHTTPStatusCode(tt.args.httpCode, tt.args.reason)).To(gomega.MatchError(tt.want))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			err := NewWithCause(tt.args.code, tt.args.cause, tt.args.reason)
			g.Expect(err.Code).To(gomega.Equal(tt.want.Code))
			g.Expect(err.Reason).To(gomega.Equal(tt.want.Reason))
			g.Expect(err.HttpCode).To(gomega.Equal(tt.want.HttpCode))
			if err.cause != nil {
				_, ok := err.cause.(stackTracer)
				g.Expect((ok)).To(gomega.BeTrue())
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
			name: "should return new ErrorFailedToCreateSSOClient error",
			args: args{
				reason: "Failed to create kafka client in the mas sso",
			},
			want: New(ErrorFailedToCreateSSOClient, "Failed to create kafka client in the mas sso"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FailedToCreateSSOClient(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorFailedToGetSSOClientSecret error",
			args: args{
				reason: "Failed to get kafka client secret from the mas sso",
			},
			want: New(ErrorFailedToGetSSOClientSecret, "Failed to get kafka client secret from the mas sso"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FailedToGetSSOClientSecret(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorFailedToGetSSOClient error",
			args: args{
				reason: "Failed to get kafka client from the mas sso",
			},
			want: New(ErrorFailedToGetSSOClient, "Failed to get kafka client from the mas sso"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FailedToGetSSOClient(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorFailedToDeleteSSOClient error",
			args: args{
				reason: "Failed to delete kafka client from the mas sso",
			},
			want: New(ErrorFailedToDeleteSSOClient, "Failed to delete kafka client from the mas sso"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FailedToDeleteSSOClient(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorFailedToCreateServiceAccount error",
			args: args{
				reason: "Failed to create service account",
			},
			want: New(ErrorFailedToCreateServiceAccount, "Failed to create service account"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FailedToCreateServiceAccount(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorFailedToDeleteServiceAccount error",
			args: args{
				reason: "Failed to delete service account",
			},
			want: New(ErrorFailedToDeleteServiceAccount, "Failed to delete service account"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FailedToDeleteServiceAccount(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorMaxLimitForServiceAccountsReached error",
			args: args{
				reason: "Max limit for the service account creation has reached",
			},
			want: New(ErrorMaxLimitForServiceAccountsReached, "Max limit for the service account creation has reached"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(MaxLimitForServiceAccountReached(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorFailedToGetServiceAccount error",
			args: args{
				reason: "Failed to get service account",
			},
			want: New(ErrorFailedToGetServiceAccount, "Failed to get service account"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FailedToGetServiceAccount(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorServiceAccountNotFound error",
			args: args{
				reason: "Failed to find service account",
			},
			want: New(ErrorServiceAccountNotFound, "Failed to find service account"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(ServiceAccountNotFound(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorRegionNotSupported error",
			args: args{
				reason: "Region not supported",
			},
			want: New(ErrorRegionNotSupported, "Region not supported"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(RegionNotSupported(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorInstanceTypeNotSupported error",
			args: args{
				reason: "Instance Type not supported",
			},
			want: New(ErrorInstanceTypeNotSupported, "Instance Type not supported"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(InstanceTypeNotSupported(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorProviderNotSupported error",
			args: args{
				reason: "Provider not supported",
			},
			want: New(ErrorProviderNotSupported, "Provider not supported"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(ProviderNotSupported(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorInstancePlanNotSupported error",
			args: args{
				reason: "Instance plan not supported",
			},
			want: New(ErrorInstancePlanNotSupported, "Instance plan not supported"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(InstancePlanNotSupported(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorMalformedKafkaClusterName error",
			args: args{
				reason: "Kafka cluster name is invalid",
			},
			want: New(ErrorMalformedKafkaClusterName, "Kafka cluster name is invalid"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(MalformedKafkaClusterName(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorMalformedServiceAccountName error",
			args: args{
				reason: "Service account name is invalid",
			},
			want: New(ErrorMalformedServiceAccountName, "Service account name is invalid"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(MalformedServiceAccountName(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorMalformedServiceAccountDesc error",
			args: args{
				reason: "Service account desc is invalid",
			},
			want: New(ErrorMalformedServiceAccountDesc, "Service account desc is invalid"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(MalformedServiceAccountDesc(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorMalformedServiceAccountId error",
			args: args{
				reason: "Service account id is invalid",
			},
			want: New(ErrorMalformedServiceAccountId, "Service account id is invalid"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(MalformedServiceAccountId(tt.args.reason)).To(gomega.MatchError(tt.want))
		})
	}
}

func Test_DuplicateKafkaClusterName(t *testing.T) {
	tests := []struct {
		name string
		want *ServiceError
	}{
		{
			name: "should return new ErrorDuplicateKafkaClusterName error",
			want: New(ErrorDuplicateKafkaClusterName, ErrorDuplicateKafkaClusterNameReason),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(DuplicateKafkaClusterName()).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorMinimumFieldLength error",
			args: args{
				reason: "Minimum field length not reached",
			},
			want: New(ErrorMinimumFieldLength, "Minimum field length not reached"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(MinimumFieldLengthNotReached(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new MaximumFieldLength error",
			args: args{
				reason: "Maximum field length has been depassed",
			},
			want: New(ErrorMaximumFieldLength, "Maximum field length has been depassed"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(MaximumFieldLengthExceeded(tt.args.reason)).To(gomega.MatchError(tt.want))
		})
	}
}

func Test_UnableToSendErrorResponse(t *testing.T) {
	tests := []struct {
		name string
		want *ServiceError
	}{
		{
			name: "should return new ErrorUnableToSendErrorResponse error",
			want: New(ErrorUnableToSendErrorResponse, ErrorUnableToSendErrorResponseReason),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(UnableToSendErrorResponse()).To(gomega.MatchError(tt.want))
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
			name: "should return new BadRequest error",
			args: args{
				reason: "Bad request",
			},
			want: New(ErrorBadRequest, "Bad request"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FailedToParseQueryParms(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return a new service error if ErrorFieldValidationError is returned",
			want: New(ErrorFieldValidationError, "Field validation failed: "),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FieldValidationError(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return a new service error if ErrorInsufficientQuota is returned",
			want: New(ErrorInsufficientQuota, "Insufficient quota: "),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(InsufficientQuotaError(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorFailedToCheckQuota error",
			want: New(ErrorFailedToCheckQuota, "Failed to check quota: "),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FailedToCheckQuota(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorBillingAccountInvalid error",
			want: New(ErrorBillingAccountInvalid, "Billing account id missing or invalid: "),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(InvalidBillingAccount(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorTermsNotAccepted error",
			args: args{
				reason: "Required terms have not been accepted",
			},
			want: New(ErrorTermsNotAccepted, "Required terms have not been accepted"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(TermsNotAccepted(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorUnauthenticated error",
			args: args{
				reason: "Account authentication could not be verified",
			},
			want: New(ErrorUnauthenticated, "Account authentication could not be verified"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(Unauthenticated(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorServiceIsUnderMaintenance error",
			args: args{
				reason: "Unable to perform this action, as the service is currently under maintenance",
			},
			want: New(ErrorServiceIsUnderMaintenance, "Unable to perform this action, as the service is currently under maintenance"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(Maintenance(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorMaxAllowedInstanceReached error",
			args: args{
				reason: "Forbidden to create more instances than the maximum allowed",
			},
			want: New(ErrorMaxAllowedInstanceReached, "Forbidden to create more instances than the maximum allowed"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(MaximumAllowedInstanceReached(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorTooManyKafkaInstancesReached error",
			args: args{
				reason: "The maximum number of allowed kafka instances has been reached",
			},
			want: New(ErrorTooManyKafkaInstancesReached, "The maximum number of allowed kafka instances has been reached"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(TooManyKafkaInstancesReached(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorValidation error",
			args: args{
				reason: "General validation failure",
			},
			want: New(ErrorValidation, "General validation failure"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(Validation(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorMalformedRequest error",
			args: args{
				reason: "Unable to read request body",
			},
			want: New(ErrorMalformedRequest, "Unable to read request body"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(MalformedRequest(tt.args.reason)).To(gomega.MatchError(tt.want))
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
			name: "should return new ErrorFailedToParseSearch error",
			want: New(ErrorFailedToParseSearch, "Failed to parse search query: "),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(FailedToParseSearch(tt.args.reason)).To(gomega.MatchError(tt.want))
		})
	}
}

func Test_SyncActionNotSupported(t *testing.T) {
	tests := []struct {
		name string
		want *ServiceError
	}{
		{
			name: "should return new ErrorSyncActionNotSupported error",
			want: New(ErrorSyncActionNotSupported, ErrorSyncActionNotSupportedReason),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(SyncActionNotSupported()).To(gomega.MatchError(tt.want))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(CodeStr(tt.args.code)).To(gomega.Equal(tt.want))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(Href(tt.args.code)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ToServiceError(t *testing.T) {
	g := gomega.NewWithT(t)
	sampleNonServiceError, err := ocmErrors.NewError().Reason("Unspecified error").Build()
	g.Expect(err).To(gomega.BeNil())
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want *ServiceError
	}{
		{
			name: "should return a service error if a service error occurred",
			args: args{
				err: BadRequest(""),
			},
			want: BadRequest(""),
		},
		{
			name: "should convert non-service error to service error and return it",
			args: args{
				err: sampleNonServiceError,
			},
			want: GeneralError(""),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(ToServiceError(tt.args.err)).To(gomega.MatchError(tt.want))
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
			name: "should return false if the error Is404() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error Is404()",
			fields: fields{
				err: &ServiceError{
					Code: NotFound("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.Is404()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsConflict() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsConflict()",
			fields: fields{
				err: &ServiceError{
					Code: Conflict("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsConflict()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsForbidden() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsForbidden()",
			fields: fields{
				err: &ServiceError{
					Code: Forbidden("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsForbidden()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsFailedToCreateSSOClient() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsFailedToCreateSSOClient()",
			fields: fields{
				err: &ServiceError{
					Code: FailedToCreateSSOClient("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsFailedToCreateSSOClient()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsFailedToGetSSOClientSecret() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsFailedToGetSSOClientSecret()",
			fields: fields{
				err: &ServiceError{
					Code: FailedToGetSSOClientSecret("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsFailedToGetSSOClientSecret()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsFailedToGetSSOClient() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsFailedToGetSSOClient()",
			fields: fields{
				err: &ServiceError{
					Code: FailedToGetSSOClient("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsFailedToGetSSOClient()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsFailedToDeleteSSOClient() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsFailedToDeleteSSOClient()",
			fields: fields{
				err: &ServiceError{
					Code: FailedToDeleteSSOClient("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsFailedToDeleteSSOClient()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsFailedToCreateServiceAccount() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsFailedToCreateServiceAccount()",
			fields: fields{
				err: &ServiceError{
					Code: FailedToCreateServiceAccount("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsFailedToCreateServiceAccount()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsFailedToGetServiceAccount() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsFailedToGetServiceAccount()",
			fields: fields{
				err: &ServiceError{
					Code: FailedToGetServiceAccount("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsFailedToGetServiceAccount()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsFailedToDeleteServiceAccount() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsFailedToDeleteServiceAccount()",
			fields: fields{
				err: &ServiceError{
					Code: FailedToDeleteServiceAccount("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsFailedToDeleteServiceAccount()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsServiceAccountNotFound() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsServiceAccountNotFound()",
			fields: fields{
				err: &ServiceError{
					Code: ServiceAccountNotFound("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsServiceAccountNotFound()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsMaxLimitForServiceAccountReached() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsMaxLimitForServiceAccountReached()",
			fields: fields{
				err: &ServiceError{
					Code: ErrorMaxLimitForServiceAccountsReached,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsMaxLimitForServiceAccountReached()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsBadRequest() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsBadRequest()",
			fields: fields{
				err: &ServiceError{
					Code: BadRequest("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsBadRequest()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error is InSufficientQuota() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error is InSufficientQuota()",
			fields: fields{
				err: &ServiceError{
					Code: InsufficientQuotaError("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.InSufficientQuota()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsFailedToCheckQuota() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsFailedToCheckQuota()",
			fields: fields{
				err: &ServiceError{
					Code: FailedToCheckQuota("").Code,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsFailedToCheckQuota()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsInstanceTypeNotSupported() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsInstanceTypeNotSupported()",
			fields: fields{
				err: &ServiceError{
					Code: ErrorInstanceTypeNotSupported,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsInstanceTypeNotSupported()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsClientErrorClass() code does not match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsClientErrorClass() code is BadRequest",
			fields: fields{
				err: &ServiceError{
					HttpCode: http.StatusBadRequest,
				},
			},
			want: true,
		},
		{
			name: "should return true if the error IsClientErrorClass() code is Conflict",
			fields: fields{
				err: &ServiceError{
					HttpCode: http.StatusConflict,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsClientErrorClass()).To(gomega.Equal(tt.want))
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
			name: "should return false if the error IsServerErrorClass() code doesn't match",
			fields: fields{
				err: &e,
			},
			want: false,
		},
		{
			name: "should return true if the error IsServerErrorClass() code is InternalServerError",
			fields: fields{
				err: &ServiceError{
					HttpCode: http.StatusInternalServerError,
				},
			},
			want: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.IsServerErrorClass()).To(gomega.Equal(tt.want))
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
				err: &e,
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
			name: "should return properly formatted compat.Error when basePath is specified",
			fields: fields{
				err: &e,
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.AsOpenapiError(tt.args.operationID, tt.args.basePath)).To(gomega.Equal(tt.want))
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
				err: &e,
			},
			want: nil,
		},
		{
			name: "should return error stacktrace if cause is defined",
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.StackTrace()).To(gomega.Equal(tt.want))
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
				err: &e,
			},
			want: fmt.Errorf(e.Error()),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.AsError()).To(gomega.MatchError(tt.want))
		})
	}
}

func Test_ErrorListToString(t *testing.T) {
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
				err: errList,
			},
			want: fmt.Sprintf("[%s]", res),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.Error()).To(gomega.Equal(tt.want))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.Unwrap()).To(gomega.MatchError(tt.want))
		})
	}
}

func Test_ErrorToString(t *testing.T) {
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
				err: &e,
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.err.Error()).To(gomega.BeEquivalentTo(tt.want))
		})
	}
}
