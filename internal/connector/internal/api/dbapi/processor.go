package dbapi

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

type ProcessorDesiredState string
type ProcessorStatusPhase string

const (
	ProcessorUnassigned ProcessorDesiredState = "unassigned"
	ProcessorReady      ProcessorDesiredState = "ready"
	ProcessorStopped    ProcessorDesiredState = "stopped"
	ProcessorDeleted    ProcessorDesiredState = "deleted"

	ProcessorStatusPhaseAssigning      ProcessorStatusPhase = "assigning"      // set by kas-fleet-manager - user request
	ProcessorStatusPhaseAssigned       ProcessorStatusPhase = "assigned"       // set by kas-fleet-manager - worker
	ProcessorStatusPhaseUpdating       ProcessorStatusPhase = "updating"       // set by kas-fleet-manager - user request
	ProcessorStatusPhaseStopped        ProcessorStatusPhase = "stopped"        // set by kas-fleet-manager - user request
	ProcessorStatusPhaseProvisioning   ProcessorStatusPhase = "provisioning"   // set by kas-agent
	ProcessorStatusPhaseReady          ProcessorStatusPhase = "ready"          // set by the agent
	ProcessorStatusPhaseFailed         ProcessorStatusPhase = "failed"         // set by the agent
	ProcessorStatusPhaseDeprovisioning ProcessorStatusPhase = "deprovisioning" // set by kas-agent
	ProcessorStatusPhaseDeleting       ProcessorStatusPhase = "deleting"       // set by the kas-fleet-manager - user request
	ProcessorStatusPhaseDeleted        ProcessorStatusPhase = "deleted"        // set by the agent
)

var ValidProcessorDesiredStates = []string{
	string(ProcessorUnassigned),
	string(ProcessorReady),
	string(ProcessorStopped),
	string(ProcessorDeleted),
}

var AgentProcessorStatusPhase = []string{
	string(ProcessorStatusPhaseProvisioning),
	string(ProcessorStatusPhaseDeprovisioning),
	string(ProcessorStatusPhaseStopped),
	string(ProcessorStatusPhaseReady),
	string(ProcessorStatusPhaseFailed),
	string(ProcessorStatusPhaseDeleted),
}

type Processor struct {
	db.Model

	NamespaceId   *string
	CloudProvider string
	Region        string
	MultiAZ       bool

	Name           string
	Owner          string
	OrganisationId string
	Version        int64                 `gorm:"type:bigserial;index:"`
	Annotations    []ProcessorAnnotation `gorm:"foreignKey:ProcessorID;references:ID"`

	ProcessorSpec  api.JSON `gorm:"type:jsonb"`
	DesiredState   ProcessorDesiredState
	ServiceAccount ServiceAccount `gorm:"embedded;embeddedPrefix:service_account_"`

	Status ProcessorStatus `gorm:"foreignKey:ID"`
}

type ProcessorAnnotation struct {
	ProcessorID string `gorm:"primaryKey;index"`
	Key         string `gorm:"primaryKey;not null"`
	Value       string `gorm:"not null"`
}

type ProcessorStatus struct {
	db.Model
	NamespaceID *string
	Phase       ProcessorStatusPhase
}

type ProcessorList []*Processor

type ProcessorWithConditions struct {
	Processor
	Conditions api.JSON `gorm:"type:jsonb"`
}

type ProcessorWithConditionsList []*ProcessorWithConditions

// ProcessorDeployment holds the deployment configuration of a processor
type ProcessorDeployment struct {
	db.Model
	Version                  int64
	ProcessorID              string
	Processor                Processor
	OperatorID               string
	ProcessorVersion         int64
	ProcessorShardMetadataID int64
	ProcessorShardMetadata   ProcessorShardMetadata
	ClusterID                string
	NamespaceID              string
	AllowUpgrade             bool
	Status                   ProcessorDeploymentStatus `gorm:"foreignKey:ID;references:ID"`
	Annotations              []ProcessorAnnotation     `gorm:"foreignKey:ProcessorID;references:ProcessorID"`
}

type ProcessorShardMetadata struct {
	ID             int64 `gorm:"primaryKey:autoIncrement"`
	Revision       int64 `gorm:"index:idx_typeid_channel_revision;default:0"`
	LatestRevision *int64
	ShardMetadata  api.JSON `gorm:"type:jsonb"`
}

type ProcessorDeploymentList []ProcessorDeployment

type ProcessorDeploymentStatus struct {
	db.Model
	Phase            ProcessorStatusPhase
	Version          int64
	Conditions       api.JSON `gorm:"type:jsonb"`
	Operators        api.JSON `gorm:"type:jsonb"`
	UpgradeAvailable bool
}

type ProcessorDeploymentOperatorUpgrade struct {
	ProcessorID  string                    `json:"connector_id,omitempty"`
	DeploymentID string                    `json:"deployment_id,omitempty"`
	NamespaceID  string                    `json:"namespace_id,omitempty"`
	Operator     *ProcessorOperatorUpgrade `json:"operator,omitempty"`
}

type ProcessorOperatorUpgrade struct {
	Assigned  ProcessorOperator `json:"assigned"`
	Available ProcessorOperator `json:"available"`
}

type ProcessorDeploymentOperatorUpgradeList []ProcessorDeploymentOperatorUpgrade

type ProcessorOperator struct {
	// the id of the operator
	Id string `json:"id,omitempty"`
	// the type of the operator
	Type string `json:"type,omitempty"`
	// the version of the operator
	Version string `json:"version,omitempty"`
}
