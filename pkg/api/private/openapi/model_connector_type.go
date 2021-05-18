/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ConnectorType Represents a connector type supported by the API
type ConnectorType struct {
	Id   string `json:"id,omitempty"`
	Kind string `json:"kind,omitempty"`
	Href string `json:"href,omitempty"`
	// Name of the connector type.
	Name string `json:"name"`
	// Version of the connector type.
	Version string `json:"version"`
	// A description of the connector.
	Description string `json:"description,omitempty"`
	// URL to an icon of the connector.
	IconHref string `json:"icon_href,omitempty"`
	// labels used to categorize the connector
	Labels []string `json:"labels,omitempty"`
	// A json schema that can be used to validate a connectors connector_spec field.
	JsonSchema map[string]interface{} `json:"json_schema"`
}
