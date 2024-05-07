package slack

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/di"
)

type FxLifecycleLoggerInterceptor struct {
	conf        core.Config
	fxLogger    fxevent.Logger
	logger      core.Logger
	slackClient *Client
}

func NewSlackFxLifecycleLoggerInterceptor(deps struct {
	fx.In

	Conf        core.Config
	LF          *core.LoggerFactory
	SlackClient *Client `optional:"true"`
}) *FxLifecycleLoggerInterceptor {
	return &FxLifecycleLoggerInterceptor{
		conf:        deps.Conf,
		logger:      deps.LF.GetLoggerForType(FxLifecycleLoggerInterceptor{}),
		fxLogger:    di.NewFxLogger(deps.LF.GetLoggerForType(fx.App{})),
		slackClient: deps.SlackClient,
	}
}

func (l FxLifecycleLoggerInterceptor) LogEvent(event fxevent.Event) {
	// Always send events to fx logger
	l.fxLogger.LogEvent(event)

	if l.slackClient == nil {
		return
	}

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
