package databasebun

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/uptrace/bun"
)

type BunLogger struct{}

var _ bun.QueryHook = (*BunLogger)(nil)

func (l *BunLogger) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

func (l *BunLogger) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	logger := log.GetLoggerFromCtx(ctx)

	now := time.Now()
	dur := now.Sub(event.StartTime)
	logger.Debug("Bun query executed",
		slog.String("query", event.Query),
		slog.Duration("duration", dur),
	)

	if !logger.Enabled(config.LogLevelDebug) {
		switch event.Err {
		case sql.ErrNoRows, sql.ErrTxDone:
			return
		}
	}

	attrs := []slog.Attr{
		slog.String("query", event.Query),
		slog.Duration("duration", dur),
		slog.String("operation", event.Operation()),
	}

	if event.Err != nil {
		attrs = append(attrs, slog.Any("error", event.Err))
		logger.Error("Bun query error", slog.GroupAttrs("sql", attrs...))
	} else {
		affected, err := event.Result.RowsAffected()
		if err == nil {
			attrs = append(attrs, slog.Int64("affected", affected))
		}
		logger.Debug("Bun query executed", slog.GroupAttrs("sql", attrs...))
	}
}
