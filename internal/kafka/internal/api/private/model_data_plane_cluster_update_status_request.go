/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 1.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// DataPlaneClusterUpdateStatusRequest Schema for the request to update a data plane cluster's status
type DataPlaneClusterUpdateStatusRequest struct {
	// The cluster data plane conditions
	Conditions []DataPlaneClusterUpdateStatusRequestConditions `json:"conditions,omitempty"`
	Total      DataPlaneClusterUpdateStatusRequestTotal        `json:"total,omitempty"`
	Remaining  DataPlaneClusterUpdateStatusRequestTotal        `json:"remaining,omitempty"`
	// Deprecated
	DeprecatedNodeInfo *DatePlaneClusterUpdateStatusRequestDeprecatedNodeInfo `json:"nodeInfo,omitempty"`
	NodeInfo           *DataPlaneClusterUpdateStatusRequestNodeInfo           `json:"node_info,omitempty"`
	// Deprecated
	DeprecatedResizeInfo *DatePlaneClusterUpdateStatusRequestDeprecatedResizeInfo `json:"resizeInfo,omitempty"`
	ResizeInfo           *DataPlaneClusterUpdateStatusRequestResizeInfo           `json:"resize_info,omitempty"`
	StrimziVersions      []string                                                 `json:"strimziVersions,omitempty"`
}
