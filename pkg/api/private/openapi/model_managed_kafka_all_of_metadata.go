/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 1.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ManagedKafkaAllOfMetadata struct for ManagedKafkaAllOfMetadata
type ManagedKafkaAllOfMetadata struct {
	Name        string                               `json:"name,omitempty"`
	Namespace   string                               `json:"namespace,omitempty"`
	Annotations ManagedKafkaAllOfMetadataAnnotations `json:"annotations,omitempty"`
}
