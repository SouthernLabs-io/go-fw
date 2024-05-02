package main

import (
	"context"
	"time"

	"github.com/google/uuid"

	lib "github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/worker"
)

type SimpleWorker struct {
	logger lib.Logger
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

func NewSimpleWorker(logger lib.Logger) *SimpleWorker {
	wf := &SimpleWorker{
		logger: logger,
		id:     uuid.NewString(),
	}
	return wf
}
