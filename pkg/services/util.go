package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/jinzhu/gorm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	truncatedNameLen = 10
	// Maximum namespace name length is 63
	// Namespace name is built using the kafka request id (always generated with 27 length) and the owner (truncated with this var).
	// Set the truncate index to 35 to ensure that the namespace name does not go over the maximum limit.
	truncatedNamespaceLen     = 35
	truncatedSyncsetIdLen     = 50
	replacementForSpecialChar = "-"
	appendChar                = "a"
)

// All namespace names must conform to DNS1123. This will inverse the validation RE from k8s (https://github.com/kubernetes/apimachinery/blob/master/pkg/util/validation/validation.go#L177)
var dns1123ReplacementRE = regexp.MustCompile(`[^a-z0-9]([^-a-z0-9]*[^a-z0-9])?`)

// All OpenShift route hosts must confirm to DNS1035. This will inverse the validation RE from k8s (https://github.com/kubernetes/apimachinery/blob/master/pkg/util/validation/validation.go#L219)
var dns1035ReplacementRE = regexp.MustCompile(`[^a-z]([^-a-z0-9]*[^a-z0-9])?`)

// Field names suspected to contain personally identifiable information
var piiFields []string = []string{
	"username",
	"first_name",
	"last_name",
	"email",
	"address",
}

func handleGetError(resourceType, field string, value interface{}, err error) *errors.ServiceError {
	// Sanitize errors of any personally identifiable information
	for _, f := range piiFields {
		if field == f {
			value = "<redacted>"
			break
		}
	}
	if gorm.IsRecordNotFoundError(err) {
		return errors.NotFound("%s with %s='%v' not found", resourceType, field, value)
	}
	return errors.GeneralError("Unable to find %s with %s='%v': %s", resourceType, field, value, err)
}

func handleCreateError(resourceType string, err error) *errors.ServiceError {
	if strings.Contains(err.Error(), "violates unique constraint") {
		return errors.Conflict("This %s already exists", resourceType)
	}
	return errors.GeneralError("Unable to create %s: %s", resourceType, err.Error())
}

func handleUpdateError(resourceType string, err error) *errors.ServiceError {
	if strings.Contains(err.Error(), "violates unique constraint") {
		return errors.Conflict("Changes to %s conflict with existing records", resourceType)
	}
	return errors.GeneralError("Unable to update %s: %s", resourceType, err.Error())
}

func truncateString(str string, num int) string {
	truncatedString := str
	if len(str) > num {
		truncatedString = str[0:num]
	}
	return truncatedString
}

// buildKafkaNamespaceIdentifier creates a unique identifier for a namespace to be used for
// the kafka request
func buildKafkaNamespaceIdentifier(kafkaRequest *api.KafkaRequest) string {
	return fmt.Sprintf("%s-%s", truncateString(kafkaRequest.Owner, truncatedNamespaceLen), strings.ToLower(kafkaRequest.ID))
}

// buildKafkaIdentifier creates a unique identifier for a kafka cluster given
// the kafka request object
func buildKafkaIdentifier(kafkaRequest *api.KafkaRequest) string {
	return fmt.Sprintf("%s-%s", kafkaRequest.Name, strings.ToLower(kafkaRequest.ID))
}

// buildTruncateKafkaIdentifier creates a unique identifier for a kafka cluster given
// the kafka request object
func buildTruncateKafkaIdentifier(kafkaRequest *api.KafkaRequest) string {
	return fmt.Sprintf("%s-%s", truncateString(kafkaRequest.Name, truncatedNameLen), strings.ToLower(kafkaRequest.ID))
}

// buildSyncsetIdentifier creates a unique identifier for the syncset given
// the unique kafka identifier
func buildSyncsetIdentifier(kafkaRequest *api.KafkaRequest) string {
	fullSyncSetId := fmt.Sprintf("ext-%s", buildKafkaIdentifier(kafkaRequest))
	// Max SyncSetID length in OCM is 50
	return truncateString(fullSyncSetId, truncatedSyncsetIdLen)
}

// maskProceedingandTrailingDash replaces the first and final character of a string with a subdomain safe
// value if is a dash.
func maskProceedingandTrailingDash(name string) string {
	if strings.HasSuffix(name, "-") {
		name = name[:len(name)-1] + appendChar
	}

	if strings.HasPrefix(name, "-") {
		name = strings.Replace(name, "-", appendChar, 1)
	}
	return name
}

// replaceNamespaceSpecialChar replaces invalid characters with random char in the namespace name
func replaceNamespaceSpecialChar(name string) (string, error) {
	replacedName := dns1123ReplacementRE.ReplaceAllString(strings.ToLower(name), replacementForSpecialChar)

	replacedName = maskProceedingandTrailingDash(replacedName)

	// This should never fail based on above replacement of invalid characters.
	validationErrors := validation.IsDNS1123Label(replacedName)
	if len(validationErrors) > 0 {
		return replacedName, fmt.Errorf("Namespace name is invalid: %s. A DNS-1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name', or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')", strings.Join(validationErrors[:], ","))
	}

	return replacedName, nil
}

// replaceHostSpecialChar replaces invalid characters with random char in the namespace name
func replaceHostSpecialChar(name string) (string, error) {
	replacedName := dns1035ReplacementRE.ReplaceAllString(strings.ToLower(name), replacementForSpecialChar)

	replacedName = maskProceedingandTrailingDash(replacedName)

	// This should never fail based on above replacement of invalid characters.
	validationErrors := validation.IsDNS1035Label(replacedName)
	if len(validationErrors) > 0 {
		return replacedName, fmt.Errorf("Host name is not valid: %s. A DNS-1035 label must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character, regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?'", strings.Join(validationErrors[:], ","))
	}

	return replacedName, nil
}

func BuildNamespaceName(kafka *api.KafkaRequest) (string, error) {
	namespaceName := buildKafkaNamespaceIdentifier(kafka)
	namespaceName, err := replaceNamespaceSpecialChar(namespaceName)
	if err != nil {
		return namespaceName, fmt.Errorf("failed to build namespace for kafka %s: %w", kafka.ID, err)
	}
	return namespaceName, nil
}

func safeString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// checks if slice of strings contains given string
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func stringSliceToInterfaceSlice(stringSlice []string) []interface{} {
	interfaceSlice := make([]interface{}, len(stringSlice))
	for i := range stringSlice {
		interfaceSlice[i] = stringSlice[i]
	}
	return interfaceSlice
}
