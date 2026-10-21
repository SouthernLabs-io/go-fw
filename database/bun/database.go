package databasebun

import (
	"context"
	"database/sql"
	"net"
	"net/url"
	"strconv"
	"testing"

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
	if testing.Testing() {
		panic(errors.Newf(errors.ErrCodeBadState, "in a test: %+v", conf.Env))
	}

	dbName := database.CreateDBName(conf)
	sqlDB, err := OpenSqlDB(conf, dbName, lf)
	if err != nil {
		return nil, err
	}

	bunDB := WrapWithBun(conf, sqlDB)
	return bunDB, nil
}

func WrapWithBun(conf config.Config, slqDB *sql.DB) *bun.DB {
	bunDB := bun.NewDB(slqDB, pgdialect.New())

	// Set connection pool settings
	bunDB.SetMaxOpenConns(conf.Database.MaxOpenConns)
	bunDB.SetMaxIdleConns(conf.Database.MaxIdleConns)
	bunDB.SetConnMaxIdleTime(conf.Database.ConnMaxIdleTime)
	bunDB.SetConnMaxLifetime(conf.Database.ConnMaxLifetime)

	// Add query logger
	bunDB.AddQueryHook(&BunLogger{})

	return bunDB
}

func OpenSqlDB(conf config.Config, dbName string, lf log.LoggerFactory) (*sql.DB, error) {
	dbConf := conf.Database
	dsn := postgresDSN(dbConf, dbName)
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
		_ = sqlDB.Close()
		return nil, errors.NewUnknownf("could not connect to DB: %s, error: %w", sanitizedDSN(dsn), err)
	}
	lf.GetLogger().Infof("DB connection established: \"%s\"", dbName)
	return sqlDB, nil
}

func postgresDSN(dbConf config.DatabaseConfig, dbName string) string {
	query := url.Values{}
	sslMode := dbConf.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	query.Set("sslmode", sslMode)
	if dbConf.SSLRootCert != "" {
		query.Set("sslrootcert", dbConf.SSLRootCert)
	}

	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbConf.User, dbConf.Pass),
		Host:     net.JoinHostPort(dbConf.Host, strconv.Itoa(dbConf.Port)),
		Path:     "/" + dbName,
		RawQuery: query.Encode(),
	}
	return dsn.String()
}

func sanitizedDSN(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "postgres://[redacted]"
	}
	if parsed.User != nil {
		parsed.User = url.User(parsed.User.Username())
	}
	return parsed.String()
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
