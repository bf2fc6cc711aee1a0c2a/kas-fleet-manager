/*
 * Connector Service Fleet Manager Private APIs
 *
 * Connector Service Fleet Manager apis that are used by internal services.
 *
 * API version: 0.0.3
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ConnectorClusterState the model 'ConnectorClusterState'
type ConnectorClusterState string

// List of ConnectorClusterState
const (
	CONNECTORCLUSTERSTATE_DISCONNECTED ConnectorClusterState = "disconnected"
	CONNECTORCLUSTERSTATE_READY        ConnectorClusterState = "ready"
	CONNECTORCLUSTERSTATE_DELETING     ConnectorClusterState = "deleting"
)
