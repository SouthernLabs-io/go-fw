package test

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"go.uber.org/fx"

	lib "github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
)

func NewTestDatabase(conf lib.Config, lf *lib.LoggerFactory) lib.Database {
	if conf.Env.Type != lib.EnvTypeTest {
		panic(errors.Newf(errors.ErrCodeBadState, "not in a test: %+v", conf.Env))
	}

	dbName := CreateTestDBName(conf)
	postgresDB := lib.MustOpenGORM(conf, "postgres", lf)
	lf.GetLogger().Infof("Resetting DB: %s", dbName)
	if err := postgresDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, dbName)).Error; err != nil {
		panic(errors.NewUnknownf("failed to drop db: %s, error: %w", dbName, err))
	}
	if err := postgresDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)).Error; err != nil {
		panic(errors.NewUnknownf("failed to create db: %s, error: %w", dbName, err))
	}
	db := lib.MustOpenGORM(conf, dbName, lf)
	return lib.Database{
		DB:     db,
		DbName: dbName,
	}
}

var dbNameReplacer = strings.NewReplacer(
	" ", "_",
	"-", "_",
	"test", "",
)

func CreateTestDBName(conf lib.Config) string {
	// Postgres max length for db name is 63
	const maxLen = 63
	s := dbNameReplacer.Replace(strings.ToLower(conf.Name))
	parts := strings.Split(s, "_")
	// Account for the extra "_" between parts
	maxCutLen := (maxLen - len(parts)) / len(parts)
	s = ""
	for _, part := range parts {
		if len(part) > maxCutLen {
			s += part[0:maxCutLen]
		} else {
			s += part
		}
		s += "_"
	}

	return fmt.Sprintf(
		"%s%.*x_%s",
		s,
		// Plus one to account for the final "_"
		// Each byte uses 2 characters, so we need to divide by 2
		(maxLen-(len(s)+len(conf.Env.Name)+1))/2,
		sha256.Sum256([]byte(conf.Name)),
		conf.Env.Name,
	)
}

func OnTestDBStop(conf lib.Config, db lib.Database, lf *lib.LoggerFactory) error {
	err := lib.OnDBStop(db)
	if err != nil {
		return err
	}

	dbName := db.DbName
	postgresDB := lib.MustOpenGORM(conf, "postgres", lf)
	lf.GetLogger().Infof("Dropping DB: %s", dbName)
	if err := postgresDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, dbName)).Error; err != nil {
		panic(errors.NewUnknownf("failed to drop db: %s, error: %w", dbName, err))
	}
	return nil
}

var TestModuleDB = fx.Provide(
	fx.Annotate(
		NewTestDatabase,
		fx.OnStop(OnTestDBStop),
	),
)
