package middleware

import (
	"net/http"

	"gorm.io/gorm"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

type DatabaseTrxMiddleware struct {
	BaseMiddleware
	db database.DB
}

func NewDatabaseTrx(
	conf config.Config,
	lf log.LoggerFactory,
	db database.DB,
) *DatabaseTrxMiddleware {
	return &DatabaseTrxMiddleware{
		BaseMiddleware{conf, lf.GetLoggerForType(DatabaseTrxMiddleware{})},
		db,
	}
}

var _ Middleware = (*DatabaseTrxMiddleware)(nil)

func (m *DatabaseTrxMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityBeforeMux
}

func (m *DatabaseTrxMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := log.GetLoggerFromCtx(ctx)

		ctx = m.db.AddToCtx(ctx)
		logger.Debugf("DB handle set on context")

		defer func() {
			if panicErr := recover(); panicErr != nil {
				tx := database.GetDBTxFromCtx(ctx)
				if tx != nil && !tx.IsAutomatic() && !tx.IsClosed() {
					// Update logger to latest in context
					logger = log.GetLoggerFromCtx(ctx)
					logger.Warnf("Rolling back transaction due to panic: %s", panicErr)
					err := tx.Rollback().Error
					if err != nil && !errors.Is(err, gorm.ErrInvalidTransaction) {
						logger.Errorf("Roll back failed with error: %s", err)
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

		tx := database.GetDBTxFromCtx(ctx)
		if tx != nil && !tx.IsAutomatic() && !tx.IsClosed() {
			// Update logger to latest in context
			logger = log.GetLoggerFromCtx(ctx)
			logger.Errorf("Dangling transaction found in ctx. Make sure to commit or rollback manually started transactions")
			logger.Warnf("Rolling back transaction")
			err := tx.Rollback().Error
			if err != nil && !errors.Is(err, gorm.ErrInvalidTransaction) {
				logger.Errorf("Roll back failed with error: %s", err)
			}
		}
	})
}

var FxExportDB = ProvideAsMiddleware(NewDatabaseTrx)
