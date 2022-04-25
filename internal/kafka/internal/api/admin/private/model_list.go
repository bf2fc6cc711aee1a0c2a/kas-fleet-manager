/*
 * Kafka Service Fleet Manager Admin APIs
 *
 * The admin APIs for the fleet manager of Kafka service
 *
 * API version: 0.0.3
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// List struct for List
type List struct {
	Kind  string            `json:"kind"`
	Page  int32             `json:"page"`
	Size  int32             `json:"size"`
	Total int32             `json:"total"`
	Items []ObjectReference `json:"items"`
}
