/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.5.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// DataPlaneClusterUpdateStatusRequest Schema for the request to update a data plane cluster's status
type DataPlaneClusterUpdateStatusRequest struct {
	// The cluster data plane conditions
	Conditions []DataPlaneClusterUpdateStatusRequestConditions `json:"conditions,omitempty"`
	Total      DataPlaneClusterUpdateStatusRequestTotal        `json:"total,omitempty"`
	Remaining  DataPlaneClusterUpdateStatusRequestTotal        `json:"remaining,omitempty"`
	NodeInfo   *DatePlaneClusterUpdateStatusRequestNodeInfo    `json:"nodeInfo,omitempty"`
	ResizeInfo *DatePlaneClusterUpdateStatusRequestResizeInfo  `json:"resizeInfo,omitempty"`
	Strimzi    []DataPlaneClusterUpdateStatusRequestStrimzi    `json:"strimzi,omitempty"`
}
