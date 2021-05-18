/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ConnectorDeploymentStatus The status of connector deployment
type ConnectorDeploymentStatus struct {
	Phase        string            `json:"phase,omitempty"`
	SpecChecksum string            `json:"spec_checksum,omitempty"`
	Conditions   []MetaV1Condition `json:"conditions,omitempty"`
}
