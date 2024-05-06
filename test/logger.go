package test

import (
	"bytes"
	"testing"

	"github.com/southernlabs-io/go-fw/core"
)

func GetTestLogger(tb testing.TB) core.Logger {
	return core.NewLoggerWithWriter(
		core.RootConfig{
			Env: core.EnvConfig{Name: "test", Type: core.EnvTypeTest},
			Log: core.LogConfig{Level: core.LogLevelDebug},
		},
		tb.Name(),
		newTestWriter(tb),
	)
}

func NewLoggerFactory(tb testing.TB, conf core.RootConfig) *core.LoggerFactory {
	return core.NewLoggerFactoryWithWriter(conf, newTestWriter(tb))
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
