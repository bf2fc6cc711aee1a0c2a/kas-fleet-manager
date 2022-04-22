/*
 * Connector Service Fleet Manager Admin APIs
 *
 * Connector Service Fleet Manager Admin is a Rest API to manage connector clusters.
 *
 * API version: 0.0.3
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ConnectorDeploymentStatusOperators struct for ConnectorDeploymentStatusOperators
type ConnectorDeploymentStatusOperators struct {
	Assigned  ConnectorOperator `json:"assigned,omitempty"`
	Available ConnectorOperator `json:"available,omitempty"`
}
