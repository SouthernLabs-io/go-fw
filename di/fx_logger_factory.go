package di

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/southernlabs-io/go-fw/log"
)

type FxLoggerFactory interface {
	CreateLogger() fxevent.Logger
}

type _FxLoggerFactory struct {
	logger log.Logger
}

func NewFxLoggerFactory(lf *log.LoggerFactory) FxLoggerFactory {
	return _FxLoggerFactory{logger: lf.GetLoggerForType(fx.App{})}
}

func (lf _FxLoggerFactory) CreateLogger() fxevent.Logger {
	return NewFxLogger(lf.logger)
}
