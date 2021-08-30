/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.3.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ManagedKafkaAllOfSpec struct for ManagedKafkaAllOfSpec
type ManagedKafkaAllOfSpec struct {
	ServiceAccounts []ManagedKafkaAllOfSpecServiceAccounts `json:"serviceAccounts,omitempty"`
	Capacity        ManagedKafkaCapacity                   `json:"capacity,omitempty"`
	Oauth           ManagedKafkaAllOfSpecOauth             `json:"oauth,omitempty"`
	Owners          []string                               `json:"owners,omitempty"`
	Endpoint        ManagedKafkaAllOfSpecEndpoint          `json:"endpoint,omitempty"`
	Versions        ManagedKafkaVersions                   `json:"versions,omitempty"`
	Deleted         bool                                   `json:"deleted"`
}
