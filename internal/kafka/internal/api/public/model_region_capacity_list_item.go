/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage Kafka instances.
 *
 * API version: 1.3.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// RegionCapacityListItem schema for a kafka instance type capacity in region
type RegionCapacityListItem struct {
	// kafka instance type
	InstanceType string `json:"instance_type,omitempty"`
	// flag indicating whether the capacity for the instance type in the region is reached
	MaxCapacityReached bool `json:"max_capacity_reached"`
}
