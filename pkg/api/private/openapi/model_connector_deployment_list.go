/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// ConnectorDeploymentList struct for ConnectorDeploymentList
type ConnectorDeploymentList struct {
	Kind  string                `json:"kind"`
	Page  int32                 `json:"page"`
	Size  int32                 `json:"size"`
	Total int32                 `json:"total"`
	Items []ConnectorDeployment `json:"items"`
}
