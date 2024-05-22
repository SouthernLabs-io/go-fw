package log

import (
	"bytes"
	"io"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/sync"
)

var loggerFactoryCtxKey = context.CtxKey("_fw_logger_factory")

// defaultLoggerFactory is the default logger factory used by the package-level functions.
var defaultLoggerFactory = NewLoggerFactory(config.GetRootConfig())

// SetDefaultLoggerFactory sets the default logger factory to use for the package-level functions.
func SetDefaultLoggerFactory(f *LoggerFactory) {
	defaultLoggerFactory = f
}

// GetDefaultLoggerFactory returns the default logger factory used by the package-level functions.
func GetDefaultLoggerFactory() *LoggerFactory {
	return defaultLoggerFactory
}

type ValueContext interface{ Value(any) any }

type LoggerFactory struct {
	loggersByPath *sync.Map[string, Logger]
	coreConfig    config.RootConfig
	writer        io.Writer
}

// NewLoggerFactory creates a new logger factory with the given core configuration.
func NewLoggerFactory(coreConfig config.RootConfig) *LoggerFactory {
	normalized := make(map[string]config.LogLevel, len(coreConfig.Log.Levels))
	for pth, level := range coreConfig.Log.Levels {
		normalized[path.Clean(pth)] = level
	}
	coreConfig.Log.Levels = normalized

	return &LoggerFactory{
		loggersByPath: sync.NewMap[string, Logger](),
		coreConfig:    coreConfig,
	}
}

// NewLoggerFactoryWithWriter creates a new logger factory with the given core configuration and writer.
func NewLoggerFactoryWithWriter(coreConfig config.RootConfig, writer io.Writer) *LoggerFactory {
	factory := NewLoggerFactory(coreConfig)
	factory.writer = writer
	return factory
}

func (lf *LoggerFactory) SetCtx(ctx context.Context) context.Context {
	return context.CtxSetValue(ctx, loggerFactoryCtxKey, lf)
}

// GetRootLogger returns the root logger. This is a shortcut for GetLoggerForPath("/").
func (lf *LoggerFactory) GetRootLogger() Logger {
	return lf.GetLoggerForPath("root")
}

// GetRootLogger returns the root logger using the default logger factory.
func GetRootLogger() Logger {
	return defaultLoggerFactory.GetRootLogger()
}

// GetLoggerForPath returns a logger for the given path and adds the context properties (if any).
func (lf *LoggerFactory) GetLoggerForPath(pth string) Logger {
	return lf.GetLoggerFromCtxForPath(context.Background(), pth)
}

// GetLoggerForPath returns a logger for the given path and adds the context properties (if any) using
// the default logger factory.
func GetLoggerForPath(pth string) Logger {
	return defaultLoggerFactory.GetLoggerForPath(pth)
}

// GetLoggerForType returns a logger for the given type and adds the context properties (if any).
func (lf *LoggerFactory) GetLoggerForType(forType any) Logger {
	return lf.GetLoggerFromCtxForType(context.Background(), forType)
}

// GetLoggerForType returns a logger for the given type and adds the context properties (if any) using
// the default logger factory.
func GetLoggerForType(forType any) Logger {
	return defaultLoggerFactory.GetLoggerForType(forType)
}

var skipPathPrefixes = []string{
	"github.com/southernlabs-io/go-fw/log.(*LoggerFactory).",
	"github.com/southernlabs-io/go-fw/log.Logger.",
	"github.com/southernlabs-io/go-fw/log.GetLogger",
	"github.com/southernlabs-io/go-fw/worker.NewWorkerContext",
}

func findCallerPath() string {
	pth := "_"
	var pcs [6]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for more := true; more; {
		var frame runtime.Frame
		frame, more = frames.Next()
		var skip bool
		for _, prefix := range skipPathPrefixes {
			if strings.HasPrefix(frame.Function, prefix) {
				skip = true
				break
			}
		}
		if !skip && frame.Function != "" {
			pth = frame.Function
			break
		}
	}
	return pth
}

// GetLogger returns a logger for the caller
func (lf *LoggerFactory) GetLogger() Logger {
	pth := findCallerPath()
	return lf.GetLoggerFromCtxForPath(context.Background(), pth)
}

// GetLogger returns a logger for the caller using the default logger factory.
func GetLogger() Logger {
	return defaultLoggerFactory.GetLogger()
}

// GetLoggerFromCtx returns a logger for the caller and adds the context properties (if any)
func (lf *LoggerFactory) GetLoggerFromCtx(ctx ValueContext) Logger {
	pth := findCallerPath()
	return lf.GetLoggerFromCtxForPath(ctx, pth)
}

// GetLoggerFromCtx returns a logger for the caller and adds the context properties (if any). Default logger Factory will
// be used if there is none in the context.
func GetLoggerFromCtx(ctx ValueContext) Logger {
	lf, is := ctx.Value(loggerFactoryCtxKey).(*LoggerFactory)
	if !is {
		lf = defaultLoggerFactory
	}

	return lf.GetLoggerFromCtx(ctx)
}

// GetLoggerFromCtxForType returns a logger for the given type and adds the context properties (if any)
func (lf *LoggerFactory) GetLoggerFromCtxForType(ctx ValueContext, forType any) Logger {
	pth := "_"
	t := reflect.TypeOf(forType)
	// Limit iterations to 10 to avoid infinite loops
	const MaxIterations = 10.
	for i := 0; i < MaxIterations; i++ {
		if t == nil {
			break
		}
		pkg := t.PkgPath()
		if pkg != "" {
			pth = pkg + "." + t.Name()
			break
		}
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
			t = t.Elem()
		default:
			t = nil
		}
	}

	return lf.GetLoggerFromCtxForPath(ctx, pth)
}

// GetLoggerFromCtxForType returns a logger for the given type and adds the context properties (if any) using
func GetLoggerFromCtxForType(ctx ValueContext, forType any) Logger {
	lf, is := ctx.Value(loggerFactoryCtxKey).(*LoggerFactory)
	if !is {
		lf = defaultLoggerFactory
	}
	return lf.GetLoggerFromCtxForType(ctx, forType)

}

// GetLoggerFromCtxForPath returns a logger for the given type and adds the context properties (if any)
func (lf *LoggerFactory) GetLoggerFromCtxForPath(ctx ValueContext, pth string) Logger {
	var logger = lf.loggersByPath.LoadOrStoreFunc(pth, lf.newLogger)

	attrs := GetLoggerAttrsFromCtx(ctx)
	if len(attrs) > 0 {
		logger = logger.WithAttrs(attrs...)
	}
	return logger
}

// GetLoggerFromCtxForPath returns a logger for the given path and adds the context properties (if any) using
// the default logger factory.
func GetLoggerFromCtxForPath(ctx ValueContext, pth string) Logger {
	lf, is := ctx.Value(loggerFactoryCtxKey).(*LoggerFactory)
	if !is {
		lf = defaultLoggerFactory
	}
	return lf.GetLoggerFromCtxForPath(ctx, pth)
}

// newLogger creates a new logger for the given path and sets the level based on the configuration.
func (lf *LoggerFactory) newLogger(pth string) Logger {
	var writer io.Writer
	if lf.writer != nil {
		writer = lf.writer
	} else {
		switch lf.coreConfig.Log.Writer {
		case "", config.LogConfigWriterStdout:
			writer = os.Stdout
		case config.LogConfigWriterStderr:
			writer = os.Stderr
		case config.LogConfigWriterBuffer:
			writer = new(bytes.Buffer)
		default:
			panic(errors.Newf(errors.ErrCodeBadArgument, "unknown log writer: %s", lf.coreConfig.Log.Writer))
		}
	}
	logger := NewLoggerWithWriter(lf.coreConfig, pth, writer)

	// Check if there is a configured level for this path
	for idx := len(pth); idx > 0; idx = strings.LastIndexAny(pth, "/.") {
		pth = pth[:idx]
		if level, present := lf.coreConfig.Log.Levels[pth]; present {
			logger.SetLevel(level)
			break
		}
	}
	return logger
}
