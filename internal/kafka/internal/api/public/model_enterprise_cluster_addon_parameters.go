/*
 * Kafka Management API
 *
 * Kafka Management API is a REST API to manage Kafka instances
 *
 * API version: 1.16.0
 * Contact: rhosak-support@redhat.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// EnterpriseClusterAddonParameters Enterprise Cluster get addon parameters response
type EnterpriseClusterAddonParameters struct {
	Id   string `json:"id"`
	Kind string `json:"kind"`
	Href string `json:"href"`
	// Enterprise Cluster fleetshard parameters array
	FleetshardParameters []FleetshardParameter `json:"fleetshard_parameters,omitempty"`
}
