package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
)

type _GormLogger struct {
	logger core.Logger
	silent bool

	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
}

func NewGormLogger(logger core.Logger) gormlogger.Interface {
	// Just skip this wrapper
	logger.SkipCallers += 1
	return _GormLogger{
		logger: logger,

		ignoreRecordNotFoundError: true,
		slowThreshold:             200 * time.Millisecond, // Default from gorm/logger
	}
}

func (gl _GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newGL := gl
	var logLevel core.LogLevel
	switch level {
	// gorm Info level is used to be verbose and should be treated as trace
	case gormlogger.Info:
		logLevel = core.LogLevelTrace
	case gormlogger.Warn:
		logLevel = core.LogLevelWarn
	case gormlogger.Error:
		logLevel = core.LogLevelError
	case gormlogger.Silent:
		logLevel = core.LogLevelError
		newGL.silent = true
	default:
		panic(errors.Newf(errors.ErrCodeBadArgument, "unknown gorm log level: %v", level))
	}
	newGL.logger.SetLevel(logLevel)
	return newGL
}

func (gl _GormLogger) configureLogger(ctx context.Context) core.Logger {
	attrs := core.GetLoggerAttrsFromCtx(ctx)
	if len(attrs) > 0 {
		return gl.logger.WithAttrs(attrs...)
	}
	return gl.logger
}

func (gl _GormLogger) Info(ctx context.Context, format string, args ...interface{}) {
	gl.configureLogger(ctx).Infof(format, args...)
}

func (gl _GormLogger) Warn(ctx context.Context, format string, args ...interface{}) {
	gl.configureLogger(ctx).Warnf(format, args...)
}

func (gl _GormLogger) Error(ctx context.Context, format string, args ...interface{}) {
	gl.configureLogger(ctx).Errorf(format, args...)
}

func (gl _GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if gl.silent {
		return // Silent
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && (!errors.Is(err, gorm.ErrRecordNotFound) || !gl.ignoreRecordNotFoundError):
		sql, rows := fc()

		attrs := []any{
			slog.Any("error", err),
			slog.String("sql", sql),
			slog.Duration("duration", elapsed),
			slog.Int64("rows", rows),
		}
		logger := gl.configureLogger(ctx)
		// as of gorm v1.25 the statement issuer is two frames down
		logger.SkipCallers += 2
		logger.Error("Failed to run query", attrs...)

	case elapsed > gl.slowThreshold && gl.slowThreshold != 0 && gl.logger.Enabled(core.LogLevelWarn):
		sql, rows := fc()

		// Append context attributes
		attrs := []any{
			slog.String("sql", sql),
			slog.Duration("duration", elapsed),
			slog.Int64("rows", rows),
			slog.Bool("slow_sql", true),
		}
		logger := gl.configureLogger(ctx)
		// as of gorm v1.25 the statement issuer is two frames down
		logger.SkipCallers += 2
		logger.Warn(fmt.Sprintf("Slow sql executed: %s", elapsed), attrs...)

	case gl.logger.Enabled(core.LogLevelTrace):
		sql, rows := fc()

		attrs := []any{
			slog.String("sql", sql),
			slog.Duration("duration", elapsed),
		}
		if rows != -1 {
			attrs = append(attrs, slog.Int64("rows", rows))
		}
		logger := gl.configureLogger(ctx)
		// as of gorm v1.25 the statement issuer is two frames down
		logger.SkipCallers += 2
		logger.Trace(fmt.Sprintf("SQL executed: %s", elapsed), attrs...)
	}
}
