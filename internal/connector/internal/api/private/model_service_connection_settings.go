/*
 * Connector Service Fleet Manager Private APIs
 *
 * Connector Service Fleet Manager apis that are used by internal services.
 *
 * API version: 0.0.3
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ServiceConnectionSettings struct for ServiceConnectionSettings
type ServiceConnectionSettings struct {
	Id           string `json:"id"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Url          string `json:"url"`
}