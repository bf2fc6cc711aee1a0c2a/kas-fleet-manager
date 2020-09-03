package api

type Cluster struct {
	Meta
	CloudProvider     string `json:"cloud_provider"`
	ClusterID         string `json:"cluster_id"`   // maps to id in an osd cluster object
	ClusterUUID       string `json:"cluster_uuid"` // maps to external id in an osd cluster object
	MultiAZ           string `json:"multiAZ"`
	Region            string `json:"region"`
	Status            string `json:"status"`
	AvailableCapacity *AvailableClusterCapacity
}

type AvailableClusterCapacity struct {
	// Storage, etc.
}
