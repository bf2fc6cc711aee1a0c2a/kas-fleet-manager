/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// DatePlaneClusterUpdateStatusRequestDeprecatedResizeInfo struct for DatePlaneClusterUpdateStatusRequestDeprecatedResizeInfo
type DatePlaneClusterUpdateStatusRequestDeprecatedResizeInfo struct {
	// Deprecated
	DeprecatedNodeDelta *int32                                                        `json:"nodeDelta,omitempty"`
	Delta               *DatePlaneClusterUpdateStatusRequestDeprecatedResizeInfoDelta `json:"delta,omitempty"`
}
