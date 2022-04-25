/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage Kafka instances.
 *
 * API version: 1.5.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// SupportedKafkaInstanceType Supported Kafka instance type
type SupportedKafkaInstanceType struct {
	// Unique identifier of the Kafka instance type.
	Id string `json:"id,omitempty"`
	// Human readable name of the supported Kafka instance type
	DisplayName string `json:"display_name,omitempty"`
	//  A list of Kafka instance sizes available for this instance type
	Sizes []SupportedKafkaSize `json:"sizes,omitempty"`
}
