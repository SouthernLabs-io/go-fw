package test

import (
	"fmt"
	"testing"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	database "github.com/southernlabs-io/go-fw/database/gorm"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

func NewTestDBGORM(conf config.Config, lf log.LoggerFactory) database.DB {
	if !testing.Testing() {
		panic(errors.Newf(errors.ErrCodeBadState, "not in a test: %+v", conf.Env))
	}

	dbName := CreateTestDBName(conf)
	postgresDB := database.MustOpenGORM(conf, "postgres", lf)
	lf.GetLogger().Infof("Resetting DB: %s", dbName)
	if err := postgresDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, dbName)).Error; err != nil {
		panic(errors.NewUnknownf("failed to drop db: %s, error: %w", dbName, err))
	}
	if err := postgresDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)).Error; err != nil {
		panic(errors.NewUnknownf("failed to create db: %s, error: %w", dbName, err))
	}
	db := database.MustOpenGORM(conf, dbName, lf)
	return database.DB{
		DB:     db,
		DbName: dbName,
	}
}

func OnTestDBGORMStop(conf config.Config, db database.DB, lf log.LoggerFactory) error {
	err := database.OnDBStop(db)
	if err != nil {
		return err
	}

	dbName := db.DbName
	postgresDB := database.MustOpenGORM(conf, "postgres", lf)
	lf.GetLogger().Infof("Dropping DB: %s", dbName)
	if err := postgresDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, dbName)).Error; err != nil {
		panic(errors.NewUnknownf("failed to drop db: %s, error: %w", dbName, err))
	}
	return nil
}

var TestFxExportDBGORM = fx.Provide(
	fx.Annotate(
		NewTestDBGORM,
		fx.OnStop(OnTestDBGORMStop),
	),
)
