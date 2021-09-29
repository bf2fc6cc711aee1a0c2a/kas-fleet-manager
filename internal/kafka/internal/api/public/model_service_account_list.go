/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage Kafka instances.
 *
 * API version: 1.1.3
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// ServiceAccountList struct for ServiceAccountList
type ServiceAccountList struct {
	Kind  string                   `json:"kind"`
	Items []ServiceAccountListItem `json:"items"`
}
