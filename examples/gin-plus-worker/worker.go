package main

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/worker"
)

type SimpleWorker struct {
	logger log.Logger
	id     string
}

func (w *SimpleWorker) GetConcurrency() worker.ConcurrencyConfig {
	return worker.ConcurrencyConfig{
		Mode: worker.ConcurrencyModeMulti,
	}
}

func (w *SimpleWorker) GetName() string {
	return "SimpleWorker"
}

func (w *SimpleWorker) GetID() string {
	return w.id
}

func (w *SimpleWorker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("context done")
			return context.Cause(ctx)
		default:
			w.logger.Info("doing work")
			time.Sleep(time.Second * 5)
		}
	}
}

var _ worker.LongRunningWorker = (*SimpleWorker)(nil)

func NewSimpleWorker(lf *log.LoggerFactory) *SimpleWorker {
	wf := &SimpleWorker{
		logger: lf.GetLoggerForType(SimpleWorker{}),
		id:     uuid.NewString(),
	}
	return wf
}
