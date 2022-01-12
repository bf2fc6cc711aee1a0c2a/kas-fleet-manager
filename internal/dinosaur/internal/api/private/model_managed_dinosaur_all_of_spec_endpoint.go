/*
 * Dinosaur Service Fleet Manager
 *
 * Dinosaur Service Fleet Manager APIs that are used by internal services e.g fleetshard operators.
 *
 * API version: 1.4.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ManagedDinosaurAllOfSpecEndpoint struct for ManagedDinosaurAllOfSpecEndpoint
type ManagedDinosaurAllOfSpecEndpoint struct {
	BootstrapServerHost string                               `json:"bootstrapServerHost,omitempty"`
	Tls                 *ManagedDinosaurAllOfSpecEndpointTls `json:"tls,omitempty"`
}
