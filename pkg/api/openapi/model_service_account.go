/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 0.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ServiceAccount Service Account created in MAS-SSO for the Kafka Cluster for authentication
type ServiceAccount struct {
	// server generated unique id of the service account
	Id           string `json:"id,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Href         string `json:"href,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	ClientID     string `json:"clientID,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	Owner        string `json:"owner,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}
