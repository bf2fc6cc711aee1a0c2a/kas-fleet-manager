/*
 * Kafka Management API
 *
 * Kafka Management API is a REST API to manage Kafka instances
 *
 * API version: 1.15.0
 * Contact: rhosak-support@redhat.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// EnterpriseCluster struct for EnterpriseCluster
type EnterpriseCluster struct {
	Id   string `json:"id"`
	Kind string `json:"kind"`
	Href string `json:"href"`
	// Indicates whether Kafkas created on this data plane cluster have to be accessed via private network
	AccessKafkasViaPrivateNetwork bool `json:"access_kafkas_via_private_network"`
	// ocm cluster id of the registered Enterprise cluster
	ClusterId string `json:"cluster_id,omitempty"`
	// status of registered Enterprise cluster
	Status string `json:"status,omitempty"`
}
