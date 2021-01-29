/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 0.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// Connector A connector holds the configuration to connect a Kafka topic to another system.
type Connector struct {
	Id                 string                           `json:"id,omitempty"`
	Kind               string                           `json:"kind,omitempty"`
	Href               string                           `json:"href,omitempty"`
	Metadata           ConnectorAllOfMetadata           `json:"metadata,omitempty"`
	DeploymentLocation ConnectorAllOfDeploymentLocation `json:"deployment_location,omitempty"`
	ConnectorTypeId    string                           `json:"connector_type_id,omitempty"`
	ConnectorSpec      map[string]interface{}           `json:"connector_spec,omitempty"`
	Status             string                           `json:"status,omitempty"`
}
