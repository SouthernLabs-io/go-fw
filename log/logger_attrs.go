package log

import (
	"context"
	"log/slog"

	context2 "github.com/southernlabs-io/go-fw/context"
)

var loggerAttrsCtxKey = context2.CtxKey("_fw_logger_attrs")

// GetLoggerAttrsFromCtx returns the attributes from the context, or nil if there are none.
func GetLoggerAttrsFromCtx(ctx ValueContext) []slog.Attr {
	if attrs, present := ctx.Value(loggerAttrsCtxKey).([]slog.Attr); present {
		return attrs
	}
	return nil
}

// CtxWithLoggerAttrs sets the given attributes to the context, it will overwrite any existing attributes.
func CtxWithLoggerAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	return context2.CtxSetValue(ctx, loggerAttrsCtxKey, attrs)
}

// CtxAppendLoggerAttrs adds the given attributes to the context, it will append to any existing attributes.
func CtxAppendLoggerAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if oldAttrs, present := ctx.Value(loggerAttrsCtxKey).([]slog.Attr); present {
		attrs = append(oldAttrs, attrs...)
	}
	return context2.CtxSetValue(ctx, loggerAttrsCtxKey, attrs)
}
