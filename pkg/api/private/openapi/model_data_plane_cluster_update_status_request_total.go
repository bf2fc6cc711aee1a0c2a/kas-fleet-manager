/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 1.1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

// DataPlaneClusterUpdateStatusRequestTotal struct for DataPlaneClusterUpdateStatusRequestTotal
type DataPlaneClusterUpdateStatusRequestTotal struct {
	IngressEgressThroughputPerSec *string `json:"ingressEgressThroughputPerSec,omitempty"`
	Connections                   *int32  `json:"connections,omitempty"`
	DataRetentionSize             *string `json:"dataRetentionSize,omitempty"`
	Partitions                    *int32  `json:"partitions,omitempty"`
}
