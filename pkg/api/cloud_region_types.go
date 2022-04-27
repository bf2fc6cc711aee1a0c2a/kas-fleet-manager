package api

type CloudRegion struct {
	Kind                   string                   `json:"kind"`
	Id                     string                   `json:"id"`
	DisplayName            string                   `json:"display_name"`
	CloudProvider          string                   `json:"cloud_provider"`
	Enabled                bool                     `json:"enabled"`
	SupportedInstanceTypes []string                 `json:"supported_instance_types"`
	Capacity               []RegionCapacityListItem `json:"capacity"`
}

type CloudRegionList *[]CloudRegion

// RegionCapacityListItem schema for a kafka instance type capacity in region
type RegionCapacityListItem struct {
	// kafka instance type
	InstanceType string `json:"instance_type,omitempty"`
	// flag indicating whether the capacity for the instance type in the region is reached
	DeprecatedMaxCapacityReached bool `json:"max_capacity_reached,omitempty"`
	// a list of kafka instance sizes that can still be created in this region
	AvailableSizes []string `json:"available_sizes,omitempty"`
}
