/*
 * Kafka Service Fleet Manager Admin APIs
 *
 * The admin APIs for the fleet manager of Kafka service
 *
 * API version: 0.0.2
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// KafkaUpdateRequest struct for KafkaUpdateRequest
type KafkaUpdateRequest struct {
	StrimziVersion string `json:"strimzi_version,omitempty"`
	KafkaVersion   string `json:"kafka_version,omitempty"`
}
