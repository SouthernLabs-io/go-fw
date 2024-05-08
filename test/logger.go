package test

import (
	"bytes"
	"testing"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/log"
)

func GetTestLogger(tb testing.TB) log.Logger {
	return log.NewLoggerWithWriter(
		config.RootConfig{
			Env: config.EnvConfig{Name: "test", Type: config.EnvTypeTest},
			Log: config.LogConfig{Level: config.LogLevelDebug},
		},
		tb.Name(),
		newTestWriter(tb),
	)
}

func NewLoggerFactory(tb testing.TB, conf config.RootConfig) *log.LoggerFactory {
	return log.NewLoggerFactoryWithWriter(conf, newTestWriter(tb))
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
