/*
 * Connector Service Fleet Manager Admin APIs
 *
 * Connector Service Fleet Manager Admin is a Rest API to manage connector clusters.
 *
 * API version: 0.0.3
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ConnectorDeploymentAdminStatus The status of connector deployment
type ConnectorDeploymentAdminStatus struct {
	Phase           ConnectorState                              `json:"phase,omitempty"`
	ResourceVersion int64                                       `json:"resource_version,omitempty"`
	ShardMetadata   ConnectorDeploymentAdminStatusShardMetadata `json:"shard_metadata,omitempty"`
	Operators       ConnectorDeploymentAdminStatusOperators     `json:"operators,omitempty"`
	Conditions      []MetaV1Condition                           `json:"conditions,omitempty"`
}
