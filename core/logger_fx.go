package core

import (
	"log/slog"
	"strings"

	"go.uber.org/fx/fxevent"
)

type _FxLogger struct {
	libLogger Logger
}

func NewFxLogger(logger Logger) fxevent.Logger {
	// As of fx v1.20, the direct caller is "go.uber.org/fx.(*logBuffer)", which is not useful, so we skip it.
	logger.SkipCallers += 2
	return _FxLogger{logger}
}

func (l _FxLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.libLogger.Debug("OnStart hook executing",
			"callee", e.FunctionName,
			"caller", e.CallerName,
		)
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.libLogger.Error("OnStart hook failed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"error", e.Err,
			)
		} else {
			l.libLogger.Debug("OnStart hook executed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"runtime", e.Runtime,
			)
		}
	case *fxevent.OnStopExecuting:
		l.libLogger.Debug("OnStop hook executing",
			"callee", e.FunctionName,
			"caller", e.CallerName,
		)
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.libLogger.Error("OnStop hook failed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"error", e.Err,
			)
		} else {
			l.libLogger.Debug("OnStop hook executed",
				"callee", e.FunctionName,
				"caller", e.CallerName,
				"runtime", e.Runtime,
			)
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			l.libLogger.Error("error encountered while applying options",
				"type", e.TypeName,
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"error", e.Err,
			)
		} else {
			l.libLogger.Debug("supplied",
				"type", e.TypeName,
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
			)
		}
	case *fxevent.Provided:
		for _, rtype := range e.OutputTypeNames {
			l.libLogger.Debug("provided",
				"constructor", e.ConstructorName,
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"type", rtype,
				maybeBool("private", e.Private),
			)
		}
		if e.Err != nil {
			l.libLogger.Error("error encountered while applying options",
				maybeModuleField(e.ModuleName),
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				"error", e.Err,
			)
		}
	case *fxevent.Replaced:
		for _, rtype := range e.OutputTypeNames {
			l.libLogger.Debug("replaced",
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"type", rtype,
			)
		}
		if e.Err != nil {
			l.libLogger.Error("error encountered while replacing",
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"error", e.Err,
			)
		}
	case *fxevent.Decorated:
		for _, rtype := range e.OutputTypeNames {
			l.libLogger.Debug("decorated",
				"decorator", e.DecoratorName,
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"type", rtype,
			)
		}
		if e.Err != nil {
			l.libLogger.Error("error encountered while applying options",
				"stacktrace", e.StackTrace,
				"moduletrace", e.ModuleTrace,
				maybeModuleField(e.ModuleName),
				"error", e.Err,
			)
		}
	case *fxevent.Run:
		if e.Err != nil {
			l.libLogger.Error("error returned",
				"name", e.Name,
				"kind", e.Kind,
				maybeModuleField(e.ModuleName),
				"error", e.Err,
			)
		} else {
			l.libLogger.Debug("run",
				"name", e.Name,
				"kind", e.Kind,
				maybeModuleField(e.ModuleName),
			)
		}
	case *fxevent.Invoking:
		// Do not log stack as it will make logs hard to read.
		l.libLogger.Debug("invoking",
			"function", e.FunctionName,
			maybeModuleField(e.ModuleName),
		)
	case *fxevent.Invoked:
		if e.Err != nil {
			l.libLogger.Error("invoke failed",
				"error", e.Err,
				"stack", e.Trace,
				"function", e.FunctionName,
				maybeModuleField(e.ModuleName),
			)
		}
	case *fxevent.Stopping:
		l.libLogger.Debug("received signal",
			"signal", strings.ToUpper(e.Signal.String()))
	case *fxevent.Stopped:
		if e.Err != nil {
			l.libLogger.Error("stop failed", "error", e.Err)
		}
	case *fxevent.RollingBack:
		l.libLogger.Error("start failed, rolling back", "error", e.StartErr)
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.libLogger.Error("rollback failed", "error", e.Err)
		}
	case *fxevent.Started:
		if e.Err != nil {
			l.libLogger.Error("start failed", "error", e.Err)
		} else {
			l.libLogger.Debug("started")
		}
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.libLogger.Error("custom logger initialization failed", "error", e.Err)
		} else {
			l.libLogger.Debug("initialized custom fxevent.Logger", "function", e.ConstructorName)
		}
	}
}

func maybeModuleField(name string) slog.Attr {
	if len(name) == 0 {
		return skipAttr
	}
	return slog.String("module", name)
}

func maybeBool(name string, b bool) slog.Attr {
	if b {
		return slog.Bool(name, true)
	}
	return skipAttr
}
