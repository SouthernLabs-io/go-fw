package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/southernlabs-io/go-fw/core"
)

var workerInfoKey = core.CtxKey("_fw_worker")

type _WorkerInfo struct {
	Name string
	ID   string
}

func NewWorkerRandomIDWithHostname() string {
	return fmt.Sprintf("%s@%s", uuid.NewString(), core.CachedHostname())
}

// NewWorkerContext creates a new child context with worker info and a configured logger with the worker name and id.
// Dependencies should be added to the context using NewContextWithDeps.
func NewWorkerContext(
	parentCtx context.Context,
	name string,
	id string,
) context.Context {
	ctx := core.CtxSetValue(parentCtx, workerInfoKey, _WorkerInfo{name, id})

	// Add worker info to the context
	ctx = core.CtxAppendLoggerAttrs(ctx, slog.Group("worker", slog.String("name", name), slog.String("id", id)))
	return ctx
}

func GetWorkerName(ctx core.ValueContext) (string, bool) {
	wCtx, ok := ctx.Value(workerInfoKey).(_WorkerInfo)
	return wCtx.Name, ok
}

func GetWorkerID(ctx core.ValueContext) (string, bool) {
	wCtx, ok := ctx.Value(workerInfoKey).(_WorkerInfo)
	return wCtx.ID, ok
}
