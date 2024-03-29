/*
 * Connector Service Fleet Manager Private APIs
 *
 * Connector Service Fleet Manager apis that are used by internal services.
 *
 * API version: 0.0.3
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ConnectorNamespaceTenant struct for ConnectorNamespaceTenant
type ConnectorNamespaceTenant struct {
	Kind ConnectorNamespaceTenantKind `json:"kind"`
	// Either user or organisation id depending on the value of kind
	Id string `json:"id"`
}
