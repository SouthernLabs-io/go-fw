package test

import (
	"bytes"
	"testing"

	lib "github.com/southernlabs-io/go-fw/core"
)

func GetTestLogger(tb testing.TB) lib.Logger {
	return lib.NewLoggerWithWriter(
		lib.CoreConfig{
			Env: lib.EnvConfig{Name: "test", Type: lib.EnvTypeTest},
			Log: lib.LogConfig{Level: lib.LogLevelDebug},
		},
		tb.Name(),
		newTestWriter(tb),
	)
}

func NewLoggerFactory(tb testing.TB, conf lib.CoreConfig) *lib.LoggerFactory {
	return lib.NewLoggerFactoryWithWriter(conf, newTestWriter(tb))
}

// _TestingWriter writes to the testing.TB.Log function.
type _TestingWriter struct {
	tb testing.TB
}

func newTestWriter(tb testing.TB) _TestingWriter {
	return _TestingWriter{
		tb: tb,
	}
}

func (w _TestingWriter) Write(p []byte) (n int, err error) {
	p = bytes.TrimSuffix(p, []byte("\n"))
	w.tb.Log(string(p))
	return len(p), nil
}
