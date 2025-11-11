package databasebun

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

// NewDB creates a new database instance
func NewDB(conf config.Config, lf log.LoggerFactory) (*bun.DB, error) {
	if conf.Env.Type == config.EnvTypeTest {
		panic(errors.Newf(errors.ErrCodeBadState, "in a test: %+v", conf.Env))
	}

	dbName := database.CreateDBName(conf)
	sqlDB, err := OpenSqlDB(conf, dbName, lf)
	if err != nil {
		return nil, err
	}

	bunDB := bun.NewDB(sqlDB, pgdialect.New())
	bunDB.SetMaxOpenConns(conf.Database.MaxOpenConns)
	bunDB.SetMaxIdleConns(conf.Database.MaxIdleConns)
	bunDB.SetConnMaxLifetime(conf.Database.ConnMaxIdle)

	bunDB.AddQueryHook(&BunLogger{})

	return bunDB, nil
}

func OpenSqlDB(conf config.Config, dbName string, lf log.LoggerFactory) (*sql.DB, error) {
	dbConf := conf.Database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		dbConf.User,
		dbConf.Pass,
		dbConf.Host,
		dbConf.Port,
		dbName,
	)
	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	// Check the connection
	err := sqlDB.Ping()

	/*if conf.Datadog.Tracing {
		sqltrace.Register("pgx", &stdlib.Driver{})
		db, err = gormtrace.Open(postgres.Open(dsn), &gormConf)
	} else {
		db, err = gorm.Open(postgres.Open(dsn), &gormConf)
	}*/
	if err != nil {
		dsn = strings.ReplaceAll(dsn, dbConf.Pass, "*")
		return nil, errors.NewUnknownf("could not connect to DB: %s, error: %w", dsn, err)
	}
	lf.GetLogger().Infof("DB connection established: \"%s\"", dbName)
	return sqlDB, nil
}

func OnDBStop(ctx context.Context, db *bun.DB) error {
	log.GetLoggerFromCtx(ctx).Info("Closing DB")
	err := db.Close()
	if err != nil {
		return errors.NewUnknownf("failed to close DB: %w", err)
	}
	return nil
}

var FxExport = di.FxProvideAs[bun.IDB](NewDB, []fx.Annotation{fx.OnStop(OnDBStop)}, nil)
