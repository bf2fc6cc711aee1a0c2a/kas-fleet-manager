/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 0.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ServiceAccountList struct for ServiceAccountList
type ServiceAccountList struct {
	Kind  string                   `json:"kind,omitempty"`
	Items []ServiceAccountListItem `json:"items,omitempty"`
}
