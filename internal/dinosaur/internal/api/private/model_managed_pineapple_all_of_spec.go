/*
 * Pineapple Service Fleet Manager
 *
 * Pineapple Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.3.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ManagedPineappleAllOfSpec struct for ManagedPineappleAllOfSpec
type ManagedPineappleAllOfSpec struct {
	// Service account configuration for a Pineapple cluster
	ServiceAccounts []ManagedPineappleAllOfSpecServiceAccounts `json:"serviceAccounts,omitempty"`
	// Capacity information of a Pineapple cluster
	Capacity map[string]interface{}         `json:"capacity,omitempty"`
	Oauth    ManagedPineappleAllOfSpecOauth `json:"oauth,omitempty"`
	// A list of users that owns this Pineapple cluster
	Owners   []string                          `json:"owners,omitempty"`
	Endpoint ManagedPineappleAllOfSpecEndpoint `json:"endpoint,omitempty"`
	Versions ManagedPineappleVersions          `json:"versions,omitempty"`
	Deleted  bool                              `json:"deleted"`
}
