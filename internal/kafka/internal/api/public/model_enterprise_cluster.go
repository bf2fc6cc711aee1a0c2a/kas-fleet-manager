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
	// The OCM's cluster id of the registered Enterprise cluster.
	ClusterId string `json:"cluster_id,omitempty"`
	// The status of Enterprise cluster registration
	Status string `json:"status,omitempty"`
	// The cloud provider for this cluster. This valus will be used as the Kafka's cloud provider value when a Kafka is created on this cluster
	CloudProvider string `json:"cloud_provider,omitempty"`
	// The region of this cluster. This valus will be used as the Kafka's region value when a Kafka is created on this cluster
	Region string `json:"region,omitempty"`
	// A flag indicating whether this cluster is available on multiple availability zones or not
	MultiAz                bool                                      `json:"multi_az"`
	SupportedInstanceTypes SupportedKafkaInstanceTypesList           `json:"supported_instance_types,omitempty"`
	CapacityInformation    EnterpriseClusterAllOfCapacityInformation `json:"capacity_information,omitempty"`
}
