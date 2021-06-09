package db

import (
	"fmt"
	"runtime"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// gormigrate is a wrapper for gorm's migration functions that adds schema versioning and rollback capabilities.
// For help writing migration steps, see the gorm documentation on migrations: https://gorm.io/docs/migration.html

// Migration rules:
//
// 1. IDs are numerical timestamps that must sort ascending.
//    Use YYYYMMDDHHMM w/ 24 hour time for format
//    Example: August 21 2018 at 2:54pm would be 201808211454.
//
// 2. Include models inline with migrations to see the evolution of the object over time.
//    Using our internal type models directly in the first migration would fail in future clean installs.
//
// 3. Migrations must be backwards compatible. There are no new required fields allowed.
//    See $project_home/db/README.md
//
// 4. Create one function in a separate file that returns your Migration. Add that single function call to this list.
var migrations []*gormigrate.Migration = []*gormigrate.Migration{
	addKafkaRequest(),
	addClusters(),
	updateKafkaMultiAZTypeToBoolean(),
	addKafkabootstrapServerHostType(),
	addClusterStatus(),
	addKafkaOrganisationId(),
	addLeaderLease(),
	addFailedReason(),
	addConnectors(),
	addKafkaPlacementId(),
	addConnectorClusters(),
	addKafkaSubscriptionId(),
	addClusterIdentityProviderID(),
	addKafkaSsoClientIdAndSecret(),
	addKafkaOwnerAccountId(),
	addKafkaVersion(),
	connectorApiChanges(),
	addMissingIndexes(),
	addClusterStatusIndex(),
	addKafkaWorkersInLeaderLeases(),
	renameDeletingKafkaLeaseType(),
	addClusterDNS(),
	connectorMigrations20210518(),
	addExternalIDsToSpecificClusters(),
	addKafkaConnectionSettingsToConnectors(),
	addClusterProviderInfo(),
	changeKafkaDeleteStatusToDeleting(),
	addKafkaQuotaTypeColumn(),
	addConnectorTypeChannel(),
}

func Migrate(conFactory *ConnectionFactory) {
	gorm := conFactory.New()

	m := newGormigrate(gorm)

	if err := m.Migrate(); err != nil {
		glog.Fatalf("Could not migrate: %v", err)
	}
}

// Migrating to a specific migration will not seed the database, seeds are up to date with the latest
// schema based on the most recent migration
// This should be for testing purposes mainly
func MigrateTo(conFactory *ConnectionFactory, migrationID string) {
	gorm := conFactory.New()
	m := newGormigrate(gorm)
	if err := m.MigrateTo(migrationID); err != nil {
		glog.Fatalf("Could not migrate: %v", err)
	}
}

// Rolls back all migrations..
func RollbackAll(conFactory *ConnectionFactory) {
	gorm := conFactory.New()
	m := newGormigrate(gorm)

	type Migration struct {
		ID string
	}
	var result Migration
	for {
		err := gorm.First(&result).Error
		if err != nil {
			return
		}
		if err := m.RollbackLast(); err != nil {
			glog.Fatalf("Could not rollback: %v", err)
		}
	}

}

func newGormigrate(db *gorm.DB) *gormigrate.Gormigrate {
	gormOptions := &gormigrate.Options{
		TableName:      "migrations",
		IDColumnName:   "id",
		IDColumnSize:   255,
		UseTransaction: false,
	}

	return gormigrate.New(db, gormOptions, migrations)
}

// Model represents the base model struct. All entities will have this struct embedded.
type Model struct {
	ID        string `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type MigrationAction func(tx *gorm.DB, apply bool) error

func CreateTableAction(table interface{}) MigrationAction {
	caller := ""
	if _, file, no, ok := runtime.Caller(1); ok {
		caller = fmt.Sprintf("[ %s:%d ]", file, no)
	}
	return func(tx *gorm.DB, apply bool) error {
		if apply {
			err := tx.AutoMigrate(table)
			if err != nil {
				return errors.Wrap(err, caller)
			}
		} else {
			err := tx.Migrator().DropTable(table)
			if err != nil {
				return errors.Wrap(err, caller)
			}
		}
		return nil
	}
}

func AddTableColumnsAction(table interface{}) MigrationAction {
	caller := ""
	if _, file, no, ok := runtime.Caller(1); ok {
		caller = fmt.Sprintf("[ %s:%d ]", file, no)
	}
	return func(tx *gorm.DB, apply bool) error {
		if apply {
			stmt := &gorm.Statement{DB: tx}
			if err := stmt.Parse(table); err != nil {
				return errors.Wrap(err, caller)
			}
			for _, field := range stmt.Schema.FieldsByDBName {
				if err := tx.Migrator().AddColumn(table, field.DBName); err != nil {
					return errors.Wrap(err, caller)
				}
			}
		} else {
			stmt := &gorm.Statement{DB: tx}
			if err := stmt.Parse(table); err != nil {
				return errors.Wrap(err, caller)
			}
			for _, field := range stmt.Schema.FieldsByDBName {
				if err := tx.Migrator().DropColumn(table, field.DBName); err != nil {
					return errors.Wrap(err, caller)
				}
			}
		}
		return nil

	}
}

func DropTableColumnsAction(table interface{}, tableName ...string) MigrationAction {
	caller := ""
	if _, file, no, ok := runtime.Caller(1); ok {
		caller = fmt.Sprintf("[ %s:%d ]", file, no)
	}
	return func(tx *gorm.DB, apply bool) error {
		if apply {
			stmt := &gorm.Statement{DB: tx}
			if err := stmt.Parse(table); err != nil {
				return errors.Wrap(err, caller)
			}
			if len(tableName) > 0 {
				stmt.Schema.Table = tableName[0]
			}
			for _, field := range stmt.Schema.FieldsByDBName {
				if err := tx.Migrator().DropColumn(table, field.DBName); err != nil {
					return errors.Wrap(err, caller)
				}
			}
		} else {
			stmt := &gorm.Statement{DB: tx}
			if err := stmt.Parse(table); err != nil {
				return errors.Wrap(err, caller)
			}
			if len(tableName) > 0 {
				stmt.Schema.Table = tableName[0]
			}
			for _, field := range stmt.Schema.FieldsByDBName {
				if err := tx.Migrator().AddColumn(table, field.DBName); err != nil {
					return errors.Wrap(err, caller)
				}
			}
		}
		return nil

	}
}

func ExecAction(applySql string, unapplySql string) MigrationAction {
	caller := ""
	if _, file, no, ok := runtime.Caller(1); ok {
		caller = fmt.Sprintf("[ %s:%d ]", file, no)
	}

	return func(tx *gorm.DB, apply bool) error {
		if apply {
			err := tx.Exec(applySql).Error
			if err != nil {
				return errors.Wrap(err, caller)
			}
		} else {
			err := tx.Exec(unapplySql).Error
			if err != nil {
				return errors.Wrap(err, caller)
			}
		}
		return nil

	}
}

func CreateMigrationFromActions(id string, actions ...MigrationAction) *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: id,
		Migrate: func(tx *gorm.DB) error {
			for _, action := range actions {
				err := action(tx, true)
				if err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			for i := len(actions) - 1; i >= 0; i-- {
				err := actions[i](tx, false)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}
