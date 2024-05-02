package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/phsym/console-slog"

	"github.com/southernlabs-io/go-fw/version"
)

var (
	skipAttr       = slog.Attr{}
	traceAsSlogStr = slog.Level(LogLevelTrace).String()
)

func init() {
	// Set a default logger for slog/log
	logger := GetRootLogger()
	slog.SetDefault(NewSlogLogger(logger))
	logLogger := slog.NewLogLogger(logger.h, slog.Level(logger.Level()))
	log.Default().SetFlags(0)
	log.Default().SetOutput(logLogger.Writer())
}

// Logger structure
type Logger struct {
	ctx    context.Context
	name   string
	h      slog.Handler
	hOpts  _HandlerOptionsAdapter
	writer io.Writer

	SkipCallers int
}

// NewLogger creates a new logger with the given core configuration.
func NewLogger(conf CoreConfig, name string) Logger {
	return NewLoggerWithWriter(conf, name, os.Stdout)
}

// NewLoggerWithWriter creates a new logger with the given core configuration and writer.
func NewLoggerWithWriter(conf CoreConfig, name string, writer io.Writer) Logger {
	logger := Logger{
		ctx:    context.Background(),
		name:   name,
		writer: writer,
		hOpts: _HandlerOptionsAdapter{
			Leveler:   &slog.LevelVar{},
			AddSource: true,
		},
	}
	logger.SetLevel(conf.Log.Level)
	if conf.Env.Type == EnvTypeLocal || conf.Env.Type == EnvTypeTest {
		consoleHOpts := console.HandlerOptions{
			Level:      logger.hOpts.Leveler,
			AddSource:  logger.hOpts.AddSource,
			TimeFormat: time.RFC3339Nano,
			Theme:      console.NewBrightTheme(),
		}
		logger.hOpts.IsConsole = true
		logger.h = console.NewHandler(writer, &consoleHOpts)
		// Add logger name for it to be visible in the console logs
		logger = logger.WithAttrs(slog.String("logger.name", name))
	} else {
		// Use JSON format for non-local environments
		slogHOpts := slog.HandlerOptions{
			Level:     logger.hOpts.Leveler,
			AddSource: logger.hOpts.AddSource,
			// Update fields key to match Datadog expectation
			ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
				if attr.Key == slog.TimeKey {
					return slog.Attr{Key: "timestamp", Value: attr.Value}
				}

				if attr.Key == slog.LevelKey && attr.Value.String() == traceAsSlogStr {
					attr.Value = slog.StringValue(LogLevelTrace.String())
					return attr
				}
				// Build logger group from slog.Source
				if attr.Key == slog.SourceKey {
					source := attr.Value.Any().(*slog.Source)

					var attrs []slog.Attr
					attrs = append(attrs, slog.String("name", name))
					if source.Function != "" {
						attrs = append(attrs, slog.String("method_name", source.Function))
					}
					if source.Line != 0 {
						attrs = append(attrs, slog.Int("line", source.Line))
					}
					if source.File != "" {
						attrs = append(attrs, slog.String("file", source.File))
					}
					return slog.Attr{Key: "logger", Value: slog.GroupValue(attrs...)}
				}
				return attr
			},
		}
		logger.hOpts.IsSlogJSON = true
		logger.h = slog.NewJSONHandler(writer, &slogHOpts).WithAttrs([]slog.Attr{
			// It must be in flat mode, so we can add dd fields later
			slog.String("dd.version", version.Full),
		})
	}

	return logger
}

// SetLevel updates the current log level.
func (l Logger) SetLevel(level LogLevel) {
	l.hOpts.SetLevel(slog.Level(level))
}

// Level returns the current log level.
func (l Logger) Level() LogLevel {
	return LogLevel(l.hOpts.Level())
}

// Writer returns the writer used by the logger.
func (l Logger) Writer() io.Writer {
	return l.writer
}

// WithContext returns a new logger with the given context.
func (l Logger) WithContext(ctx context.Context) Logger {
	ll := l
	ll.ctx = ctx
	return ll
}

// WithAttrs returns a new logger with the given attributes.
func (l Logger) WithAttrs(attrs ...slog.Attr) Logger {
	ll := l
	ll.h = ll.h.WithAttrs(attrs)
	return ll
}

// With returns a new logger with the given attributes.
func (l Logger) With(args ...any) Logger {
	return l.WithAttrs(argsToAttrSlice(args)...)
}

// Enabled returns true if the given level is enabled.
func (l Logger) Enabled(level LogLevel) bool {
	return l.h.Enabled(l.ctx, slog.Level(level))
}

func (l Logger) Tracef(format string, args ...any) {
	if !l.Enabled(LogLevelTrace) {
		return
	}
	l.log(time.Now(), LogLevelTrace, fmt.Sprintf(format, args...))
}

func (l Logger) Trace(msg string, args ...any) {
	if !l.Enabled(LogLevelTrace) {
		return
	}
	l.log(time.Now(), LogLevelTrace, msg, args...)
}

func (l Logger) Debugf(format string, args ...any) {
	if !l.Enabled(LogLevelDebug) {
		return
	}
	l.log(time.Now(), LogLevelDebug, fmt.Sprintf(format, args...))
}

func (l Logger) Debug(msg string, args ...any) {
	if !l.Enabled(LogLevelDebug) {
		return
	}
	l.log(time.Now(), LogLevelDebug, msg, args...)
}

func (l Logger) Infof(format string, args ...any) {
	if !l.Enabled(LogLevelInfo) {
		return
	}
	l.log(time.Now(), LogLevelInfo, fmt.Sprintf(format, args...))
}

func (l Logger) Info(msg string, args ...any) {
	if !l.Enabled(LogLevelInfo) {
		return
	}
	l.log(time.Now(), LogLevelInfo, msg, args...)
}

func (l Logger) Warnf(format string, args ...any) {
	if !l.Enabled(LogLevelWarn) {
		return
	}
	l.log(time.Now(), LogLevelWarn, fmt.Sprintf(format, args...))
}

func (l Logger) Warn(msg string, args ...any) {
	if !l.Enabled(LogLevelWarn) {
		return
	}
	l.log(time.Now(), LogLevelWarn, msg, args...)
}

func (l Logger) Errorf(format string, args ...any) {
	if !l.Enabled(LogLevelError) {
		return
	}
	l.log(time.Now(), LogLevelError, fmt.Sprintf(format, args...))
}

func (l Logger) Error(msg string, args ...any) {
	if !l.Enabled(LogLevelError) {
		return
	}
	l.log(time.Now(), LogLevelError, msg, args...)
}

func (l Logger) ErrorE(err error) {
	if !l.Enabled(LogLevelError) {
		return
	}
	l.log(time.Now(), LogLevelError, fmt.Sprintf("%v", err), slog.Any("error", err))
}

func (l Logger) Log(level LogLevel, msg string, args ...any) {
	l.log(time.Now(), level, msg, args...)
}

func (l Logger) LogAttrs(level LogLevel, msg string, attrs ...slog.Attr) {
	l.logAttrs(time.Now(), level, 2, msg, attrs...)
}

func (l Logger) log(logTime time.Time, level LogLevel, msg string, args ...any) {
	l.logAttrs(logTime, level, 3, msg, argsToAttrSlice(args)...)
}

func (l Logger) logAttrs(logTime time.Time, level LogLevel, skipCallers int, msg string, attrs ...slog.Attr) {
	if !l.Enabled(level) {
		return
	}
	var pc uintptr
	if l.hOpts.AddSource {
		var pcs [1]uintptr
		// Plus one to remove this function from the stack
		runtime.Callers(l.SkipCallers+skipCallers+1, pcs[:])
		pc = pcs[0]
	}
	r := slog.NewRecord(logTime, slog.Level(level), msg, pc)
	r.AddAttrs(attrs...)

	_ = l.h.Handle(l.ctx, r)
}
