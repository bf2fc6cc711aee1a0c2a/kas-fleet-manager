/*
 * OCM Managed Service API
 *
 * OCM Managed Service API
 *
 * API version: 0.0.1
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
