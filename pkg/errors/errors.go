package errors

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/compat"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

const (
	ERROR_CODE_PREFIX = "KAFKAS-MGMT"

	// HREF for API errors
	ERROR_HREF = "/api/kafkas_mgmt/v1/errors/"

	// To support connector errors too..
	CONNECTOR_MGMT_ERROR_CODE_PREFIX = "CONNECTOR-MGMT"
	CONNECTOR_MGMT_ERROR_HREF        = "/api/connector_mgmt/v1/errors/"

	// Forbidden occurs when a user is not allowed to access the service
	ErrorForbidden       ServiceErrorCode = 4
	ErrorForbiddenReason string           = "Forbidden to perform this action"

	// MaxAllowedInstanceReached occurs when a user or organisation has reached maximum number of allowed instances
	ErrorMaxAllowedInstanceReached       ServiceErrorCode = 5
	ErrorMaxAllowedInstanceReachedReason string           = "Forbidden to create more instances than the maximum allowed"

	// Conflict occurs when a database constraint is violated
	ErrorConflict       ServiceErrorCode = 6
	ErrorConflictReason string           = "An entity with the specified unique values already exists"

	// NotFound occurs when a record is not found in the database
	ErrorNotFound       ServiceErrorCode = 7
	ErrorNotFoundReason string           = "Resource not found"

	// Validation occurs when an object fails validation
	ErrorValidation       ServiceErrorCode = 8
	ErrorValidationReason string           = "General validation failure"

	// General occurs when an error fails to match any other error code
	ErrorGeneral       ServiceErrorCode = 9
	ErrorGeneralReason string           = "Unspecified error"

	// NotImplemented occurs when an API REST method is not implemented in a handler
	ErrorNotImplemented       ServiceErrorCode = 10
	ErrorNotImplementedReason string           = "HTTP Method not implemented for this endpoint"

	// Unauthorized occurs when the requester is not authorized to perform the specified action
	ErrorUnauthorized       ServiceErrorCode = 11
	ErrorUnauthorizedReason string           = "Account is unauthorized to perform this action"

	// Unauthorized occurs when the requester is not authorized to perform the specified action
	ErrorTermsNotAccepted       ServiceErrorCode = 12
	ErrorTermsNotAcceptedReason string           = "Required terms have not been accepted"

	// Unauthenticated occurs when the provided credentials cannot be validated
	ErrorUnauthenticated       ServiceErrorCode = 15
	ErrorUnauthenticatedReason string           = "Account authentication could not be verified"

	// MalformedRequest occurs when the request body cannot be read
	ErrorMalformedRequest       ServiceErrorCode = 17
	ErrorMalformedRequestReason string           = "Unable to read request body"

	// ServiceIsUnderMaintenance occurs when a user is unable to access the service due to ongoing maintenance
	ErrorServiceIsUnderMaintenance       ServiceErrorCode = 18
	ErrorServiceIsUnderMaintenanceReason string           = "Unable to perform this action, as the service is currently under maintenance"

	// Bad Request
	ErrorBadRequest       ServiceErrorCode = 21
	ErrorBadRequestReason string           = "Bad request"

	// Invalid Search Query
	ErrorFailedToParseSearch       ServiceErrorCode = 23
	ErrorFailedToParseSearchReason string           = "Failed to parse search query"

	// TooManyRequests occurs when a the kafka instances capacity gets filled up
	ErrorTooManyKafkaInstancesReached       ServiceErrorCode = 24
	ErrorTooManyKafkaInstancesReachedReason string           = "The maximum number of allowed kafka instances has been reached"

	// Gone occurs when a record is accessed that has been deleted
	ErrorGone       ServiceErrorCode = 25
	ErrorGoneReason string           = "Resource gone"

	// Synchronous request not supported
	ErrorSyncActionNotSupported       ServiceErrorCode = 103
	ErrorSyncActionNotSupportedReason string           = "Synchronous action is not supported, use async=true parameter"

	// Failed to create sso client - an internal error incurred when calling keycloak server
	ErrorFailedToCreateSSOClient       ServiceErrorCode = 106
	ErrorFailedToCreateSSOClientReason string           = "Failed to create kafka client in the mas sso"

	// Failed to get sso client secret  - an internal error incurred when calling keycloak server
	ErrorFailedToGetSSOClientSecret       ServiceErrorCode = 107
	ErrorFailedToGetSSOClientSecretReason string           = "Failed to get kafka client secret from the mas sso"

	// Failed to get sso client - an internal error incurred when calling keycloak server
	ErrorFailedToGetSSOClient       ServiceErrorCode = 108
	ErrorFailedToGetSSOClientReason string           = "Failed to get kafka client from the mas sso"

	// Failed to delete sso client - an internal error incurred when calling keycloak server
	ErrorFailedToDeleteSSOClient       ServiceErrorCode = 109
	ErrorFailedToDeleteSSOClientReason string           = "Failed to delete kafka client from the mas sso"

	// Failed to create service account, after validating user's request, but failed at the server end
	// it is an internal server error
	ErrorFailedToCreateServiceAccount       ServiceErrorCode = 110
	ErrorFailedToCreateServiceAccountReason string           = "Failed to create service account"

	// Failed to get service account - an internal error incurred when calling keycloak server
	ErrorFailedToGetServiceAccount       ServiceErrorCode = 111
	ErrorFailedToGetServiceAccountReason string           = "Failed to get service account"

	// Failed to delete service account - an internal error incurred when calling keycloak server
	ErrorFailedToDeleteServiceAccount       ServiceErrorCode = 112
	ErrorFailedToDeleteServiceAccountReason string           = "Failed to delete service account"

	// Failed to find service account - a client error as incorrect SA is given
	ErrorServiceAccountNotFound       ServiceErrorCode = 113
	ErrorServiceAccountNotFoundReason string           = "Failed to find service account"

	ErrorMaxLimitForServiceAccountsReached       ServiceErrorCode = 115
	ErrorMaxLimitForServiceAccountsReachedReason string           = "Max limit for the service account creation has reached"

	// Insufficient quota
	ErrorInsufficientQuota       ServiceErrorCode = 120
	ErrorInsufficientQuotaReason string           = "Insufficient quota"
	// Failed to check Quota
	ErrorFailedToCheckQuota       ServiceErrorCode = 121
	ErrorFailedToCheckQuotaReason string           = "Failed to check quota"
	// Provider not supported
	ErrorProviderNotSupported       ServiceErrorCode = 30
	ErrorProviderNotSupportedReason string           = "Provider not supported"

	// Region not supported
	ErrorRegionNotSupported       ServiceErrorCode = 31
	ErrorRegionNotSupportedReason string           = "Region not supported"

	// Invalid kafka cluster name
	ErrorMalformedKafkaClusterName       ServiceErrorCode = 32
	ErrorMalformedKafkaClusterNameReason string           = "Kafka cluster name is invalid"

	// Minimum field length validation
	ErrorMinimumFieldLength       ServiceErrorCode = 33
	ErrorMinimumFieldLengthReason string           = "Minimum field length not reached"

	// Maximum field length validation
	ErrorMaximumFieldLength       ServiceErrorCode = 34
	ErrorMaximumFieldLengthReason string           = "Maximum field length has been depassed"

	// Kafka cluster name must be unique
	ErrorDuplicateKafkaClusterName       ServiceErrorCode = 36
	ErrorDuplicateKafkaClusterNameReason string           = "Kafka cluster name is already used"

	// A generic field validation error when validating API requests input
	ErrorFieldValidationError       ServiceErrorCode = 37
	ErrorFieldValidationErrorReason string           = "Field validation failed"

	// Failure to send an error response (i.e. unable to send error response as the error can't be converted to JSON.)
	ErrorUnableToSendErrorResponse       ServiceErrorCode = 1000
	ErrorUnableToSendErrorResponseReason string           = "An unexpected error happened, please check the log of the service for details"

	// Invalid service account name
	ErrorMalformedServiceAccountName       ServiceErrorCode = 38
	ErrorMalformedServiceAccountNameReason string           = "Service account name is invalid"

	// Invalid service account desc
	ErrorMalformedServiceAccountDesc       ServiceErrorCode = 39
	ErrorMalformedServiceAccountDescReason string           = "Service account desc is invalid"

	// Invalid service account desc
	ErrorMalformedServiceAccountId       ServiceErrorCode = 40
	ErrorMalformedServiceAccountIdReason string           = "Service account id is invalid"

	// Instance type not supported
	ErrorInstanceTypeNotSupported       ServiceErrorCode = 41
	ErrorInstanceTypeNotSupportedReason string           = "Instance Type not supported"

	// Instance plan not supported
	ErrorInstancePlanNotSupported       ServiceErrorCode = 42
	ErrorInstancePlanNotSupportedReason string           = "Instance plan not supported"

	// Billing account id missing or invalid
	ErrorBillingAccountInvalid       ServiceErrorCode = 43
	ErrorBillingAccountInvalidReason string           = "Billing account id missing or invalid"

	// Enterprise cluster id must be unique
	ErrorDuplicateClusterId       ServiceErrorCode = 44
	ErrorDuplicateClusterIdReason string           = "Enterprise cluster ID is already used"

	// Enterprise cluster id is invalid
	ErrorInvalidClusterId       ServiceErrorCode = 45
	ErrorInvalidClusterIdReason string           = "Enterprise cluster ID is invalid"

	// Enterprise cluster external id is invalid
	ErrorInvalidExternalClusterId       ServiceErrorCode = 46
	ErrorInvalidExternalClusterIdReason string           = "Enterprise external cluster ID is invalid"

	// Dns name is invalid
	ErrorInvalidDnsName       ServiceErrorCode = 47
	ErrorInvalidDnsNameReason string           = "Dns name is invalid"

	// Too Many requests error. Used by rate limiting
	ErrorTooManyRequests       ServiceErrorCode = 429
	ErrorTooManyRequestsReason string           = "Too many requests"
)

type ErrorList []error

func (e ErrorList) Error() string {
	var res string

	firstElem := true
	for _, err := range e {
		if firstElem {
			res = err.Error()
			firstElem = false
		} else {
			res = fmt.Sprintf("%s;%s", res, err)
		}
	}

	res = fmt.Sprintf("[%s]", res)
	return res
}

func (e ErrorList) IsEmpty() bool {
	return len(e) == 0
}

// AddErrors adds the provided list of errors to the ErrorList.
// If the provided list of errors contain error elements that are of type
// ErrorList those are recursively "unrolled" so the result does not contain
// appended ErrorList elements.
// The method modifies the underlying slice.
func (e *ErrorList) AddErrors(errs ...error) {
	for _, err := range errs {
		var errList ErrorList
		if errors.As(err, &errList) {
			e.AddErrors(errList...)
		} else {
			*e = append(*e, err)
		}
	}
}

// ToErrorSlice returns the ErrorList as a slice of error
func (e ErrorList) ToErrorSlice() []error {
	return []error(e)
}

type ServiceErrorCode int

type ServiceErrors []ServiceError

func Find(code ServiceErrorCode) (bool, *ServiceError) {
	for _, err := range Errors() {
		if err.Code == code {
			return true, &err
		}
	}
	return false, nil
}

func Errors() ServiceErrors {
	return ServiceErrors{
		ServiceError{ErrorForbidden, ErrorForbiddenReason, http.StatusForbidden, nil, false},
		ServiceError{ErrorMaxAllowedInstanceReached, ErrorMaxAllowedInstanceReachedReason, http.StatusForbidden, nil, false},
		ServiceError{ErrorTooManyKafkaInstancesReached, ErrorTooManyKafkaInstancesReachedReason, http.StatusForbidden, nil, false},
		ServiceError{ErrorTooManyRequests, ErrorTooManyRequestsReason, http.StatusTooManyRequests, nil, false},
		ServiceError{ErrorConflict, ErrorConflictReason, http.StatusConflict, nil, false},
		ServiceError{ErrorNotFound, ErrorNotFoundReason, http.StatusNotFound, nil, false},
		ServiceError{ErrorGone, ErrorGoneReason, http.StatusGone, nil, false},
		ServiceError{ErrorValidation, ErrorValidationReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorGeneral, ErrorGeneralReason, http.StatusInternalServerError, nil, false},
		ServiceError{ErrorNotImplemented, ErrorNotImplementedReason, http.StatusMethodNotAllowed, nil, false},
		ServiceError{ErrorUnauthorized, ErrorUnauthorizedReason, http.StatusForbidden, nil, false},
		ServiceError{ErrorTermsNotAccepted, ErrorTermsNotAcceptedReason, http.StatusForbidden, nil, false},
		ServiceError{ErrorUnauthenticated, ErrorUnauthenticatedReason, http.StatusUnauthorized, nil, false},
		ServiceError{ErrorMalformedRequest, ErrorMalformedRequestReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorServiceIsUnderMaintenance, ErrorServiceIsUnderMaintenanceReason, http.StatusForbidden, nil, false},
		ServiceError{ErrorBadRequest, ErrorBadRequestReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorFailedToParseSearch, ErrorFailedToParseSearchReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorSyncActionNotSupported, ErrorSyncActionNotSupportedReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorFailedToCreateSSOClient, ErrorFailedToCreateSSOClientReason, http.StatusInternalServerError, nil, false},
		ServiceError{ErrorFailedToGetSSOClientSecret, ErrorFailedToGetSSOClientSecretReason, http.StatusInternalServerError, nil, false},
		ServiceError{ErrorFailedToGetSSOClient, ErrorFailedToGetSSOClientReason, http.StatusInternalServerError, nil, false},
		ServiceError{ErrorFailedToDeleteSSOClient, ErrorFailedToDeleteSSOClientReason, http.StatusInternalServerError, nil, false},
		ServiceError{ErrorFailedToCreateServiceAccount, ErrorFailedToCreateServiceAccountReason, http.StatusInternalServerError, nil, false},
		ServiceError{ErrorFailedToGetServiceAccount, ErrorFailedToGetServiceAccountReason, http.StatusInternalServerError, nil, false},
		ServiceError{ErrorServiceAccountNotFound, ErrorServiceAccountNotFoundReason, http.StatusNotFound, nil, false},
		ServiceError{ErrorFailedToDeleteServiceAccount, ErrorFailedToDeleteServiceAccountReason, http.StatusInternalServerError, nil, false},
		ServiceError{ErrorProviderNotSupported, ErrorProviderNotSupportedReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorRegionNotSupported, ErrorRegionNotSupportedReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorInstanceTypeNotSupported, ErrorInstanceTypeNotSupportedReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorMalformedKafkaClusterName, ErrorMalformedKafkaClusterNameReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorMinimumFieldLength, ErrorMinimumFieldLengthReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorMaximumFieldLength, ErrorMaximumFieldLengthReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorDuplicateKafkaClusterName, ErrorDuplicateKafkaClusterNameReason, http.StatusConflict, nil, false},
		ServiceError{ErrorUnableToSendErrorResponse, ErrorUnableToSendErrorResponseReason, http.StatusInternalServerError, nil, false},
		ServiceError{ErrorFieldValidationError, ErrorFieldValidationErrorReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorInsufficientQuota, ErrorInsufficientQuotaReason, http.StatusForbidden, nil, false},
		ServiceError{ErrorFailedToCheckQuota, ErrorFailedToCheckQuotaReason, http.StatusInternalServerError, nil, false},
		ServiceError{ErrorMalformedServiceAccountName, ErrorMalformedServiceAccountNameReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorMalformedServiceAccountDesc, ErrorMalformedServiceAccountDescReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorMalformedServiceAccountId, ErrorMalformedServiceAccountIdReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorMaxLimitForServiceAccountsReached, ErrorMaxLimitForServiceAccountsReachedReason, http.StatusForbidden, nil, false},
		ServiceError{ErrorInstancePlanNotSupported, ErrorInstancePlanNotSupportedReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorBillingAccountInvalid, ErrorBillingAccountInvalidReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorDuplicateClusterId, ErrorDuplicateClusterIdReason, http.StatusConflict, nil, false},
		ServiceError{ErrorInvalidClusterId, ErrorInvalidClusterIdReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorInvalidExternalClusterId, ErrorInvalidExternalClusterIdReason, http.StatusBadRequest, nil, false},
		ServiceError{ErrorInvalidDnsName, ErrorInvalidDnsNameReason, http.StatusBadRequest, nil, false},
	}
}

func NewErrorFromHTTPStatusCode(httpCode int, reason string, values ...interface{}) *ServiceError {
	if httpCode >= http.StatusBadRequest && httpCode < http.StatusInternalServerError {
		switch httpCode {
		case http.StatusUnauthorized:
			return Unauthorized(reason, values...)
		case http.StatusForbidden:
			return Forbidden(reason, values...)
		case http.StatusNotFound:
			return NotFound(reason, values...)
		case http.StatusMethodNotAllowed:
			return NotImplemented(reason, values...)
		case http.StatusConflict:
			return Conflict(reason, values...)
		//StatusBadRequest and all other errors will result in BadRequest error being created
		default:
			return BadRequest(reason, values...)
		}
	}

	if httpCode >= http.StatusInternalServerError {
		switch httpCode {
		//StatusInternalServerError and all other errors will result in GeneralError() error being created
		default:
			return GeneralError(reason, values...)
		}
	}

	return GeneralError(reason, values...)
}

func ToServiceError(err error) *ServiceError {
	switch convertedErr := err.(type) {
	case *ServiceError:
		return convertedErr
	default:
		return GeneralError(convertedErr.Error())
	}
}

type ServiceErrorBuilder interface {
	Wrap(serviceError ServiceError) ServiceErrorBuilder
	WithCode(code ServiceErrorCode) ServiceErrorBuilder
	WithReason(reason string, values ...any) ServiceErrorBuilder
	WithHttpCode(httpCode int) ServiceErrorBuilder
	WithCause(cause error) ServiceErrorBuilder
	Recoverable() ServiceErrorBuilder
	Build() *ServiceError
}

var _ ServiceErrorBuilder = &serviceErrorBuilder{}

func NewServiceErrorBuilder() ServiceErrorBuilder {
	return &serviceErrorBuilder{}
}

type serviceErrorBuilder struct {
	ServiceError
}

func (seb *serviceErrorBuilder) Wrap(serviceError ServiceError) ServiceErrorBuilder {
	seb.WithReason(serviceError.Reason).
		WithCode(serviceError.Code).
		WithCause(&serviceError)
	return seb
}

func (seb *serviceErrorBuilder) WithCode(code ServiceErrorCode) ServiceErrorBuilder {
	seb.Code = code
	return seb
}

func (seb *serviceErrorBuilder) WithReason(reason string, values ...any) ServiceErrorBuilder {
	seb.Reason = fmt.Sprintf(reason, values...)
	return seb
}

func (seb *serviceErrorBuilder) WithHttpCode(httpCode int) ServiceErrorBuilder {
	seb.HttpCode = httpCode
	return seb
}

func (seb *serviceErrorBuilder) WithCause(cause error) ServiceErrorBuilder {
	seb.cause = cause
	return seb
}

func (seb *serviceErrorBuilder) Recoverable() ServiceErrorBuilder {
	seb.recoverable = true
	return seb
}

func (seb *serviceErrorBuilder) Build() *ServiceError {
	err := NewWithCause(seb.Code, seb.cause, seb.Error(), seb.Reason)
	err.recoverable = seb.recoverable
	return err
}

type ServiceError struct {
	// Code is the numeric and distinct ID for the error
	Code ServiceErrorCode
	// Reason is the context-specific reason the error was generated
	Reason string
	// HttpCode is the HttpCode associated with the error when the error is returned as an API response
	HttpCode int
	// cause - The original error that is causing the ServiceError, can be used for inspection
	cause error
	// recoverable - if true, retrying the operation could succeed
	recoverable bool
}

// Reason can be a string with format verbs, which will be replace by the specified values
func New(code ServiceErrorCode, reason string, values ...interface{}) *ServiceError {
	return NewWithCause(code, nil, reason, values...)
}

func NewWithCause(code ServiceErrorCode, cause error, reason string, values ...interface{}) *ServiceError {
	// If the code isn't defined, use the general error code
	var err *ServiceError
	exists, err := Find(code)
	if !exists {
		glog.Errorf("Undefined error code used: %d", code)
		err = &ServiceError{ErrorGeneral, "unspecified error", http.StatusInternalServerError, nil, false}
	}

	// TODO - if cause is nil, should we use the reason as the cause?
	if cause != nil {
		_, ok := cause.(stackTracer)
		if !ok {
			cause = errors.WithStack(cause) // add stacktrace info
		}
	}
	err.cause = cause

	// If the reason is unspecified, use the default
	if reason != "" {
		err.Reason = fmt.Sprintf(reason, values...)
	}

	return err
}

// Unwrap returns the original error that caused the ServiceError. Can be used with errors.Unwrap.
func (e *ServiceError) Unwrap() error {
	return e.cause
}

// StackTrace returns errors stacktrace.
func (e *ServiceError) StackTrace() errors.StackTrace {
	if e.cause == nil {
		return nil
	}

	err, ok := e.cause.(stackTracer)
	if !ok {
		return nil
	}

	return err.StackTrace()
}

func (e *ServiceError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s\n caused by: %s", CodeStr(e.Code), e.Reason, e.cause.Error())
	}

	return fmt.Sprintf("%s: %s", CodeStr(e.Code), e.Reason)
}

func (e *ServiceError) Recoverable() bool {
	return e.recoverable
}
func (e *ServiceError) AsError() error {
	return fmt.Errorf(e.Error())
}

func (e *ServiceError) Is404() bool {
	return e.Code == NotFound("").Code
}

func (e *ServiceError) IsConflict() bool {
	return e.Code == Conflict("").Code
}

func (e *ServiceError) IsForbidden() bool {
	return e.Code == Forbidden("").Code
}

func (e *ServiceError) IsFailedToCreateSSOClient() bool {
	return e.Code == FailedToCreateSSOClient("").Code
}

func (e *ServiceError) IsClientErrorClass() bool {
	return e.HttpCode >= http.StatusBadRequest && e.HttpCode < http.StatusInternalServerError
}

func (e *ServiceError) IsServerErrorClass() bool {
	return e.HttpCode >= http.StatusInternalServerError
}

func (e *ServiceError) IsFailedToGetSSOClientSecret() bool {
	return e.Code == FailedToGetSSOClientSecret("").Code
}

func (e *ServiceError) IsFailedToGetSSOClient() bool {
	return e.Code == FailedToGetSSOClient("").Code
}

func (e *ServiceError) IsFailedToDeleteSSOClient() bool {
	return e.Code == FailedToDeleteSSOClient("").Code
}

func (e *ServiceError) IsFailedToCreateServiceAccount() bool {
	return e.Code == FailedToCreateServiceAccount("").Code
}

func (e *ServiceError) IsFailedToGetServiceAccount() bool {
	return e.Code == FailedToGetServiceAccount("").Code
}

func (e *ServiceError) IsFailedToDeleteServiceAccount() bool {
	return e.Code == FailedToDeleteServiceAccount("").Code
}

func (e *ServiceError) IsServiceAccountNotFound() bool {
	return e.Code == ServiceAccountNotFound("").Code
}

func (e *ServiceError) IsMaxLimitForServiceAccountReached() bool {
	return e.Code == ErrorMaxLimitForServiceAccountsReached
}

func (e *ServiceError) IsBadRequest() bool {
	return e.Code == BadRequest("").Code
}
func (e *ServiceError) InSufficientQuota() bool {
	return e.Code == InsufficientQuotaError("").Code
}

func (e *ServiceError) IsFailedToCheckQuota() bool {
	return e.Code == FailedToCheckQuota("").Code
}

func (e *ServiceError) IsInstanceTypeNotSupported() bool {
	return e.Code == ErrorInstanceTypeNotSupported
}

func (e *ServiceError) AsOpenapiError(operationID string, basePath string) compat.Error {
	href := Href(e.Code)
	code := CodeStr(e.Code)

	if strings.Contains(basePath, "/api/connector_mgmt/") {
		href = strings.Replace(href, ERROR_HREF, CONNECTOR_MGMT_ERROR_HREF, 1)
		code = strings.Replace(code, ERROR_CODE_PREFIX, CONNECTOR_MGMT_ERROR_CODE_PREFIX, 1)
	}

	// end-temporary code
	return compat.Error{
		Kind:        "Error",
		Id:          strconv.Itoa(int(e.Code)),
		Href:        href,
		Code:        code,
		Reason:      e.Reason,
		OperationId: operationID,
	}
}

func CodeStr(code ServiceErrorCode) string {
	return fmt.Sprintf("%s-%d", ERROR_CODE_PREFIX, code)
}

func Href(code ServiceErrorCode) string {
	return fmt.Sprintf("%s%d", ERROR_HREF, code)
}

func NotFound(reason string, values ...interface{}) *ServiceError {
	return New(ErrorNotFound, reason, values...)
}

func GeneralError(reason string, values ...interface{}) *ServiceError {
	return New(ErrorGeneral, reason, values...)
}

func Unauthorized(reason string, values ...interface{}) *ServiceError {
	return New(ErrorUnauthorized, reason, values...)
}

func TermsNotAccepted(reason string, values ...interface{}) *ServiceError {
	return New(ErrorTermsNotAccepted, reason, values...)
}

func Unauthenticated(reason string, values ...interface{}) *ServiceError {
	return New(ErrorUnauthenticated, reason, values...)
}

func Forbidden(reason string, values ...interface{}) *ServiceError {
	return New(ErrorForbidden, reason, values...)
}

func Maintenance(reason string, values ...interface{}) *ServiceError {
	return New(ErrorServiceIsUnderMaintenance, reason, values...)
}

func MaximumAllowedInstanceReached(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMaxAllowedInstanceReached, reason, values...)
}

func TooManyKafkaInstancesReached(reason string, values ...interface{}) *ServiceError {
	return New(ErrorTooManyKafkaInstancesReached, reason, values...)
}

func NotImplemented(reason string, values ...interface{}) *ServiceError {
	return New(ErrorNotImplemented, reason, values...)
}

func Conflict(reason string, values ...interface{}) *ServiceError {
	return New(ErrorConflict, reason, values...)
}

func Validation(reason string, values ...interface{}) *ServiceError {
	return New(ErrorValidation, reason, values...)
}

func MalformedRequest(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedRequest, reason, values...)
}

func BadRequest(reason string, values ...interface{}) *ServiceError {
	return New(ErrorBadRequest, reason, values...)
}

func FailedToParseSearch(reason string, values ...interface{}) *ServiceError {
	message := fmt.Sprintf("%s: %s", ErrorFailedToParseSearchReason, reason)
	return New(ErrorFailedToParseSearch, message, values...)
}

func SyncActionNotSupported() *ServiceError {
	return New(ErrorSyncActionNotSupported, ErrorSyncActionNotSupportedReason)
}

func FailedToCreateSSOClient(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToCreateSSOClient, reason, values...)
}

func FailedToGetSSOClientSecret(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToGetSSOClientSecret, reason, values...)
}

func FailedToGetSSOClient(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToGetSSOClient, reason, values...)
}

func FailedToDeleteSSOClient(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToDeleteSSOClient, reason, values...)
}

func FailedToCreateServiceAccount(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToCreateServiceAccount, reason, values...)
}

func FailedToDeleteServiceAccount(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToDeleteServiceAccount, reason, values...)
}

func MaxLimitForServiceAccountReached(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMaxLimitForServiceAccountsReached, reason, values...)
}

func FailedToGetServiceAccount(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToGetServiceAccount, reason, values...)
}

func ServiceAccountNotFound(reason string, values ...interface{}) *ServiceError {
	return New(ErrorServiceAccountNotFound, reason, values...)
}

func RegionNotSupported(reason string, values ...interface{}) *ServiceError {
	return New(ErrorRegionNotSupported, reason, values...)
}

func InstanceTypeNotSupported(reason string, values ...interface{}) *ServiceError {
	return New(ErrorInstanceTypeNotSupported, reason, values...)
}

func ProviderNotSupported(reason string, values ...interface{}) *ServiceError {
	return New(ErrorProviderNotSupported, reason, values...)
}

func InstancePlanNotSupported(reason string, values ...interface{}) *ServiceError {
	return New(ErrorInstancePlanNotSupported, reason, values...)
}

func MalformedKafkaClusterName(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedKafkaClusterName, reason, values...)
}

func MalformedServiceAccountName(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedServiceAccountName, reason, values...)
}

func MalformedServiceAccountDesc(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedServiceAccountDesc, reason, values...)
}

func MalformedServiceAccountId(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedServiceAccountId, reason, values...)
}

func InvalidExternalClusterId(reason string, values ...interface{}) *ServiceError {
	return New(ErrorInvalidExternalClusterId, reason, values...)
}

func InvalidClusterId(reason string, values ...interface{}) *ServiceError {
	return New(ErrorInvalidClusterId, reason, values...)
}

func InvalidDnsName(reason string, values ...interface{}) *ServiceError {
	return New(ErrorInvalidDnsName, reason, values...)
}

func DuplicateKafkaClusterName() *ServiceError {
	return New(ErrorDuplicateKafkaClusterName, ErrorDuplicateKafkaClusterNameReason)
}

func DuplicateClusterId() *ServiceError {
	return New(ErrorDuplicateClusterId, ErrorDuplicateClusterIdReason)
}

func MinimumFieldLengthNotReached(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMinimumFieldLength, reason, values...)
}

func MaximumFieldLengthExceeded(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMaximumFieldLength, reason, values...)
}

func UnableToSendErrorResponse() *ServiceError {
	return New(ErrorUnableToSendErrorResponse, ErrorUnableToSendErrorResponseReason)
}

func FailedToParseQueryParms(reason string, values ...interface{}) *ServiceError {
	return New(ErrorBadRequest, reason, values...)

}

func FieldValidationError(reason string, values ...interface{}) *ServiceError {
	message := fmt.Sprintf("%s: %s", ErrorFieldValidationErrorReason, reason)

	return New(ErrorFieldValidationError, message, values...)
}

func InsufficientQuotaError(reason string, values ...interface{}) *ServiceError {
	message := fmt.Sprintf("%s: %s", ErrorInsufficientQuotaReason, reason)
	return New(ErrorInsufficientQuota, message, values...)
}

func FailedToCheckQuota(reason string, values ...interface{}) *ServiceError {
	message := fmt.Sprintf("%s: %s", ErrorFailedToCheckQuotaReason, reason)
	return New(ErrorFailedToCheckQuota, message, values...)
}

func InvalidBillingAccount(reason string, values ...interface{}) *ServiceError {
	message := fmt.Sprintf("%s: %s", ErrorBillingAccountInvalidReason, reason)
	return New(ErrorBillingAccountInvalid, message, values...)
}
