package middleware

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest"
)

type DatabaseTrxMiddleware struct {
	BaseMiddleware
	db database.DB
}

func NewDatabaseTrx(
	conf config.Config,
	lf *log.LoggerFactory,
	db database.DB,
) *DatabaseTrxMiddleware {
	return &DatabaseTrxMiddleware{
		BaseMiddleware{conf, lf.GetLoggerForType(DatabaseTrxMiddleware{})},
		db,
	}
}

func (m *DatabaseTrxMiddleware) Setup(httpHandler rest.HTTPHandler) {
	httpHandler.Root.Use(m.Run)
}

func (m *DatabaseTrxMiddleware) Run(ctx *gin.Context) {
	logger := log.GetLoggerFromCtx(ctx)
	// We ignore the returned context because is the same as the one passed in
	m.db.SetCtx(ctx)
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

	ctx.Next()

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
}

func (m *DatabaseTrxMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityAuthN - 1
}

var DatabaseTrxModule = ProvideAsMiddleware(NewDatabaseTrx)
