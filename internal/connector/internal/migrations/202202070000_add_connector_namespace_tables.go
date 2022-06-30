package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"time"
)

func addConnectorNamespaceTables(migrationId string) *gormigrate.Migration {

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

	return db.CreateMigrationFromActions(migrationId,
		db.CreateTableAction(&ConnectorTenantUser{}),
		db.CreateTableAction(&ConnectorTenantOrganisation{}),
		db.CreateTableAction(&ConnectorNamespaceAnnotation{}),
		db.CreateTableAction(&ConnectorNamespace{}),

		// connector namespace version
		db.ExecAction(`
			CREATE OR REPLACE FUNCTION connector_namespaces_version_trigger() RETURNS TRIGGER LANGUAGE plpgsql AS '
			BEGIN
			NEW.version := nextval(''connector_namespaces_version_seq'');
			RETURN NEW;
			END;'
		`, `
			DROP FUNCTION IF EXISTS connector_namespaces_version_trigger
		`),
		db.ExecAction(`DROP TRIGGER IF EXISTS connector_namespaces_version_trigger ON connector_namespaces`, ``),
		db.ExecAction(`
			CREATE TRIGGER connector_namespaces_version_trigger BEFORE INSERT OR UPDATE ON connector_namespaces
			FOR EACH ROW EXECUTE PROCEDURE connector_namespaces_version_trigger();
		`, `
			DROP TRIGGER IF EXISTS connector_namespaces_version_trigger ON connector_namespaces
		`),

		// foreign key relationship between namespaces and clusters
		db.ExecAction(
			"ALTER TABLE connector_namespaces DROP CONSTRAINT IF EXISTS fk_connector_namespaces_cluster_id",
			""),
		db.ExecAction(
			"ALTER TABLE connector_namespaces ADD CONSTRAINT fk_connector_namespaces_cluster_id "+
				"FOREIGN KEY (cluster_id) REFERENCES connector_clusters(id)",
			"ALTER TABLE connector_namespaces DROP CONSTRAINT IF EXISTS fk_connector_namespaces_cluster_id"),

		// only one of tenantUser or tenantOrg in ConnectorNamespace can be not null
		db.ExecAction(
			"ALTER TABLE connector_namespaces DROP CONSTRAINT IF EXISTS connector_namespaces_tenant_check",
			""),
		db.ExecAction(
			"ALTER TABLE connector_namespaces ADD CONSTRAINT connector_namespaces_tenant_check "+
				"CHECK (((tenant_user_id is not null)::integer + (tenant_organisation_id is not null)::integer) = 1)",
			"ALTER TABLE connector_namespaces DROP CONSTRAINT IF EXISTS connector_namespaces_tenant_check"),
	)
}
