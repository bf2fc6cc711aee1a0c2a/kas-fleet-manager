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

type Migration struct {
	DbFactory   *ConnectionFactory
	Gormigrate  *gormigrate.Gormigrate
	GormOptions *gormigrate.Options
}

func NewMigration(dbConfig *DatabaseConfig, gormOptions *gormigrate.Options, migrations []*gormigrate.Migration) (*Migration, func(), error) {
	err := dbConfig.ReadFiles()
	if err != nil {
		return nil, nil, err
	}
	dbFactory, cleanup := NewConnectionFactory(dbConfig)

	return &Migration{
		DbFactory:   dbFactory,
		GormOptions: gormOptions,
		Gormigrate:  gormigrate.New(dbFactory.New(), gormOptions, migrations),
	}, cleanup, nil
}

func (m *Migration) Migrate() {
	if err := m.Gormigrate.Migrate(); err != nil {
		glog.Fatalf("Could not migrate: %v", err)
	}
}

// Migrating to a specific migration will not seed the database, seeds are up to date with the latest
// schema based on the most recent migration
// This should be for testing purposes mainly
func (m *Migration) MigrateTo(migrationID string) {
	if err := m.Gormigrate.MigrateTo(migrationID); err != nil {
		glog.Fatalf("Could not migrate: %v", err)
	}
}

func (m *Migration) RollbackLast() {
	if err := m.Gormigrate.RollbackLast(); err != nil {
		glog.Fatalf("Could not migrate: %v", err)
	}
	m.deleteMigrationTableIfEmpty(m.DbFactory.New())
}

func (m *Migration) RollbackTo(migrationID string) {
	if err := m.Gormigrate.RollbackTo(migrationID); err != nil {
		glog.Fatalf("Could not migrate: %v", err)
	}
}

// Rolls back all migrations..
func (m *Migration) RollbackAll() {
	db := m.DbFactory.New()
	type Result struct {
		ID string
	}
	sql := fmt.Sprintf("SELECT %s AS id FROM %s", m.GormOptions.IDColumnName, m.GormOptions.TableName)
	for {
		var result Result
		err := db.Raw(sql).Scan(&result).Error
		if err != nil || result.ID == "" {
			break
		}
		if err := m.Gormigrate.RollbackLast(); err != nil {
			glog.Fatalf("Could not rollback last migration: %v", err)
		}
	}
	m.deleteMigrationTableIfEmpty(db)
}

func (m *Migration) deleteMigrationTableIfEmpty(db *gorm.DB) {
	if !db.Migrator().HasTable(m.GormOptions.TableName) {
		return
	}
	result := m.CountMigrationsApplied()
	if result == 0 {
		if err := db.Migrator().DropTable(m.GormOptions.TableName); err != nil {
			glog.Fatalf("Could not drop migration table: %v", err)
		}
	}
}

func (m *Migration) CountMigrationsApplied() int {
	db := m.DbFactory.New()
	if !db.Migrator().HasTable(m.GormOptions.TableName) {
		return 0
	}
	sql := fmt.Sprintf("SELECT count(%s) AS id FROM %s", m.GormOptions.IDColumnName, m.GormOptions.TableName)
	var count int
	if err := db.Raw(sql).Scan(&count).Error; err != nil {
		glog.Fatalf("Could not get migration count: %v", err)
	}
	return count
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
			if applySql != "" {
				err := tx.Exec(applySql).Error
				if err != nil {
					return errors.Wrap(err, caller)
				}
			}
		} else {
			if unapplySql != "" {
				err := tx.Exec(unapplySql).Error
				if err != nil {
					return errors.Wrap(err, caller)
				}
			}
		}
		return nil

	}
}

func FuncAction(applyFunc func(*gorm.DB) error, unapplyFunc func(*gorm.DB) error) MigrationAction {
	caller := ""
	if _, file, no, ok := runtime.Caller(1); ok {
		caller = fmt.Sprintf("[ %s:%d ]", file, no)
	}

	return func(tx *gorm.DB, apply bool) error {
		if apply {
			err := applyFunc(tx)
			if err != nil {
				return errors.Wrap(err, caller)
			}
		} else {
			err := unapplyFunc(tx)
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
