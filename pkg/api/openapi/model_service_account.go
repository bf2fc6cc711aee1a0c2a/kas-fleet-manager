/*
 * Managed Service API
 *
 * Managed Service API
 *
 * API version: 0.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ServiceAccount Service Account created in MAS-SSO for the Kafka Cluster for authentication
type ServiceAccount struct {
	ClientID     string `json:"clientID,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Id           string `json:"id,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Href         string `json:"href,omitempty"`
}
