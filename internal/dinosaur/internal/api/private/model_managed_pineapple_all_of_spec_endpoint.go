/*
 * Pineapple Service Fleet Manager
 *
 * Pineapple Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.3.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ManagedPineappleAllOfSpecEndpoint struct for ManagedPineappleAllOfSpecEndpoint
type ManagedPineappleAllOfSpecEndpoint struct {
	BootstrapServerHost string                                `json:"bootstrapServerHost,omitempty"`
	Tls                 *ManagedPineappleAllOfSpecEndpointTls `json:"tls,omitempty"`
}
