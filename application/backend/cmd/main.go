package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/interruption"
	"github.com/valentin-kaiser/go-core/logging"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/go-core/version"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/database"
	"github.com/valentin-kaiser/hdns/pkg/web"
)

var (
	backup  = false
	restore string
	migrate string
	plan    string
)

func init() {
	defer interruption.Catch()
	interruption.Write = true

	logging.Anonymous(true)
	apperror.Anonymous(true)
	apperror.ErrorHandler = func(err error, msg string) {
		log.Error().Err(err).Msg(msg)
	}

	logging.SetGlobalAdapter(logging.
		NewZerologAdapter().
		WithConsole().
		WithFileRotation(filepath.Join(flag.Path, "logs", "hdns.log"), 10, 30, 30, true).
		WithStream(200).
		SetLevel(logging.InfoLevel))

	flag.Register("backup", &backup, "Creates a backup of the database and exits")
	flag.Register("restore", &restore, "Restores the database from the specified backup file and exits")
	flag.Register("migrate", &migrate, "Database migration command: up, down, status")
	flag.Register("plan", &plan, "Plans the database migrations in the given direction: up, down")

	config.Init()
	config.OnChange(func(o *config.App, n *config.App) error {
		log.Info().Msg("applying configuration changes")
		if o.Service.LogLevel != n.Service.LogLevel {
			adapter, _ := logging.GetGlobalAdapter[logging.Adapter]()
			adapter.SetLevel(logging.Level(n.Service.LogLevel))
		}
		return nil
	})
}

func main() {
	defer interruption.Catch()

	apperror.WithDetails = flag.Debug
	logging.Debug(flag.Debug)
	logging.SetGlobalAdapter(logging.
		NewZerologAdapter().
		WithConsole().
		WithFileRotation(filepath.Join(flag.Path, "logs", "hdns.log"), 10, 30, 30, true).
		WithStream(200).
		SetLevel(logging.Level(config.Get().Service.LogLevel)))

	if flag.Help {
		flag.PrintHelp()
		return
	}

	if flag.Version {
		fmt.Print(version.String())
		return
	}

	if plan != "" {
		RunPlanCommand()
		return
	}

	if migrate != "" {
		RunMigrationCommand()
		return
	}

	if backup {
		RunBackupCommand()
		return
	}

	if restore != "" {
		RunRestoreCommand()
		return
	}

	log.Info().Msgf("=== HDNS %s ===", version.String())
	if flag.Debug {
		log.Debug().Msg("running in debug mode")
		log.Debug().Msgf("data path: %s", flag.Path)
		log.Debug().Msgf("git tag: %s", version.GitTag)
		log.Debug().Msgf("git commit: %s", version.GitCommit)
		log.Debug().Msgf("git short: %s", version.GitShort)
		log.Debug().Msgf("build date: %s", version.BuildDate)
		log.Debug().Msgf("runtime version: %s %s", version.GoVersion, version.Platform)

		for _, mod := range version.Modules {
			log.Debug().Msgf("module => %s %s %s", mod.Path, mod.Version, mod.Sum)
		}
	}

	database.HDNS().RegisterOnConnectHandler(database.Setup)

	go database.Connect()
	database.HDNS().AwaitConnection()

	go web.Start()

	signal := interruption.OnSignal([]func() error{
		func() error {
			log.Info().Msg("stopped gracefully")
			return nil
		},
	}, os.Interrupt, syscall.SIGTERM)
	defer interruption.WaitForShutdown(signal)
}

func RunMigrationCommand() {
	database.Connect()
	database.HDNS().AwaitConnection()
	defer database.HDNS().Disconnect()

	switch migrate {
	case "up":
		log.Info().Msg("applying pending migrations...")
		err := database.HDNS().Execute(func(db *sql.DB) error {
			// Access migrations through the database package
			// sql-migrate will handle the gorp_migrations table internally
			return database.Migrate(db)
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to apply migrations")
			return
		}
		log.Info().Msg("migrations completed successfully")
	case "down":
		log.Info().Msg("rolling back last migration...")
		err := database.HDNS().Execute(func(db *sql.DB) error {
			return database.MigrateDown(db)
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to rollback migration")
			return
		}
		log.Info().Msg("rollback completed successfully")
	case "status":
		log.Info().Msg("checking migration status...")
		err := database.HDNS().Execute(func(db *sql.DB) error {
			records, err := database.MigrateStatus(db)
			if err != nil {
				return err
			}

			if len(records) == 0 {
				log.Info().Msg("no migrations applied yet")
				return nil
			}

			log.Info().Msgf("applied %d migration(s):", len(records))
			for _, record := range records {
				log.Info().Msgf("  - %s (applied at: %s)", record.Id, record.AppliedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get migration status")
			return
		}
	default:
		log.Error().Msgf("unknown migrate command: %s", migrate)
	}
}

func RunPlanCommand() {
	database.Connect()
	database.HDNS().AwaitConnection()
	defer database.HDNS().Disconnect()
	switch plan {
	case "up":
		err := database.HDNS().Execute(func(db *sql.DB) error {
			plans, err := database.PlanMigration(db, 0)
			if err != nil {
				return err
			}
			if len(plans) == 0 {
				log.Info().Msg("no pending migrations")
				return nil
			}
			log.Info().Msgf("planned %d migration(s) up:", len(plans))
			for _, plan := range plans {
				log.Info().Msgf("  - %s", plan.Id)
			}
			return nil
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to plan migrations up")
			return
		}
	case "down":
		err := database.HDNS().Execute(func(db *sql.DB) error {
			plans, err := database.PlanMigration(db, 1)
			if err != nil {
				return err
			}
			if len(plans) == 0 {
				log.Info().Msg("no migrations to rollback")
				return nil
			}
			log.Info().Msgf("planned %d migration(s) down:", len(plans))
			for _, plan := range plans {
				log.Info().Msgf("  - %s", plan.Id)
			}
			return nil
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to plan migrations down")
			return
		}
	default:
		log.Error().Msgf("unknown plan command: %s", plan)
	}
}

func RunBackupCommand() {
	database.Connect()
	database.HDNS().AwaitConnection()
	defer database.HDNS().Disconnect()

	path := filepath.Join(flag.Path, "backups", version.GitTag, time.Now().Format("20060102-150405")+".sql")
	log.Info().Field("path", path).Msg("creating database backup...")
	err := database.HDNS().Backup(path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create database backup")
		return
	}
	log.Info().Msg("database backup created successfully")
}

func RunRestoreCommand() {
	database.Connect()
	database.HDNS().AwaitConnection()
	defer database.HDNS().Disconnect()

	log.Info().Field("path", restore).Msg("restoring database from backup...")
	err := database.HDNS().Restore(restore)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to restore database from backup")
		return
	}
	log.Info().Msg("database restored successfully")
}
