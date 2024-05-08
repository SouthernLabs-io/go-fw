package worker

import (
	"log/slog"

	"github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/log"
)

var workerInfoKey = context.CtxKey("_fw_worker")

type _WorkerInfo struct {
	Name string
	ID   string
}

// NewWorkerContext creates a new child context with worker info and a configured logger with the worker name and id.
// Dependencies should be added to the context using NewContextWithDeps.
func NewWorkerContext(
	parentCtx context.Context,
	name string,
	id string,
) context.Context {
	ctx := context.CtxSetValue(parentCtx, workerInfoKey, _WorkerInfo{name, id})

	// Add worker info to the context
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.Group("worker", slog.String("name", name), slog.String("id", id)))
	return ctx
}

func GetWorkerName(ctx log.ValueContext) (string, bool) {
	wCtx, ok := ctx.Value(workerInfoKey).(_WorkerInfo)
	return wCtx.Name, ok
}

func GetWorkerID(ctx log.ValueContext) (string, bool) {
	wCtx, ok := ctx.Value(workerInfoKey).(_WorkerInfo)
	return wCtx.ID, ok
}
