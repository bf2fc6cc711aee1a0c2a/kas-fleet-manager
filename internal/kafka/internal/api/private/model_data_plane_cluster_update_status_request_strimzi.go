/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager APIs that are used by internal services e.g kas-fleetshard operators.
 *
 * API version: 1.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// DataPlaneClusterUpdateStatusRequestStrimzi struct for DataPlaneClusterUpdateStatusRequestStrimzi
type DataPlaneClusterUpdateStatusRequestStrimzi struct {
	Ready   bool   `json:"ready"`
	Version string `json:"version"`
}
