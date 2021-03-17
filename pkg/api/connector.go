package api

type ConnectorStatusPhase = string

const (
	ConnectorStatusPhaseAssigning    ConnectorStatusPhase = "assigning"    // set by kas-fleet-manager - user request
	ConnectorStatusPhaseAssigned     ConnectorStatusPhase = "assigned"     // set by kas-fleet-manager - worker
	ConnectorStatusPhaseProvisioning ConnectorStatusPhase = "provisioning" // set by kas-fleet-manager -  user request
	ConnectorStatusPhaseReady        ConnectorStatusPhase = "ready"        // set by the agent
	ConnectorStatusPhaseFailed       ConnectorStatusPhase = "failed"       // set by the agent
	ConnectorStatusPhaseDeleting     ConnectorStatusPhase = "deleting"     // set by the kas-fleet-manager - user request
	ConnectorStatusPhaseDeleted      ConnectorStatusPhase = "deleted"      // set by the agent
)

var AgentSetConnectorStatusPhase = []ConnectorStatusPhase{
	ConnectorStatusPhaseReady,
	ConnectorStatusPhaseFailed,
	ConnectorStatusPhaseDeleted,
}

type TargetKind = string

const (
	AddonTargetKind         TargetKind = "addon"
	CloudProviderTargetKind TargetKind = "cloud_provider"
)

var AllTargetKind = []TargetKind{
	AddonTargetKind,
	CloudProviderTargetKind,
}

type Connector struct {
	Meta

	TargetKind     TargetKind `json:"target_kind"`
	AddonClusterId string     `json:"addon_cluster_id"`
	CloudProvider  string     `json:"cloud_provider"`
	Region         string     `json:"region"`
	MultiAZ        bool       `json:"multi_az"`

	Name           string `json:"name"`
	Owner          string `json:"owner"`
	OrganisationId string `json:"organisation_id"`
	KafkaID        string `json:"kafka_id"`
	Version        int64  `json:"version"`

	ConnectorTypeId string `json:"connector_type_id,omitempty"`
	ConnectorSpec   JSON   `json:"connector_spec"`
	DesiredState    string `json:"desired_state"`

	Status ConnectorStatus `json:"status" gorm:"foreignkey:ID"`
}

type ConnectorStatus struct {
	Meta
	ClusterID string `json:"cluster_id"`
	Phase     string `json:"phase,omitempty"`
}

type ConnectorList []*Connector

// ConnectorDeployment Holds the deployment configuration of a connector
type ConnectorDeployment struct {
	Meta
	Version              int64                     `json:"version"`
	ConnectorID          string                    `json:"connector_id"`
	ClusterID            string                    `json:"cluster_id"`
	ConnectorTypeService string                    `json:"connector_type_service"`
	SpecChecksum         string                    `json:"spec_checksum,omitempty"`
	Status               ConnectorDeploymentStatus `json:"status" gorm:"foreignkey:ID"`
}

type ConnectorDeploymentList []ConnectorDeployment

type ConnectorDeploymentStatus struct {
	Meta
	Phase        string `json:"phase,omitempty"`
	SpecChecksum string `json:"spec_checksum,omitempty"`
	Conditions   JSON   `json:"conditions,omitempty"`
}

type ConnectorDeploymentSpecStatusExtractors struct {
	ApiVersion    string `json:"apiVersion,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Name          string `json:"name,omitempty"`
	JsonPath      string `json:"jsonPath,omitempty"`
	ConditionType string `json:"conditionType,omitempty"`
}
