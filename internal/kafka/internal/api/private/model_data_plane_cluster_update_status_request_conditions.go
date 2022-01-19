/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.4.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// DataPlaneClusterUpdateStatusRequestConditions struct for DataPlaneClusterUpdateStatusRequestConditions
type DataPlaneClusterUpdateStatusRequestConditions struct {
	Type    string `json:"type,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}