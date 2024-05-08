package slack

import (
	"go.uber.org/fx/fxevent"

	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/log"
)

type _FxLoggerFactory struct {
	slackClient *Client
	fxLogger    fxevent.Logger
	logger      log.Logger
}

func NewFxLoggerFactory(fxLF di.FxLoggerFactory, slackClient *Client, lf *log.LoggerFactory) di.FxLoggerFactory {
	// If the client is nil, it means it is disabled by configuration
	if slackClient == nil {
		return fxLF
	}
	return &_FxLoggerFactory{
		fxLogger:    fxLF.CreateLogger(),
		slackClient: slackClient,
		logger:      lf.GetLoggerForType(FxLogger{}),
	}
}

func (lf _FxLoggerFactory) CreateLogger() fxevent.Logger {
	return NewFxLogger(lf.slackClient, lf.fxLogger, lf.logger)
}

type FxLogger struct {
	slackClient *Client
	fxLogger    fxevent.Logger
	logger      log.Logger
}

func NewFxLogger(slackClient *Client, fxLogger fxevent.Logger, logger log.Logger) *FxLogger {
	return &FxLogger{
		slackClient: slackClient,
		fxLogger:    fxLogger,
		logger:      logger,
	}
}

func (l *FxLogger) LogEvent(event fxevent.Event) {
	// Always send events to fx logger
	l.fxLogger.LogEvent(event)

	switch ev := event.(type) {
	case *fxevent.Started:

		var err error
		if ev.Err != nil {
			err = l.slackClient.Errorf("Application *failed* to start: ```%s```", ev.Err)
		} else {
			err = l.slackClient.Infof("Application *successfully* started 🟢")
		}
		if err != nil {
			l.logger.Warnf("Failed to send slack message for event: %#v, error: %s", event, err)
		}

	case *fxevent.Stopped:
		// Send message synchronously as we are shutting down
		var err error
		if ev.Err != nil {
			err = l.slackClient.Errorf("Application *failed* to stop cleanly: ```%s```", ev.Err)
		} else {
			err = l.slackClient.Infof("Application *successfully* stopped 🔴")
		}
		if err != nil {
			l.logger.Warnf("Failed to send slack message for event: %#v, error: %s", event, err)
		}
	}
}
