package dbapi

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

type ConnectorStatusPhase = string

const (
	ConnectorStatusPhaseAssigning    ConnectorStatusPhase = "assigning"    // set by kas-fleet-manager - user request
	ConnectorStatusPhaseAssigned     ConnectorStatusPhase = "assigned"     // set by kas-fleet-manager - worker
	ConnectorStatusPhaseUpdating     ConnectorStatusPhase = "updating"     // set by kas-fleet-manager - user request
	ConnectorStatusPhaseProvisioning ConnectorStatusPhase = "provisioning" // set by kas-agent
	ConnectorStatusPhaseReady        ConnectorStatusPhase = "ready"        // set by the agent
	ConnectorStatusPhaseFailed       ConnectorStatusPhase = "failed"       // set by the agent
	ConnectorStatusPhaseDeleting     ConnectorStatusPhase = "deleting"     // set by the kas-fleet-manager - user request
	ConnectorStatusPhaseDeleted      ConnectorStatusPhase = "deleted"      // set by the agent
)

var AgentSetConnectorStatusPhase = []ConnectorStatusPhase{
	ConnectorStatusPhaseProvisioning,
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
	api.Meta

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

	ConnectorTypeId string                  `json:"connector_type_id,omitempty"`
	ConnectorSpec   api.JSON                `json:"connector_spec"`
	DesiredState    string                  `json:"desired_state"`
	Channel         string                  `json:"channel,omitempty"`
	Kafka           KafkaConnectionSettings `json:"kafka,omitempty" gorm:"embedded;embeddedPrefix:kafka_"`

	Status ConnectorStatus `json:"status" gorm:"foreignKey:ID"`
}

type ConnectorStatus struct {
	api.Meta
	ClusterID string `json:"cluster_id"`
	Phase     string `json:"phase,omitempty"`
}

type ConnectorList []*Connector

// ConnectorDeployment Holds the deployment configuration of a connector
type ConnectorDeployment struct {
	api.Meta
	Version                int64                     `json:"version"`
	ConnectorID            string                    `json:"connector_id"`
	ConnectorVersion       int64                     `json:"connector_version,omitempty"`
	ConnectorTypeChannelId int64                     `json:"connector_type_channel_id,omitempty"`
	ClusterID              string                    `json:"cluster_id"`
	AllowUpgrade           bool                      `json:"allow_upgrade,omitempty"`
	Status                 ConnectorDeploymentStatus `json:"status" gorm:"foreignKey:ID"`
}

type ConnectorDeploymentList []ConnectorDeployment

type ConnectorDeploymentStatus struct {
	api.Meta
	Phase            string   `json:"phase,omitempty"`
	Version          int64    `json:"version"`
	Conditions       api.JSON `json:"conditions,omitempty"`
	Operators        api.JSON `json:"operators,omitempty"`
	UpgradeAvailable bool     `json:"upgrade_available,omitempty"`
}

type ConnectorDeploymentSpecStatusExtractors struct {
	ApiVersion    string `json:"apiVersion,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Name          string `json:"name,omitempty"`
	JsonPath      string `json:"jsonPath,omitempty"`
	ConditionType string `json:"conditionType,omitempty"`
}

type KafkaConnectionSettings struct {
	BootstrapServer string `json:"bootstrap_server,omitempty"`
	ClientId        string `json:"client_id,omitempty"`
	ClientSecret    string `json:"client_secret,omitempty" gorm:"-"`
	ClientSecretRef string `json:"client_secret_ref,omitempty" gorm:"column:client_secret"`
}

type ConnectorDeploymentAvailableUpgrades struct {
	DeploymentID    string                    `json:"deployment_id,omitempty"`
	ConnectorTypeId string                    `json:"connector_type_id,omitempty"`
	Channel         string                    `json:"channel,omitempty"`
	ShardMetadata   *ConnectorTypeUpgrade     `json:"shard_metadata,omitempty"`
	Operator        *ConnectorOperatorUpgrade `json:"operator,omitempty"`
}
type ConnectorTypeUpgrade struct {
	AssignedId  int64 `json:"assigned_id,omitempty"`
	AvailableId int64 `json:"available_id,omitempty"`
}
type ConnectorOperatorUpgrade struct {
	Assigned  ConnectorOperator `json:"assigned"`
	Available ConnectorOperator `json:"available"`
}
type ConnectorOperator struct {
	// the id of the operator
	Id string `json:"id,omitempty"`
	// the type of the operator
	Type string `json:"type,omitempty"`
	// the version of the operator
	Version string `json:"version,omitempty"`
}
