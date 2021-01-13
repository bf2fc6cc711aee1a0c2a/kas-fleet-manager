package errors

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang/glog"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
)

const (
	ERROR_CODE_PREFIX = "MGD-SERV-API"

	// HREF for API errors
	ERROR_HREF = "/api/managed-services-api/v1/errors/"

	// Forbidden occurs when a user is not allowed to access the service
	ErrorForbidden       ServiceErrorCode = 4
	ErrorForbiddenReason string           = "Forbidden to perform this action"

	// Forbidden occurs when a user or organisation has reached maximum number of allowed instances
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

	// Unauthenticated occurs when the provided credentials cannot be validated
	ErrorUnauthenticated       ServiceErrorCode = 15
	ErrorUnauthenticatedReason string           = "Account authentication could not be verified"

	// MalformedRequest occurs when the request body cannot be read
	ErrorMalformedRequest       ServiceErrorCode = 17
	ErrorMalformedRequestReason string           = "Unable to read request body"

	// Bad Request
	ErrorBadRequest       ServiceErrorCode = 21
	ErrorBadRequestReason string           = "Bad request"

	// Invalid Search Query
	ErrorFailedToParseSearch       ServiceErrorCode = 23
	ErrorFailedToParseSearchReason string           = "Failed to parse search query"

	// Synchronous request not supported
	ErrorSyncActionNotSupported       ServiceErrorCode = 103
	ErrorSyncActionNotSupportedReason string           = "Synchronous action is not supported, use async=true parameter"

	// Failed to create sso client
	ErrorFailedToCreateSSOClient       ServiceErrorCode = 106
	ErrorFailedToCreateSSOClientReason string           = "Failed to create kafka client in the mas sso"

	// Failed to get sso client secret
	ErrorFailedToGetSSOClientSecret       ServiceErrorCode = 107
	ErrorFailedToGetSSOClientSecretReason string           = "Failed to get kafka client secret from the mas sso"

	// Failed to get sso client
	ErrorFailedToGetSSOClient       ServiceErrorCode = 108
	ErrorFailedToGetSSOClientReason string           = "Failed to get kafka client from the mas sso"

	// Failed to delete sso client
	ErrorFailedToDeleteSSOClient       ServiceErrorCode = 109
	ErrorFailedToDeleteSSOClientReason string           = "Failed to delete kafka client from the mas sso"

	// Failed to create service account
	ErrorFailedToCreateServiceAccount       ServiceErrorCode = 110
	ErrorFailedToCreateServiceAccountReason string           = "Failed to create service account"

	// Failed to get service account
	ErrorFailedToGetServiceAccount       ServiceErrorCode = 111
	ErrorFailedToGetServiceAccountReason string           = "Failed to get service account"

	// Failed to delete service account
	ErrorFailedToDeleteServiceAccount       ServiceErrorCode = 112
	ErrorFailedToDeleteServiceAccountReason string           = "Failed to delete service account"

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

	// Only MultiAZ is supported
	ErrorOnlyMultiAZSupported       ServiceErrorCode = 35
	ErrorOnlyMultiAZSupportedReason string           = "Only multiAZ Kafkas are supported, use multi_az=true"

	// Failure to send an error response (i.e. unable to send error response as the error can't be converted to JSON.)
	ErrorUnableToSendErrorResponse       ServiceErrorCode = 1000
	ErrorUnableToSendErrorResponseReason string           = "An unexpected error happened, please check the log of the service for details"
)

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
		ServiceError{ErrorForbidden, ErrorForbiddenReason, http.StatusForbidden},
		ServiceError{ErrorMaxAllowedInstanceReached, ErrorMaxAllowedInstanceReachedReason, http.StatusForbidden},
		ServiceError{ErrorConflict, ErrorConflictReason, http.StatusConflict},
		ServiceError{ErrorNotFound, ErrorNotFoundReason, http.StatusNotFound},
		ServiceError{ErrorValidation, ErrorValidationReason, http.StatusBadRequest},
		ServiceError{ErrorGeneral, ErrorGeneralReason, http.StatusInternalServerError},
		ServiceError{ErrorNotImplemented, ErrorNotImplementedReason, http.StatusMethodNotAllowed},
		ServiceError{ErrorUnauthorized, ErrorUnauthorizedReason, http.StatusForbidden},
		ServiceError{ErrorUnauthenticated, ErrorUnauthenticatedReason, http.StatusUnauthorized},
		ServiceError{ErrorMalformedRequest, ErrorMalformedRequestReason, http.StatusBadRequest},
		ServiceError{ErrorBadRequest, ErrorBadRequestReason, http.StatusBadRequest},
		ServiceError{ErrorFailedToParseSearch, ErrorFailedToParseSearchReason, http.StatusBadRequest},
		ServiceError{ErrorSyncActionNotSupported, ErrorSyncActionNotSupportedReason, http.StatusBadRequest},
		ServiceError{ErrorFailedToCreateSSOClient, ErrorFailedToCreateSSOClientReason, http.StatusBadRequest},
		ServiceError{ErrorFailedToGetSSOClientSecret, ErrorFailedToGetSSOClientSecretReason, http.StatusNotFound},
		ServiceError{ErrorFailedToGetSSOClient, ErrorFailedToGetSSOClientReason, http.StatusNotFound},
		ServiceError{ErrorFailedToDeleteSSOClient, ErrorFailedToDeleteSSOClientReason, http.StatusNotFound},
		ServiceError{ErrorFailedToCreateServiceAccount, ErrorFailedToCreateServiceAccountReason, http.StatusBadRequest},
		ServiceError{ErrorFailedToGetServiceAccount, ErrorFailedToGetServiceAccountReason, http.StatusNotFound},
		ServiceError{ErrorFailedToDeleteServiceAccount, ErrorFailedToDeleteServiceAccountReason, http.StatusNotFound},
		ServiceError{ErrorProviderNotSupported, ErrorProviderNotSupportedReason, http.StatusBadRequest},
		ServiceError{ErrorRegionNotSupported, ErrorRegionNotSupportedReason, http.StatusBadRequest},
		ServiceError{ErrorMalformedKafkaClusterName, ErrorMalformedKafkaClusterNameReason, http.StatusBadRequest},
		ServiceError{ErrorMinimumFieldLength, ErrorMinimumFieldLengthReason, http.StatusBadRequest},
		ServiceError{ErrorMaximumFieldLength, ErrorMaximumFieldLengthReason, http.StatusBadRequest},
		ServiceError{ErrorOnlyMultiAZSupported, ErrorOnlyMultiAZSupportedReason, http.StatusBadRequest},
		ServiceError{ErrorUnableToSendErrorResponse, ErrorUnableToSendErrorResponseReason, http.StatusInternalServerError},
	}
}

func NewErrorFromHTTPStatusCode(httpCode int, reason string, values ...interface{}) *ServiceError {
	if httpCode >= http.StatusBadRequest && httpCode < http.StatusInternalServerError {
		switch httpCode {
		case http.StatusBadRequest:
			return BadRequest(reason, values...)
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
		default:
			return BadRequest(reason, values...)
		}
	}

	if httpCode >= http.StatusInternalServerError {
		switch httpCode {
		case http.StatusInternalServerError:
			return GeneralError(reason, values...)
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

type ServiceError struct {
	// Code is the numeric and distinct ID for the error
	Code ServiceErrorCode
	// Reason is the context-specific reason the error was generated
	Reason string
	// HttopCode is the HttpCode associated with the error when the error is returned as an API response
	HttpCode int
}

// Reason can be a string with format verbs, which will be replace by the specified values
func New(code ServiceErrorCode, reason string, values ...interface{}) *ServiceError {
	// If the code isn't defined, use the general error code
	var err *ServiceError
	exists, err := Find(code)
	if !exists {
		glog.Errorf("Undefined error code used: %d", code)
		err = &ServiceError{ErrorGeneral, "Unspecified error", 500}
	}

	// If the reason is unspecified, use the default
	if reason != "" {
		err.Reason = fmt.Sprintf(reason, values...)
	}

	return err
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", CodeStr(e.Code), e.Reason)
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

func (e *ServiceError) AsOpenapiError(operationID string) openapi.Error {
	return openapi.Error{
		Kind:        "Error",
		Id:          strconv.Itoa(int(e.Code)),
		Href:        Href(e.Code),
		Code:        CodeStr(e.Code),
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

func Unauthenticated(reason string, values ...interface{}) *ServiceError {
	return New(ErrorUnauthenticated, reason, values...)
}

func Forbidden(reason string, values ...interface{}) *ServiceError {
	return New(ErrorForbidden, reason, values...)
}

func MaximumAllowedInstanceReached(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMaxAllowedInstanceReached, reason, values...)
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

func NotMultiAzActionNotSupported() *ServiceError {
	return New(ErrorOnlyMultiAZSupported, ErrorOnlyMultiAZSupportedReason)
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

func FailedToGetServiceAccount(reason string, values ...interface{}) *ServiceError {
	return New(ErrorFailedToGetServiceAccount, reason, values...)
}

func RegionNotSupported(reason string, values ...interface{}) *ServiceError {
	return New(ErrorRegionNotSupported, reason, values...)
}

func ProviderNotSupported(reason string, values ...interface{}) *ServiceError {
	return New(ErrorProviderNotSupported, reason, values...)
}

func MalformedKafkaClusterName(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMalformedKafkaClusterName, reason, values...)
}

func MinimumFieldLengthNotReached(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMinimumFieldLength, reason, values...)
}

func MaximumFieldLengthMissing(reason string, values ...interface{}) *ServiceError {
	return New(ErrorMaximumFieldLength, reason, values...)
}

func UnableToSendErrorResponse() *ServiceError {
	return New(ErrorUnableToSendErrorResponse, ErrorUnableToSendErrorResponseReason)
}
