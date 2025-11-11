package test

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	database "github.com/southernlabs-io/go-fw/database/bun"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

func NewTestDBBun(conf config.Config, lf log.LoggerFactory) *bun.DB {
	if conf.Env.Type != config.EnvTypeTest {
		panic(errors.Newf(errors.ErrCodeBadState, "not in a test: %+v", conf.Env))
	}

	dbName := CreateTestDBName(conf)
	postgresDB, err := database.OpenSqlDB(conf, "postgres", lf)
	if err != nil {
		panic(errors.NewUnknownf("failed to open postgres db: %w", err))
	}
	lf.GetLogger().Infof("Resetting DB: %s", dbName)
	if _, err := postgresDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, dbName)); err != nil {
		panic(errors.NewUnknownf("failed to drop db: %s, error: %w", dbName, err))
	}
	if _, err := postgresDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)); err != nil {
		panic(errors.NewUnknownf("failed to create db: %s, error: %w", dbName, err))
	}

	if err = postgresDB.Close(); err != nil {
		panic(errors.NewUnknownf("failed to close postgres db: %w", err))
	}

	testDB, err := database.OpenSqlDB(conf, dbName, lf)
	if err != nil {
		panic(errors.NewUnknownf("failed to open test db: %w", err))
	}

	db := database.WrapWithBun(conf, testDB)
	return db
}

func OnTestDBBunStop(ctx context.Context, conf config.Config, db *bun.DB, lf log.LoggerFactory) error {

	// Get the current database name
	var dbName string
	err := db.NewRaw("SELECT current_database()").Scan(ctx, &dbName)
	if err != nil {
		return errors.NewUnknownf("failed to get current database name: %w", err)
	}

	err = database.OnDBStop(ctx, db)
	if err != nil {
		return errors.NewUnknownf("failed to stop db: %w", err)
	}

	// Drop the test database
	postgresDB, err := database.OpenSqlDB(conf, "postgres", lf)
	if err != nil {
		return errors.NewUnknownf("failed to open postgres db: %w", err)
	}
	lf.GetLogger().Infof("Dropping DB: %s", dbName)
	if _, err = postgresDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, dbName)); err != nil {
		return errors.NewUnknownf("failed to drop db: %s, error: %w", dbName, err)
	}

	return nil
}

var TestFxExportDBBun = fx.Provide(
	fx.Annotate(
		NewTestDBBun,
		fx.OnStop(OnTestDBBunStop),
	),
)
