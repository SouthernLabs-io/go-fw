package middleware

import (
	"database/sql"
	"net/http"

	"github.com/uptrace/bun"

	"github.com/southernlabs-io/go-fw/config"
	database "github.com/southernlabs-io/go-fw/database/bun"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

type DatabaseBunTrxMiddleware struct {
	BaseMiddleware
	db *bun.DB
}

func NewDatabaseBunTrxMiddleware(
	conf config.Config,
	lf log.LoggerFactory,
	db *bun.DB,
) *DatabaseBunTrxMiddleware {
	return &DatabaseBunTrxMiddleware{
		BaseMiddleware{Conf: conf, Logger: lf.GetLoggerForType(DatabaseBunTrxMiddleware{})},
		db,
	}
}

var _ Middleware = (*DatabaseBunTrxMiddleware)(nil)

func (m *DatabaseBunTrxMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityBeforeMux
}

func (m *DatabaseBunTrxMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := log.GetLoggerFromCtx(ctx)

		ctx = database.AddToCtx(ctx, m.db)
		logger.Debug("Bun DB set in context")

		defer func() {
			// Handle panics to ensure transaction rollback
			if panicErr := recover(); panicErr != nil {
				ctx = r.Context()
				idb := database.GetDBFromCtx(ctx)
				if tx, ok := idb.(*bun.Tx); ok {
					// Update logger to latest in context
					logger = log.GetLoggerFromCtx(ctx)
					// Rollback the sql transaction so we close the whole transaction including save steps.
					err := tx.Tx.Rollback()
					if err == nil || !errors.Is(err, sql.ErrTxDone) {
						logger.Warnf("Rolling back transaction due to panic: %s", panicErr)
						if err != nil {
							logger.Errorf("Roll back failed with error: %s", err)
						}
					}
				}
				// Continue panic chain
				panic(panicErr)
			}
		}()

		if r.Context() != ctx {
			// Update request context if changed
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)

		// Check for dangling transactions
		ctx = r.Context()
		idb := database.GetDBFromCtx(ctx)
		if tx, ok := idb.(*bun.Tx); ok {
			logger = log.GetLoggerFromCtx(ctx)

			// Rollback the sql transaction so we close the whole transaction including save steps.
			err := tx.Tx.Rollback()
			if err == nil || !errors.Is(err, sql.ErrTxDone) {
				logger.Errorf("Dangling transaction found in ctx. Make sure to commit or rollback manually started transactions. Rolling back now.")
				if err != nil {
					logger.Errorf("Roll back failed with error: %s", err)
				}
			}
		}
	})
}

var FxExportDBBun = ProvideAsMiddleware(NewDatabaseBunTrxMiddleware)
