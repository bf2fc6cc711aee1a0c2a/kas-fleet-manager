/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 1.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ErrorList struct for ErrorList
type ErrorList struct {
	Kind  string  `json:"kind"`
	Page  int32   `json:"page"`
	Size  int32   `json:"size"`
	Total int32   `json:"total"`
	Items []Error `json:"items"`
}
