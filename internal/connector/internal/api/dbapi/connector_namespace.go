package dbapi

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"time"
)

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
	Version    int64  `gorm:"type:bigserial;index"`
	Expiration *time.Time

	// metadata
	Annotations []ConnectorNamespaceAnnotation `gorm:"foreignKey:NamespaceId;references:ID"`

	// tenant, only one of the below fields can be not null
	TenantUserId         *string                      `gorm:"index:connector_namespaces_user_organisation_idx;index:,where:tenant_user_id is not null"`
	TenantOrganisationId *string                      `gorm:"index:connector_namespaces_user_organisation_idx;index:,where:tenant_organisation_id is not null"`
	TenantUser           *ConnectorTenantUser         `gorm:"foreignKey:TenantUserId"`
	TenantOrganisation   *ConnectorTenantOrganisation `gorm:"foreignKey:TenantOrganisationId"`
}

type ConnectorNamespaceList []*ConnectorNamespace
