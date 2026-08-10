package main

import (
	dbpkg "github.com/getevo/evo/v2/lib/db"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// sqliteDriver is the smallest db.Driver that satisfies evo.Setup: enough to
// open a file backed SQLite database, which is all this example needs. It uses
// the pure Go SQLite build, so it runs without a C toolchain.
type sqliteDriver struct{}

func (sqliteDriver) Name() string { return "sqlite" }

func (sqliteDriver) Open(config dbpkg.DriverConfig, gormConfig *gorm.Config) (*gorm.DB, error) {
	// The file name may be given as either the database or the server, so
	// accept both and fall back to an in-memory database.
	dsn := config.Database
	if dsn == "" {
		dsn = config.Server
	}
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := gorm.Open(sqlite.Open(dsn), gormConfig)
	if err != nil {
		return db, err
	}
	db.Exec("PRAGMA foreign_keys = ON")
	return db, nil
}

// GetMigrationScript returns nothing: this example creates its tables with
// AutoMigrate rather than through the framework's migration pipeline.
func (sqliteDriver) GetMigrationScript(db *gorm.DB) []string { return nil }
