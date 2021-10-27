/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage Kafka instances.
 *
 * API version: 1.2.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// ErrorList struct for ErrorList
type ErrorList struct {
	Kind  string  `json:"kind"`
	Page  int32   `json:"page"`
	Size  int32   `json:"size"`
	Total int32   `json:"total"`
	Items []Error `json:"items"`
}
