/*
 * Kafka Management API
 *
 * Kafka Management API is a REST API to manage Kafka instances
 *
 * API version: 1.13.0
 * Contact: rhosak-support@redhat.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// ServiceAccountRequest Schema for the request to create a service account
type ServiceAccountRequest struct {
	// The name of the service account
	Name string `json:"name"`
	// A description for the service account
	Description string `json:"description,omitempty"`
}
