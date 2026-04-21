package database

import (
	"database/sql"
	"embed"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/logging/log"
)

//go:embed schema/migrations/*.sql
var migrations embed.FS

// Migrate runs all pending migrations up
func Migrate(db *sql.DB) error {
	migration := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "schema/migrations",
	}

	n, err := migrate.Exec(db, "mysql", migration, migrate.Up)
	if err != nil {
		return apperror.NewError("failed to run migrations").AddError(err)
	}

	log.Info().Msgf("Applied %d migration(s)", n)
	return nil
}

// MigrateDown rolls back one migration
func MigrateDown(db *sql.DB) error {
	migration := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "schema/migrations",
	}

	n, err := migrate.ExecMax(db, "mysql", migration, migrate.Down, 1)
	if err != nil {
		return apperror.NewError("failed to rollback migration").AddError(err)
	}

	log.Info().Msgf("Rolled back %d migration(s)", n)
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
func PlanMigration(db *sql.DB, dir int) ([]*migrate.PlannedMigration, error) {
	migration := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrations,
		Root:       "schema/migrations",
	}
	plans, _, err := migrate.PlanMigration(db, "mysql", migration, migrate.MigrationDirection(dir), 0)
	if err != nil {
		return nil, apperror.NewError("failed to plan migrations").AddError(err)
	}
	return plans, nil
}
