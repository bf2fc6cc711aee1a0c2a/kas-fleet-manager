package dbapi

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"time"
)

type ConnectorNamespacePhaseEnum string

const (
	// ConnectorNamespacePhaseDisconnected - Namespace status when first created
	ConnectorNamespacePhaseDisconnected ConnectorNamespacePhaseEnum = "disconnected"
	// ConnectorNamespacePhaseReady- Namespace status when it operational
	ConnectorNamespacePhaseReady ConnectorNamespacePhaseEnum = "ready"
	// ConnectorNamespacePhaseDeleting- Namespace status when in the process of being deleted
	ConnectorNamespacePhaseDeleting ConnectorNamespacePhaseEnum = "deleting"
)

var AllConnectorNamespaceStatus = []ConnectorNamespacePhaseEnum{
	ConnectorNamespacePhaseDisconnected,
	ConnectorNamespacePhaseReady,
	ConnectorNamespacePhaseDeleting,
}

type ConnectorTenantUser struct {
	db.Model // user id in Id, required for references and data consistency
}

type ConnectorTenantOrganisation struct {
	db.Model // org id in Id, required for references and data consistency
}

type ConnectorNamespaceAnnotation struct {
	NamespaceId string `gorm:"primaryKey;index"`
	Name        string `gorm:"primaryKey;not null"`
	Value       string `gorm:"not null"`
}

type ConnectorNamespace struct {
	db.Model
	Name      string `gorm:"not null;uniqueIndex:idx_connector_namespaces_name_cluster_id"`
	ClusterId string `gorm:"not null;uniqueIndex:idx_connector_namespaces_name_cluster_id;index"`

	Owner      string `gorm:"not null;index"`
	Expiration *time.Time

	// metadata
	Annotations []ConnectorNamespaceAnnotation `gorm:"foreignKey:NamespaceId;references:ID"`

	// tenant, only one of the below fields can be not null
	TenantUserId         *string                      `gorm:"index:connector_namespaces_user_organisation_idx;index:,where:tenant_user_id is not null"`
	TenantOrganisationId *string                      `gorm:"index:connector_namespaces_user_organisation_idx;index:,where:tenant_organisation_id is not null"`
	TenantUser           *ConnectorTenantUser         `gorm:"foreignKey:TenantUserId"`
	TenantOrganisation   *ConnectorTenantOrganisation `gorm:"foreignKey:TenantOrganisationId"`

	Status ConnectorNamespaceStatus `gorm:"embedded;embeddedPrefix:status_"`
}

type ConnectorNamespaceStatus struct {
	Phase ConnectorNamespacePhaseEnum `gorm:"not null;index"`
	// the version of the agent
	Version            string
	ConnectorsDeployed int32         `gorm:"not null"`
	Conditions         ConditionList `gorm:"type:jsonb"`
}

type ConnectorNamespaceList []*ConnectorNamespace
