package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"time"

	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/database"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/logging"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/go-core/version"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/database/gen/hdnsdb"

	_ "embed"
)

var (
	instance *database.Database[hdnsdb.Queries]
)

func init() {
	instance = database.New[hdnsdb.Queries]("hdns").RegisterQueries(hdnsdb.New)
}

func HDNS() *database.Database[hdnsdb.Queries] {
	return instance
}

func Setup(db *sql.DB, config database.Config) error {
	q := hdnsdb.New(db)
	release, err := q.GetLatestRelease(context.Background())
	if err == nil && release.GitTag == version.GitTag {
		log.Info().Msg("suite database schema is up-to-date")
		return nil
	}

	pending, err := PlanMigration(db, 0)
	if err != nil {
		return apperror.Wrap(err)
	}

	if len(pending) == 0 {
		log.Info().Msg("no pending migrations for suite database")
		return nil
	}

	log.Info().Msgf("applying %d pending migration(s) to suite database...", len(pending))
	path := filepath.Join(flag.Path, "backups", version.GitTag, time.Now().Format("20060102_150405")+".sql")
	err = instance.Backup(path)
	if err != nil {
		return apperror.NewError("failed to backup suite database before applying schema").AddError(err)
	}

	err = Migrate(db)
	if err != nil {
		return apperror.Wrap(err)
	}

	_, err = hdnsdb.New(db).CreateRelease(context.Background(), hdnsdb.CreateReleaseParams{
		GitTag:    version.GitTag,
		GitCommit: version.GitCommit,
		GitShort:  sql.NullString{String: version.GitShort, Valid: version.GitShort != ""},
		BuildDate: sql.NullString{String: version.BuildDate, Valid: version.BuildDate != ""},
		GoVersion: sql.NullString{String: version.GoVersion, Valid: version.GoVersion != ""},
		Platform:  sql.NullString{String: version.Platform, Valid: version.Platform != ""},
	})
	if err != nil {
		return apperror.NewError("failed to create release entry in suite database").AddError(err)
	}

	return nil
}

func Connect() {
	sl := database.NewLoggingMiddleware(logging.GetPackageLogger("database.hdns"))
	instance.RegisterMiddleware(sl)
	sl.SetEnabled(flag.Debug && config.Get().Service.LogLevel <= int(logging.VerboseLevel)).SetTrace(config.Get().Service.LogLevel < int(logging.VerboseLevel))
	instance.Connect(time.Second, database.Config{
		Driver:   config.Get().Database.Driver,
		Host:     config.Get().Database.Host,
		Port:     config.Get().Database.Port,
		User:     config.Get().Database.Username,
		Password: config.Get().Database.Password,
		Name:     config.Get().Database.Name,
	})
}

func Reconnect() {
	instance.Reconnect(database.Config{
		Driver:   config.Get().Database.Driver,
		Host:     config.Get().Database.Host,
		Port:     config.Get().Database.Port,
		User:     config.Get().Database.Username,
		Password: config.Get().Database.Password,
		Name:     config.Get().Database.Name,
	})
}
