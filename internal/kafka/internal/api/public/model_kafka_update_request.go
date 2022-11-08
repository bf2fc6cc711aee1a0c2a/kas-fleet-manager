/*
 * Kafka Management API
 *
 * Kafka Management API is a REST API to manage Kafka instances
 *
 * API version: 1.13.0
 * Contact: rhosak-support@redhat.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// KafkaUpdateRequest struct for KafkaUpdateRequest
type KafkaUpdateRequest struct {
	Owner *string `json:"owner,omitempty"`
	// Whether connection reauthentication is enabled or not. If set to true, connection reauthentication on the Kafka instance will be required every 5 minutes.
	ReauthenticationEnabled *bool `json:"reauthentication_enabled,omitempty"`
}
