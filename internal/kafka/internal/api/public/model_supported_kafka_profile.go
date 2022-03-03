/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage Kafka instances.
 *
 * API version: 1.3.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// SupportedKafkaProfile Supported Kafka Profile
type SupportedKafkaProfile struct {
	// Indicates the type of this object. Will be 'SupportedKafkaProfile' link.
	Kind string `json:"kind,omitempty"`
	// Unique identifier of the Kafka instance profile.
	Id string `json:"id,omitempty"`
	//  A list of Kafka instance sizes available for this profile
	Sizes []SupportedKafkaSize `json:"sizes,omitempty"`
}
