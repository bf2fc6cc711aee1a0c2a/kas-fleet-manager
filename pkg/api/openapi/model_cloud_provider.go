/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage Kafka instances and connectors.
 *
 * API version: 1.1.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// CloudProvider Cloud provider.
type CloudProvider struct {
	// Indicates the type of this object. Will be 'CloudProvider' link.
	Kind string `json:"kind,omitempty"`
	// Unique identifier of the object.
	Id string `json:"id,omitempty"`
	// Name of the cloud provider for display purposes.
	DisplayName string `json:"display_name,omitempty"`
	// Human friendly identifier of the cloud provider, for example `aws`.
	Name string `json:"name,omitempty"`
	// Whether the cloud provider is enabled for deploying an OSD cluster.
	Enabled bool `json:"enabled"`
}
