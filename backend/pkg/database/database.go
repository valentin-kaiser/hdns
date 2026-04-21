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
	"github.com/valentin-kaiser/hdns/pkg/database/schema"

	_ "embed"
)

var (
	hdnsdb *database.Database[schema.Queries]

	// SeedNotificationRulesFunc is set by the notification package to seed
	// default notification rules after creating the default admin user.
	SeedNotificationRulesFunc func(ctx context.Context, q *schema.Queries, adminID []byte)
)

func init() {
	hdnsdb = database.New[schema.Queries](database.DriverMySQL, "hdns").RegisterQueries(schema.New)
}

func HDNS() *database.Database[schema.Queries] {
	return hdnsdb
}

func Setup(db *sql.DB) error {
	pending, err := PlanMigration(db, 0, 0)
	if err != nil {
		return apperror.Wrap(err)
	}

	if len(pending) > 0 {
		log.Info().Msgf("applying %d pending migration(s) to hdns database...", len(pending))
		path := filepath.Join(flag.Path, "backups", version.GitTag, time.Now().Format("20060102_150405")+".sql")
		err = hdnsdb.Backup(path, "")
		if err != nil {
			return apperror.NewError("failed to backup hdns database before applying schema").AddError(err)
		}

		err = Migrate(db, 0)
		if err != nil {
			return apperror.Wrap(err)
		}
	}

	return nil
}

func Connect() {
	sl := database.NewLoggingMiddleware(logging.GetPackageLogger("database.hdns"))
	hdnsdb.RegisterMiddleware(sl)
	sl.SetEnabled(flag.Debug && config.Get().LogLevel <= int(logging.VerboseLevel)).SetTrace(config.Get().LogLevel < int(logging.VerboseLevel))
	hdnsdb.Connect(time.Second, config.Get().Database)
}

func Reconnect() {
	hdnsdb.Reconnect(config.Get().Database)
}
