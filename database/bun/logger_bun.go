package databasebun

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/log"
)

type BunLogger struct{}

var _ bun.QueryHook = (*BunLogger)(nil)

func (l *BunLogger) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

func (l *BunLogger) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	logger := log.GetLoggerFromCtxForType(ctx, event.DB)

	now := time.Now()
	dur := now.Sub(event.StartTime)
	if !logger.Enabled(config.LogLevelDebug) {
		if errors.Is(event.Err, sql.ErrNoRows) || errors.Is(event.Err, sql.ErrTxDone) || errors.Is(event.Err, context.Canceled) {
			return
		}
	}

	attrs := []slog.Attr{
		slog.String("query", event.Query),
		slog.Duration("duration", dur),
		slog.String("operation", event.Operation()),
	}

	if event.Err != nil {
		if errors.Is(event.Err, sql.ErrNoRows) || errors.Is(event.Err, sql.ErrTxDone) {
			logger.Debug(fmt.Sprintf("Bun query terminated with: %s", event.Err.Error()), slog.GroupAttrs("sql", attrs...))
		} else if errors.Is(event.Err, context.Canceled) {
			logger.Debug(fmt.Sprintf("Bun query canceled: %s", event.Err.Error()), slog.GroupAttrs("sql", attrs...))
		} else if errors.Is(event.Err, context.DeadlineExceeded) {
			logger.Debug(fmt.Sprintf("Bun query deadline exceeded: %s", event.Err.Error()), slog.GroupAttrs("sql", attrs...))
		} else {
			err := event.Err
			var pgErr pgdriver.Error

			if errors.As(err, &pgErr) {
				// Create a new error with pgErr extra fields
				err = fmt.Errorf("%w, with pg.details: %q, pg.where: %q, pg.column: %q, pg.constraint: %q", pgErr, pgErr.Field('D'), pgErr.Field('W'), pgErr.Field('c'), pgErr.Field('n'))
			}
			attrs = append(attrs, slog.Any("error", err))
			logger.Error("Bun query failed", slog.GroupAttrs("sql", attrs...))
		}
	} else {
		if event.Result != nil {
			affected, err := event.Result.RowsAffected()
			if err == nil {
				attrs = append(attrs, slog.Int64("affected", affected))
			}
		}
		logger.Debug("Bun query executed", slog.GroupAttrs("sql", attrs...))
	}
}
