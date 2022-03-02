/*
 * Connector Service Fleet Manager
 *
 * Connector Service Fleet Manager is a Rest API to manage connectors.
 *
 * API version: 0.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// ConnectorNamespaceRequest A connector namespace create request
type ConnectorNamespaceRequest struct {
	Name        string                                     `json:"name"`
	Annotations []ConnectorNamespaceRequestMetaAnnotations `json:"annotations,omitempty"`
	ClusterId   string                                     `json:"cluster_id"`
	Kind        ConnectorNamespaceTenantKind               `json:"kind"`
}
