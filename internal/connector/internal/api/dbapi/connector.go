package dbapi

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

type ConnectorStatusPhase = string

const (
	ConnectorStatusPhaseAssigning      ConnectorStatusPhase = "assigning"      // set by kas-fleet-manager - user request
	ConnectorStatusPhaseAssigned       ConnectorStatusPhase = "assigned"       // set by kas-fleet-manager - worker
	ConnectorStatusPhaseUpdating       ConnectorStatusPhase = "updating"       // set by kas-fleet-manager - user request
	ConnectorStatusPhaseStopped        ConnectorStatusPhase = "stopped"        // set by kas-fleet-manager - user request
	ConnectorStatusPhaseProvisioning   ConnectorStatusPhase = "provisioning"   // set by kas-agent
	ConnectorStatusPhaseReady          ConnectorStatusPhase = "ready"          // set by the agent
	ConnectorStatusPhaseFailed         ConnectorStatusPhase = "failed"         // set by the agent
	ConnectorStatusPhaseDeprovisioning ConnectorStatusPhase = "deprovisioning" // set by kas-agent
	ConnectorStatusPhaseDeleting       ConnectorStatusPhase = "deleting"       // set by the kas-fleet-manager - user request
	ConnectorStatusPhaseDeleted        ConnectorStatusPhase = "deleted"        // set by the agent
)

var ValidDesiredStates = []ConnectorStatusPhase{
	ConnectorStatusPhaseReady,
	ConnectorStatusPhaseStopped,
	ConnectorStatusPhaseDeleted,
}

var AgentSetConnectorStatusPhase = []ConnectorStatusPhase{
	ConnectorStatusPhaseProvisioning,
	ConnectorStatusPhaseDeprovisioning,
	ConnectorStatusPhaseStopped,
	ConnectorStatusPhaseReady,
	ConnectorStatusPhaseFailed,
	ConnectorStatusPhaseDeleted,
}

type Connector struct {
	api.Meta

	NamespaceId   *string
	CloudProvider string
	Region        string
	MultiAZ       bool

	Name           string
	Owner          string
	OrganisationId string
	Version        int64 `gorm:"type:bigserial;index:"`

	ConnectorTypeId string
	ConnectorSpec   api.JSON `gorm:"type:jsonb"`
	DesiredState    string
	Channel         string
	Kafka           KafkaConnectionSettings          `gorm:"embedded;embeddedPrefix:kafka_"`
	SchemaRegistry  SchemaRegistryConnectionSettings `gorm:"embedded;embeddedPrefix:schema_registry_"`
	ServiceAccount  ServiceAccount                   `gorm:"embedded;embeddedPrefix:service_account_"`

	Status ConnectorStatus `gorm:"foreignKey:ID"`
}

type ConnectorStatus struct {
	api.Meta
	NamespaceID *string
	Phase       string
}

type ConnectorList []*Connector

// ConnectorDeployment Holds the deployment configuration of a connector
type ConnectorDeployment struct {
	api.Meta
	Version                int64
	ConnectorID            string
	OperatorID             string
	ConnectorVersion       int64
	ConnectorTypeChannelId int64
	ClusterID              string
	NamespaceID            string
	NamespaceName          string `gorm:"->"` // readonly field, used to join connector_namespaces
	AllowUpgrade           bool
	Status                 ConnectorDeploymentStatus `gorm:"foreignKey:ID"`
}

type ConnectorDeploymentList []ConnectorDeployment

type ConnectorDeploymentStatus struct {
	api.Meta
	Phase            string
	Version          int64
	Conditions       api.JSON `gorm:"type:jsonb"`
	Operators        api.JSON `gorm:"type:jsonb"`
	UpgradeAvailable bool
}

type KafkaConnectionSettings struct {
	KafkaID         string `gorm:"column:id"`
	BootstrapServer string
}

type SchemaRegistryConnectionSettings struct {
	SchemaRegistryID string `gorm:"column:id"`
	Url              string
}

type ServiceAccount struct {
	ClientId        string
	ClientSecret    string `gorm:"-"`
	ClientSecretRef string `gorm:"column:client_secret"`
}

type ConnectorDeploymentTypeUpgrade struct {
	ConnectorID     string                `json:"connector_id,omitempty"`
	DeploymentID    string                `json:"deployment_id,omitempty"`
	ConnectorTypeId string                `json:"connector_type_id,omitempty"`
	NamespaceID     string                `json:"namespace_id,omitempty"`
	Channel         string                `json:"channel,omitempty"`
	ShardMetadata   *ConnectorTypeUpgrade `json:"shard_metadata,omitempty"`
}

type ConnectorTypeUpgrade struct {
	AssignedId  int64 `json:"assigned_id,omitempty"`
	AvailableId int64 `json:"available_id,omitempty"`
}

type ConnectorDeploymentTypeUpgradeList []ConnectorDeploymentTypeUpgrade

type ConnectorDeploymentOperatorUpgrade struct {
	ConnectorID     string                    `json:"connector_id,omitempty"`
	DeploymentID    string                    `json:"deployment_id,omitempty"`
	ConnectorTypeId string                    `json:"connector_type_id,omitempty"`
	NamespaceID     string                    `json:"namespace_id,omitempty"`
	Channel         string                    `json:"channel,omitempty"`
	Operator        *ConnectorOperatorUpgrade `json:"operator,omitempty"`
}

type ConnectorOperatorUpgrade struct {
	Assigned  ConnectorOperator `json:"assigned"`
	Available ConnectorOperator `json:"available"`
}

type ConnectorDeploymentOperatorUpgradeList []ConnectorDeploymentOperatorUpgrade

type ConnectorOperator struct {
	// the id of the operator
	Id string `json:"id,omitempty"`
	// the type of the operator
	Type string `json:"type,omitempty"`
	// the version of the operator
	Version string `json:"version,omitempty"`
}
