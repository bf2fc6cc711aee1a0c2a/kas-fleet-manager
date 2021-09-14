/*
 * Pineapple Service Fleet Manager
 *
 * Pineapple Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.3.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// DataPlaneClusterUpdateStatusRequest Schema for the request to update a data plane cluster's status
type DataPlaneClusterUpdateStatusRequest struct {
	// The data plane cluster conditions
	Conditions []DataPlaneClusterUpdateStatusRequestConditions `json:"conditions,omitempty"`
	// The total capacity of the data plane cluster for Pineapple clusters.  This should state the total amount of currently used and available data plane resources that can be consumed by Pineapple clusters.
	Total map[string]interface{} `json:"total,omitempty"`
	// The remaining capacity of the data plane cluster for Pineapple clusters.  This should state the remaining amount of available data plane resources that can be consumed by Pineapple clusters.
	Remaining  map[string]interface{}                         `json:"remaining,omitempty"`
	NodeInfo   *DatePlaneClusterUpdateStatusRequestNodeInfo   `json:"nodeInfo,omitempty"`
	ResizeInfo *DatePlaneClusterUpdateStatusRequestResizeInfo `json:"resizeInfo,omitempty"`
	// A list of the Pineapple operator versions that can be installed on the data plane cluster
	PineappleOperatorVersions []string `json:"pineappleOperatorVersions,omitempty"`
	// The status and version of the Pineapple operator
	PineappleOperator []DataPlaneClusterUpdateStatusRequestPineappleOperator `json:"pineappleOperator,omitempty"`
}
