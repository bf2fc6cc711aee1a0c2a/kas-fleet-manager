package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"

	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	truncatedNameLen = 10
	// Maximum namespace name length is 63
	// Namespace name is built using the kafka request id (always generated with 27 length) and the owner (truncated with this var).
	// Set the truncate index to 35 to ensure that the namespace name does not go over the maximum limit.
	truncatedNamespaceLen     = 35
	replacementForSpecialChar = "-"
	appendChar                = "a"
)

// All namespace names must conform to DNS1123. This will inverse the validation RE from k8s (https://github.com/kubernetes/apimachinery/blob/master/pkg/util/validation/validation.go#L177)
var dns1123ReplacementRE = regexp.MustCompile(`[^a-z0-9]([^-a-z0-9]*[^a-z0-9])?`)

// All OpenShift route hosts must confirm to DNS1035. This will inverse the validation RE from k8s (https://github.com/kubernetes/apimachinery/blob/master/pkg/util/validation/validation.go#L219)
var dns1035ReplacementRE = regexp.MustCompile(`[^a-z]([^-a-z0-9]*[^a-z0-9])?`)

func truncateString(str string, num int) string {
	truncatedString := str
	if len(str) > num {
		truncatedString = str[0:num]
	}
	return truncatedString
}

// buildKafkaNamespaceIdentifier creates a unique identifier for a namespace to be used for
// the kafka request
func buildKafkaNamespaceIdentifier(kafkaRequest *dbapi.KafkaRequest) string {
	return fmt.Sprintf("%s-%s", truncateString(kafkaRequest.Owner, truncatedNamespaceLen), strings.ToLower(kafkaRequest.ID))
}

// buildTruncateKafkaIdentifier creates a unique identifier for a kafka cluster given
// the kafka request object
func buildTruncateKafkaIdentifier(kafkaRequest *dbapi.KafkaRequest) string {
	return fmt.Sprintf("%s-%s", truncateString(kafkaRequest.Name, truncatedNameLen), strings.ToLower(kafkaRequest.ID))
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

// This is there to ensure backward compatibility of old namespace
func BuildNamespaceName(kafka *dbapi.KafkaRequest) (string, error) {
	namespaceName := buildKafkaNamespaceIdentifier(kafka)
	namespaceName, err := replaceNamespaceSpecialChar(namespaceName)
	if err != nil {
		return namespaceName, fmt.Errorf("failed to build namespace for kafka %s: %w", kafka.ID, err)
	}
	return namespaceName, nil
}

// BuildKeycloakClientNameIdentifier builds an identifier based on the kafka request id
func BuildKeycloakClientNameIdentifier(kafkaRequestID string) string {
	return fmt.Sprintf("%s-%s", "kafka", strings.ToLower(kafkaRequestID))
}

func BuildCustomClaimCheck(kafkaRequest *dbapi.KafkaRequest) string {
	return fmt.Sprintf("@.rh-org-id == '%s' && (( @.rh-user-id && @.rh-user-id =='%s') || !@.rh-user-id)", kafkaRequest.OrganisationId, kafkaRequest.OwnerAccountId)
}
