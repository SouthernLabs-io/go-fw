package log

import (
	"log/slog"

	"github.com/southernlabs-io/go-fw/context"
)

var loggerAttrsCtxKey = context.CtxKey("_fw_logger_attrs")

// GetLoggerAttrsFromCtx returns the attributes from the context, or nil if there are none.
func GetLoggerAttrsFromCtx(ctx ValueContext) []slog.Attr {
	if attrs, present := ctx.Value(loggerAttrsCtxKey).([]slog.Attr); present {
		return attrs
	}
	return nil
}

// CtxWithLoggerAttrs sets the given attributes to the context, it will overwrite any existing attributes.
func CtxWithLoggerAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	return context.WithValue(ctx, loggerAttrsCtxKey, attrs)
}

// CtxAppendLoggerAttrs adds the given attributes to the context, it will append to any existing attributes.
// If an attribute with the same key already exists, it is replaced if the value differs, or skipped if identical.
func CtxAppendLoggerAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	oldAttrs, _ := ctx.Value(loggerAttrsCtxKey).([]slog.Attr)
	if oldAttrs == nil {
		return context.WithValue(ctx, loggerAttrsCtxKey, attrs)
	}

	merged := make([]slog.Attr, len(oldAttrs))
	copy(merged, oldAttrs)

	for _, newAttr := range attrs {
		found := false
		for i, existing := range merged {
			if existing.Key == newAttr.Key {
				if !existing.Value.Equal(newAttr.Value) {
					merged[i] = newAttr
				}
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, newAttr)
		}
	}

	return context.WithValue(ctx, loggerAttrsCtxKey, merged)
}
