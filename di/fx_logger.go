package di

import (
	"log/slog"
	"strings"

	"go.uber.org/fx/fxevent"

	"github.com/southernlabs-io/go-fw/log"
)

type _FxLogger struct {
	logger log.Logger
}

func NewFxLogger(logger log.Logger) fxevent.Logger {
	// skip internal not usable callers in fx
	logger.SkipCallers += 2
	return _FxLogger{logger}

}

func (l _FxLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.logger.Debug("OnStart hook executing",
			"callee", e.FunctionName,
			"caller", e.CallerName,
		)
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.logger.Error("OnStart hook failed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"error", e.Err,
			)
		} else {
			l.logger.Debug("OnStart hook executed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"runtime", e.Runtime,
			)
		}
	case *fxevent.OnStopExecuting:
		l.logger.Debug("OnStop hook executing",
			"callee", e.FunctionName,
			"caller", e.CallerName,
		)
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.logger.Error("OnStop hook failed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"error", e.Err,
			)
		} else {
			l.logger.Debug("OnStop hook executed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"runtime", e.Runtime,
			)
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			l.logger.Error("Error encountered while applying options",
				"type", e.TypeName,
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"error", e.Err,
			)
		} else {
			l.logger.Debug("Supplied",
				"type", e.TypeName,
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
			)
		}
	case *fxevent.Provided:
		for _, rtype := range e.OutputTypeNames {
			l.logger.Debug("Provided",
				"constructor", e.ConstructorName,
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"type", rtype,
				maybeBool("private", e.Private),
			)
		}
		if e.Err != nil {
			l.logger.Error("Error encountered while applying options",
				maybeModuleField(e.ModuleName),
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				"error", e.Err,
			)
		}
	case *fxevent.Replaced:
		for _, rtype := range e.OutputTypeNames {
			l.logger.Debug("Replaced",
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"type", rtype,
			)
		}
		if e.Err != nil {
			l.logger.Error("Error encountered while replacing",
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"error", e.Err,
			)
		}
	case *fxevent.Decorated:
		for _, rtype := range e.OutputTypeNames {
			l.logger.Debug("Decorated",
				"decorator", e.DecoratorName,
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"type", rtype,
			)
		}
		if e.Err != nil {
			l.logger.Error("Error encountered while applying options",
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"error", e.Err,
			)
		}
	case *fxevent.Run:
		if e.Err != nil {
			l.logger.Error("Error returned",
				"name", e.Name,
				"kind", e.Kind,
				maybeModuleField(e.ModuleName),
				"error", e.Err,
			)
		} else {
			l.logger.Debug("Run",
				"name", e.Name,
				"kind", e.Kind,
				maybeModuleField(e.ModuleName),
			)
		}
	case *fxevent.Invoking:
		// Do not log stack as it will make logs hard to read.
		l.logger.Debug("Invoking",
			"function", e.FunctionName,
			maybeModuleField(e.ModuleName),
		)
	case *fxevent.Invoked:
		if e.Err != nil {
			l.logger.Error("Invoke failed",
				"error", e.Err,
				"stack", e.Trace,
				"function", e.FunctionName,
				maybeModuleField(e.ModuleName),
			)
		}
	case *fxevent.Stopping:
		l.logger.Debug("Received signal",
			"signal", strings.ToUpper(e.Signal.String()))
	case *fxevent.Stopped:
		if e.Err != nil {
			l.logger.Error("Stop failed", "error", e.Err)
		} else {
			l.logger.Info("Stopped")
		}
	case *fxevent.RollingBack:
		l.logger.Error("Start failed, rolling back", "error", e.StartErr)
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.logger.Error("Rollback failed", "error", e.Err)
		}
	case *fxevent.Started:
		if e.Err != nil {
			l.logger.Error("Start failed", "error", e.Err)
		} else {
			l.logger.Info("Started")
		}
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.logger.Error("custom logger initialization failed", "error", e.Err)
		} else {
			l.logger.Debug("initialized custom fxevent.Logger", "function", e.ConstructorName)
		}
	default:
		l.logger.Warn("Unknown event!, you are using an fx version newer than what it is supported", "event", event)
	}
}

func maybeModuleField(name string) slog.Attr {
	if len(name) == 0 {
		return log.SkipAttr
	}
	return slog.String("module", name)
}

func maybeBool(name string, b bool) slog.Attr {
	if b {
		return slog.Bool(name, true)
	}
	return log.SkipAttr
}
