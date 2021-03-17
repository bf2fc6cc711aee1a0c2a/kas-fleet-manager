package api

type ConnectorStatus = string

const (
	ConnectorStatusAssigning ConnectorStatus = "assigning" // set by kas-fleet-manager - user request
	ConnectorStatusAssigned  ConnectorStatus = "assigned"  // set by kas-fleet-manager - worker
	ConnectorStatusReady     ConnectorStatus = "ready"     // set by the agent
	ConnectorStatusFailed    ConnectorStatus = "failed"    // set by the agent
	ConnectorStatusDeleting  ConnectorStatus = "deleting"  // set by the kas-fleet-manager - user request
	ConnectorStatusDeleted   ConnectorStatus = "deleted"   // set by the agent
)

var AgentSetConnectorStatus = []ConnectorStatus{
	ConnectorStatusReady,
	ConnectorStatusFailed,
	ConnectorStatusDeleted,
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
	ConnectorTypeId string          `json:"connector_type_id,omitempty"`
	ConnectorSpec   JSON            `json:"connector_spec"`
	Region          string          `json:"region"`
	ClusterID       string          `json:"cluster_id"`
	CloudProvider   string          `json:"cloud_provider"`
	MultiAZ         bool            `json:"multi_az"`
	Name            string          `json:"name"`
	Status          ConnectorStatus `json:"status"`
	Owner           string          `json:"owner"`
	OrganisationId  string          `json:"organisation_id"`
	KafkaID         string          `json:"kafka_id"`
	Version         int64           `json:"version"`
	TargetKind      TargetKind      `json:"target_kind"`
	AddonClusterId  string          `json:"addon_cluster_id"`
}

type ConnectorList []*Connector

// ConnectorDeployment Holds the deployment configuration of a connector
type ConnectorDeployment struct {
	Meta
	Version              int64                     `json:"version"`
	ClusterID            string                    `json:"cluster_id"`
	ConnectorTypeService string                    `json:"connector_type_service"`
	Spec                 ConnectorDeploymentSpec   `json:"spec,omitempty" gorm:"embedded;embedded_prefix:spec_"`
	Status               ConnectorDeploymentStatus `json:"status,omitempty" gorm:"embedded;embedded_prefix:status_"`
}

type ConnectorDeploymentSpec struct {
	ConnectorId      string
	OperatorsIds     JSON `gorm:"type:jsonb"`
	Resources        JSON `gorm:"type:jsonb"`
	StatusExtractors JSON `gorm:"type:jsonb"`
}

type ConnectorDeploymentList []ConnectorDeployment

type ConnectorDeploymentStatus struct {
	Phase      string `json:"phase,omitempty"`
	Conditions JSON   `json:"conditions,omitempty"`
}

type ConnectorDeploymentSpecStatusExtractors struct {
	ApiVersion    string `json:"apiVersion,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Name          string `json:"name,omitempty"`
	JsonPath      string `json:"jsonPath,omitempty"`
	ConditionType string `json:"conditionType,omitempty"`
}
