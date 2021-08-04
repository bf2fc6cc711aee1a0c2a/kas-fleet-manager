/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.2.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ManagedKafkaVersions struct for ManagedKafkaVersions
type ManagedKafkaVersions struct {
	Kafka   string `json:"kafka,omitempty"`
	Strimzi string `json:"strimzi,omitempty"`
}
