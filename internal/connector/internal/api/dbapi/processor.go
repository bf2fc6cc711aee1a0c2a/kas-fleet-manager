package dbapi

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
)

type ProcessorDesiredState string
type ProcessorStatusPhase string

const (
	ProcessorReady   ProcessorDesiredState = "ready"
	ProcessorStopped ProcessorDesiredState = "stopped"
	ProcessorDeleted ProcessorDesiredState = "deleted"

	ProcessorStatusPhasePreparing      ProcessorStatusPhase = "preparing"      // set by kas-fleet-manager - user request
	ProcessorStatusPhasePrepared       ProcessorStatusPhase = "prepared"       // set by kas-fleet-manager - worker
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
	Name            string
	NamespaceId     string
	ProcessorTypeId string
	Channel         string
	DesiredState    ProcessorDesiredState
	Owner           string
	OrganisationId  string
	Version         int64                   `gorm:"type:bigserial;index"`
	Annotations     []ProcessorAnnotation   `gorm:"foreignKey:ProcessorID;references:ID"`
	Kafka           KafkaConnectionSettings `gorm:"embedded;embeddedPrefix:kafka_"`
	ServiceAccount  ServiceAccount          `gorm:"embedded;embeddedPrefix:service_account_"`
	Definition      api.JSON                `gorm:"definition"`
	ErrorHandler    api.JSON                `gorm:"error_handler"`
	Status          ProcessorStatus         `gorm:"foreignKey:ID"`
}

type ProcessorAnnotation struct {
	ProcessorID string `gorm:"primaryKey;index"`
	Key         string `gorm:"primaryKey;index;not null"`
	Value       string `gorm:"not null"`
}

type ProcessorStatus struct {
	db.Model
	NamespaceID string
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
	Version                  int64  `gorm:"type:bigserial;index"`
	ProcessorID              string `gorm:"index:idx_processor_deployments_processor_id_idx;unique"`
	Processor                Processor
	ProcessorVersion         int64
	ProcessorShardMetadataID int64
	ProcessorShardMetadata   ProcessorShardMetadata
	ClusterID                string
	NamespaceID              string
	OperatorID               string
	AllowUpgrade             bool
	Status                   ProcessorDeploymentStatus `gorm:"foreignKey:ID;references:ID"`
}

type ProcessorShardMetadata struct {
	ID              int64  `gorm:"primaryKey:autoIncrement"`
	ProcessorTypeId string `gorm:"index:idx_typeid_channel_revision;index:idx_typeid_channel"`
	Channel         string `gorm:"index:idx_typeid_channel_revision;index:idx_typeid_channel"`
	Revision        int64  `gorm:"index:idx_processor_shard_revision;default:0"`
	LatestRevision  *int64
	ShardMetadata   api.JSON `gorm:"type:jsonb"`
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
