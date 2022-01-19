/*
 * Kafka Service Fleet Manager Admin APIs
 *
 * The admin APIs for the fleet manager of Kafka service
 *
 * API version: 0.0.2
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// Error struct for Error
type Error struct {
	Id          string `json:"id,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Href        string `json:"href,omitempty"`
	Code        string `json:"code,omitempty"`
	Reason      string `json:"reason,omitempty"`
	OperationId string `json:"operation_id,omitempty"`
}