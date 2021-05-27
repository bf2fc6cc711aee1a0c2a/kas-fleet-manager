/*
 * Connector Service Fleet Manager
 *
 * Connector Service Fleet Manager is a Rest API to manage connectors.
 *
 * API version: 0.0.2
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// KafkaConnectionSettings struct for KafkaConnectionSettings
type KafkaConnectionSettings struct {
	BootstrapServer string `json:"bootstrap_server,omitempty"`
	ClientId        string `json:"client_id,omitempty"`
	ClientSecret    string `json:"client_secret,omitempty"`
}
