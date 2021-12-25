/*
 * Connector Service Fleet Manager Private APIs
 *
 * Connector Service Fleet Manager apis that are used by internal services.
 *
 * API version: 0.0.3
 * Contact: foo@bar
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ConnectorClusterStatus Schema for the request to update a data plane cluster's status
type ConnectorClusterStatus struct {
	Phase      string            `json:"phase,omitempty"`
	Version    string            `json:"version,omitempty"`
	Conditions []MetaV1Condition `json:"conditions,omitempty"`
	// the list of installed operators
	Operators []ConnectorClusterStatusOperators `json:"operators,omitempty"`
}
