package db

import (
	"time"

	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

// gormigrate is a wrapper for gorm's migration functions that adds schema versioning and rollback capabilities.
// For help writing migration steps, see the gorm documentation on migrations: http://doc.gorm.io/database.html#migration

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
	DeletedAt *time.Time `sql:"index"`
}
