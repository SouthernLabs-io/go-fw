package context

import (
	"context"
	"runtime"
	"time"
)

// CtxKey must be an alias to any and set as string like: ctxKey("lib_logger_factory") for gin.Context.Value to work properly.
type CtxKey any

type noDeadlineContext struct {
	context.Context
}

func (*noDeadlineContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (*noDeadlineContext) Done() <-chan struct{} {
	return nil
}

func (*noDeadlineContext) Err() error {
	return nil
}

func (c *noDeadlineContext) Value(key any) any {
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		funcDef := runtime.FuncForPC(pc)
		if funcDef != nil && funcDef.Name() == "context.Cause" {
			return nil
		}
	}
	return c.Context.Value(key)
}

// NoDeadlineAndNotCancellableContext This creates a context that is not connected to the parent deadline/cancellable behavior.
func NoDeadlineAndNotCancellableContext(parent context.Context) context.Context {
	return &noDeadlineContext{parent}
}

func CtxSetValue(ctx context.Context, key any, value any) context.Context {
	if keyStr, is := key.(string); is {
		if setCtx, is := ctx.(interface{ Set(key string, value any) }); is {
			setCtx.Set(keyStr, value)
			return ctx
		}
	}
	return context.WithValue(ctx, key, value)
}
