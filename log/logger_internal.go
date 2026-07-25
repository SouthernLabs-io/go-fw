package log

import (
	"io"
	"log/slog"
	"sync"
)

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newSyncWriter(w io.Writer) io.Writer {
	if sw, ok := w.(*syncWriter); ok {
		return sw
	}
	return &syncWriter{w: w}
}

func (s *syncWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

func (s *syncWriter) Unwrap() io.Writer {
	return s.w
}

type _HandlerOptionsAdapter struct {
	IsSlogJSON bool
	IsConsole  bool
	AddSource  bool
	Leveler    *slog.LevelVar
}

func (h _HandlerOptionsAdapter) Level() slog.Level {
	return h.Leveler.Level()
}

func (h _HandlerOptionsAdapter) SetLevel(level slog.Level) {
	h.Leveler.Set(level)
}

const badKey = "!BADKEY"

func argsToAttrSlice(args []any) []slog.Attr {
	var (
		attr  slog.Attr
		attrs []slog.Attr
	)
	for len(args) > 0 {
		attr, args = argsToAttr(args)
		attrs = append(attrs, attr)
	}
	return attrs
}

// argsToAttr turns a prefix of the nonempty args slice into an Attr
// and returns the unconsumed portion of the slice.
// If args[0] is an Attr, it returns it.
// If args[0] is a string, it treats the first two elements as
// a key-value pair.
// Otherwise, it treats args[0] as a value with a missing key.
func argsToAttr(args []any) (slog.Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return slog.String(badKey, x), nil
		}
		return slog.Any(x, args[1]), args[2:]

	case slog.Attr:
		return x, args[1:]

	default:
		return slog.Any(badKey, x), args[1:]
	}
}
