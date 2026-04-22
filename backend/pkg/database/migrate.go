package database

import (
	"database/sql"
	"embed"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/logging/log"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate runs all pending migrations up
func Migrate(db *sql.DB, step int) error {
	migration := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "migrations",
	}

	n, err := migrate.ExecMax(db, "mysql", migration, migrate.Up, step)
	if err != nil {
		return apperror.NewError("failed to run migrations").AddError(err)
	}

	log.Info().Msgf("applied %d migration(s)", n)
	return nil
}

// MigrateDown rolls back one migration
func MigrateDown(db *sql.DB, step int) error {
	migration := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "migrations",
	}

	n, err := migrate.ExecMax(db, "mysql", migration, migrate.Down, step)
	if err != nil {
		return apperror.NewError("failed to rollback migration").AddError(err)
	}

	log.Info().Msgf("rolled back %d migration(s)", n)
	return nil
}

// MigrateStatus returns the current migration status
func MigrateStatus(db *sql.DB) ([]*migrate.MigrationRecord, error) {
	records, err := migrate.GetMigrationRecords(db, "mysql")
	if err != nil {
		return nil, apperror.NewError("failed to get migration status").AddError(err)
	}

	return records, nil
}

// PlanMigration plans the migrations in the given direction
func PlanMigration(db *sql.DB, dir int, step int) ([]*migrate.PlannedMigration, error) {
	migration := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "migrations",
	}
	plans, _, err := migrate.PlanMigration(db, "mysql", migration, migrate.MigrationDirection(dir), step)
	if err != nil {
		return nil, apperror.NewError("failed to plan migrations").AddError(err)
	}
	return plans, nil
}
