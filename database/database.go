package database

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/fx"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
	gormtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorm.io/gorm.v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
)

var (
	DBCtxKey   = core.CtxKey("_fw_db")
	DBTxCtxKey = core.CtxKey("_fw_db_tx")
)

const (
	ErrCodeCommitFailed   = "DB_COMMIT_FAILED"
	ErrCodeRollbackFailed = "DB_ROLLBACK_FAILED"
)

type DB struct {
	*gorm.DB
	DbName string
}

func CreateDBName(conf core.Config) string {
	return strings.ReplaceAll(
		strings.ToLower(fmt.Sprintf("%s_%s", conf.Name, conf.Env.Name)),
		"-",
		"_",
	)
}

// NewDB creates a new database instance
func NewDB(conf core.Config, lf *core.LoggerFactory) DB {
	if conf.Env.Type == core.EnvTypeTest {
		panic(errors.Newf(errors.ErrCodeBadState, "in a test: %+v", conf.Env))
	}

	dbName := CreateDBName(conf)
	db := MustOpenGORM(conf, dbName, lf)
	return DB{
		DB:     db,
		DbName: dbName,
	}
}

func (d DB) SetCtx(ctx context.Context) context.Context {
	if d.DB == nil {
		return ctx
	}

	return core.CtxSetValue(ctx, DBCtxKey, d.WithContext(ctx))
}

func GetDBFromCtx(ctx context.Context) *gorm.DB {
	// DB context is set in the middleware, and can also be set manually in tests, worker contexts, etc...
	if db, is := ctx.Value(DBCtxKey).(*gorm.DB); is {
		// Update DB context, it could have change if a no deadline context was passed
		db = db.WithContext(ctx)
		return db
	}
	return nil
}

func (d DB) HealthCheck() error {
	return d.Exec("SELECT 1").Error
}

type DBTx struct {
	*gorm.DB
	closed    bool
	automatic bool
	parentTx  *DBTx
	savePoint string
}

func (t *DBTx) IsAutomatic() bool {
	return t.automatic
}

func (t *DBTx) IsClosed() bool {
	return t.closed
}

func (t *DBTx) IsSub() bool {
	return t.parentTx != nil
}

func (t *DBTx) DeferredCommitOrRollback(err *error) {
	if r := recover(); r != nil {
		*err = errors.Newf(errors.ErrCodePanic, "panic in transaction: %v", r)
		// FIXME: a panic produces a deadlock in a transaction when trying to rollback/commit: https://github.com/lib/pq/issues/178
		// log it here and kill the app, because it will leak memory otherwise
		var logger core.Logger
		if t.Statement != nil && t.Statement.Context != nil {
			logger = core.GetLoggerFromCtx(t.Statement.Context)
		} else {
			logger = core.GetLogger()
		}
		logger.Errorf(
			`A panic produces a deadlock in a transaction when trying to rollback/commit: https://github.com/lib/pq/issues/178
Killing the process, because it will leak memory otherwise
%s`,
			*err,
		)
		os.Exit(1)
	}
	if *err != nil {
		if rollbackErr := t.Rollback().Error; rollbackErr != nil {
			var logger core.Logger
			if t.Statement != nil && t.Statement.Context != nil {
				logger = core.GetLoggerFromCtx(t.Statement.Context)
			} else {
				logger = core.GetLoggerFromCtx(context.Background())
			}
			logger.ErrorE(errors.Newf(ErrCodeRollbackFailed, "failed to rollback on error: %w,\n rollback error: %w", *err, rollbackErr))
		}
	} else {
		if commitErr := t.Commit().Error; commitErr != nil {
			*err = errors.Newf(ErrCodeCommitFailed, "failed to commit trx: %w", commitErr)
		}
	}
}
func (t *DBTx) Commit() *gorm.DB {
	defer func() {
		t.closed = true
		// Avoid future use of this subTx
		t.DB = &gorm.DB{Error: sql.ErrTxDone}
	}()
	if t.parentTx != nil {
		core.GetLoggerFromCtx(t.DB.Statement.Context).Debugf("SubTx: release savepoint: %s", t.savePoint)
		return t.DB.Exec("RELEASE SAVEPOINT " + t.savePoint)
	}
	return t.DB.Commit()
}

func (t *DBTx) Rollback() *gorm.DB {
	defer func() {
		t.closed = true
		// Avoid future use of this subTx
		t.DB = &gorm.DB{Error: sql.ErrTxDone}
	}()
	if t.parentTx != nil {
		core.GetLoggerFromCtx(t.DB.Statement.Context).Infof("SubTx: rollback to savepoint: %s", t.savePoint)
		return t.parentTx.RollbackTo(t.savePoint)
	}
	return t.DB.Rollback()
}

func GetDBTxFromCtx(ctx context.Context) *DBTx {
	if tx, is := ctx.Value(DBTxCtxKey).(*DBTx); is {
		if !tx.closed {
			// Update tx context, it could have change if a no deadline context was passed
			tx.DB = tx.DB.WithContext(ctx)
		}
		return tx
	}
	return nil
}

func InTx(ctx context.Context) *DBTx {
	// check if there is one already
	tx := GetDBTxFromCtx(ctx)
	if tx != nil && !tx.closed {
		core.GetLoggerFromCtx(ctx).Debugf("tx found in ctx, returning it!")
		return tx
	}

	db := GetDBFromCtx(ctx)
	if db == nil {
		panic(errors.Newf(errors.ErrCodeBadState, "no db in context!"))
	}
	// gorm does automatic transaction handling per query
	return &DBTx{DB: db, automatic: true}
}

func WithTx(ctx context.Context, txOptions ...*sql.TxOptions) (*DBTx, context.Context) {
	// check if there is one already
	tx := GetDBTxFromCtx(ctx)
	if tx != nil && !tx.closed {
		savePoint := fmt.Sprintf("sub_%d_%d", time.Now().UnixNano(), rand.Uint32())
		core.GetLoggerFromCtx(ctx).Debugf("tx found in ctx, creating a sub tx with savepoint: %s", savePoint)
		err := tx.DB.Exec("SAVEPOINT " + savePoint).Error
		if err != nil {
			panic(errors.NewUnknownf("failed to create a save point in the db transaction: %w", err))
		}
		return &DBTx{
			DB:        tx.DB,
			closed:    false,
			automatic: false,
			parentTx:  tx,
			savePoint: savePoint,
		}, ctx
	}

	db := GetDBFromCtx(ctx)
	if db == nil {
		panic(errors.Newf(errors.ErrCodeBadState, "no db in context!"))
	}

	tx = &DBTx{DB: db.Begin(txOptions...)}
	ctx = core.CtxSetValue(ctx, DBTxCtxKey, tx)

	return tx, ctx
}

func MustOpenGORM(conf core.Config, dbName string, lf *core.LoggerFactory) *gorm.DB {
	dbConf := conf.Database
	dsn := fmt.Sprintf("host='%s' user='%s' password='%s' dbname='%s' port=%d",
		dbConf.Host,
		dbConf.User,
		dbConf.Pass,
		dbName,
		dbConf.Port)
	gormConf := gorm.Config{
		Logger: core.NewGormLogger(lf.GetLoggerForType(gorm.DB{})),
		NowFunc: func() time.Time {
			// Return time with microsecond precision. Postgres timestamp type has microsecond precision.
			return time.UnixMicro(time.Now().UnixMicro())
		},
	}

	var db *gorm.DB
	var err error

	if conf.Datadog.Tracing {
		sqltrace.Register("pgx", &stdlib.Driver{})
		db, err = gormtrace.Open(postgres.Open(dsn), &gormConf)
	} else {
		db, err = gorm.Open(postgres.Open(dsn), &gormConf)
	}
	if err != nil {
		dsn = strings.ReplaceAll(dsn, "'"+dbConf.Pass+"'", "*")
		panic(errors.NewUnknownf("could not connect to DB: %s, error: %w", dsn, err))
	}
	lf.GetLogger().Infof("DB connection established: \"%s\"", dbName)
	return db
}

func OnDBStop(db DB) error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

var Module = fx.Provide(fx.Annotate(NewDB, fx.OnStop(OnDBStop)))
