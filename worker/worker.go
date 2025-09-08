package worker

import (
	"time"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/distributedlock"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/panics"
)

var ErrWorkerHandlerNoWorkers = errors.Newf("WORKER_HANDLER_NO_WORKERS", "worker handler has no workers")
var errWorkerHandlerStopped = errors.Newf("WORKER_HANDLER_STOPPED", "worker handler stopped")

var ErrCodeWorkerError = "WORKER_ERROR"

type Handler interface {
}

//go:generate stringer -type=ConcurrencyMode
type ConcurrencyMode int

const (
	ConcurrencyModeMulti  ConcurrencyMode = iota
	ConcurrencyModeSingle ConcurrencyMode = iota
)

type ConcurrencyConfig struct {
	Mode          ConcurrencyMode
	SingleLockTTL time.Duration
}

type LongRunningWorker interface {
	GetName() string
	GetID() string
	GetConcurrency() ConcurrencyConfig
	Run(ctx context.Context) error
}

type LongRunningWorkerHandler struct {
	conf   config.Config
	logger log.Logger
	db     database.DB
	sd     fx.Shutdowner

	dlFactory distributedlock.Factory
	workers   []LongRunningWorker

	// these are the context and cancelCauseFunc for the workerHandler
	ctx             context.Context
	cancelCauseFunc context.CancelCauseFunc

	closedChn chan any
}

type LongRunningWorkerHandlerParams struct {
	di.BaseParams
	DLFactory distributedlock.Factory
	Workers   []LongRunningWorker `group:"long_running_workers"`
}

func NewLongRunningWorkerHandlerFx(params LongRunningWorkerHandlerParams) *LongRunningWorkerHandler {
	return NewLongRunningWorkerHandler(
		params.Conf,
		params.LF,
		params.DB,
		params.FxLifecycle,
		params.FxShutdowner,
		params.DLFactory,
		params.Workers,
	)
}

func NewLongRunningWorkerHandler(
	conf config.Config,
	lf *log.LoggerFactory,
	db database.DB,
	fxLifecycle fx.Lifecycle,
	fxShutdowner fx.Shutdowner,
	dlFactory distributedlock.Factory,
	workers []LongRunningWorker,
) *LongRunningWorkerHandler {
	wHandler := &LongRunningWorkerHandler{
		conf:      conf,
		logger:    lf.GetLoggerForType(LongRunningWorkerHandler{}),
		sd:        fxShutdowner,
		db:        db,
		dlFactory: dlFactory,
		workers:   workers,
		closedChn: make(chan any),
	}
	fxLifecycle.Append(fx.StartStopHook(
		func() { wHandler.Start() },
		func(ctx context.Context) { wHandler.Stop(ctx) },
	))

	ctx := context.Background()
	ctx = db.SetCtx(ctx)

	wHandler.ctx, wHandler.cancelCauseFunc = context.WithCancelCause(ctx)

	return wHandler
}

// Start starts all workers asynchronously and returns immediately
func (h *LongRunningWorkerHandler) Start() {
	go func() {
		err := h.Run()
		if err != nil {
			if errors.Is(err, errWorkerHandlerStopped) {
				h.logger.Infof("Long running worker handler stopped")
			} else if errors.Is(err, ErrWorkerHandlerNoWorkers) {
				h.shutdownFxApp(err)
			} else {
				h.logger.Errorf("Long running worker handler stopped with error: %s", err)
			}
		} else {
			h.logger.Errorf("Long running worker handler unexpectedly stopped without error")
		}
	}()
}

// Run runs all workers asynchronously and blocks until all workers are done.
// It returns error if any of the workers returns error from LongRunningWorker.Run
func (h *LongRunningWorkerHandler) Run() error {
	h.logger.Infof("Long running workers: %d", len(h.workers))
	defer close(h.closedChn)

	handlerErrChn := make(chan error, len(h.workers))
	for _, worker := range h.workers {
		grWorker := worker
		grCtx := NewWorkerContext(h.ctx, grWorker.GetName(), grWorker.GetID())
		go func() {
			var err error
			logger := log.GetLoggerFromCtx(grCtx)
			switch grWorker.GetConcurrency().Mode {
			case ConcurrencyModeMulti:
				err = h.multiWorkerRunner(grCtx, grWorker)
			case ConcurrencyModeSingle:
				err = h.singleWorkerRunner(grCtx, grWorker)
			default:
				err = errors.Newf(
					errors.ErrCodeBadState,
					"worker: %s has invalid concurrency mode: %d",
					grWorker.GetName(),
					grWorker.GetConcurrency(),
				)
			}
			if err != nil {
				if errors.Is(err, errWorkerHandlerStopped) {
					handlerErrChn <- errWorkerHandlerStopped
				} else {
					handlerErrChn <- err
				}
			} else {
				handlerErrChn <- errors.Newf(
					ErrCodeWorkerError,
					"worker: %s un-expectedly finished processing without error",
					grWorker.GetName(),
				)
			}

			logger.Infof("Worker: %s stopped", grWorker.GetName())
		}()
	}

	var errs []error
	for i := 0; i < len(h.workers); i++ {
		err := <-handlerErrChn
		h.shutdownFxApp(err)
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		return errors.Newf(ErrCodeWorkerError, "%w", errors.Join(errs...))
	}

	if len(h.workers) == 0 {
		return ErrWorkerHandlerNoWorkers
	}
	return nil
}

func (h *LongRunningWorkerHandler) singleWorkerRunner(ctx context.Context, worker LongRunningWorker) error {
	logger := log.GetLoggerFromCtx(ctx)
	dl := h.dlFactory.NewDistributedLock(worker.GetName(), worker.GetConcurrency().SingleLockTTL)
	for {
		// Use a function closure to use defer to unlock the lock
		err := func() (err error) {
			defer panics.DeferredPanicToError(
				&err,
				"long running worker handler panicked while managing lock for single worker: %s",
				worker.GetName(),
			)
			// Lock will use the handler context, so errors when locking will not be linked to the worker
			err = dl.Lock(h.ctx)
			if err != nil {
				return errors.NewUnknownf("failed to lock worker: %s error: %w", worker.GetName(), err)
			}

			// Defer unlock
			ndcCtx := context.NoDeadlineAndNotCancellableContext(ctx)
			defer func(dl distributedlock.DistributedLock, ctx context.Context) {
				err := dl.Unlock(ctx)
				if err != nil {
					logger.Warnf("Failed to unlock worker: %s error: %s", worker.GetName(), err)
				}
			}(dl, ndcCtx)

			// AutoExtend
			wCtx, err := dl.AutoExtend(ctx)
			if err != nil {
				return err
			}

			logger.Infof("Running worker: %s, with concurrency: %+v", worker.GetName(), worker.GetConcurrency())
			return worker.Run(wCtx)
		}()
		if err != nil {
			if errors.Is(err, errWorkerHandlerStopped) || errors.IsCode(err, errors.ErrCodePanic) {
				return err
			}
			var fwErr *errors.Error
			if errors.AsCode(err, &fwErr, distributedlock.ErrCodeLockNotAutoExtended) {
				logger.Info(fwErr.Message)
				continue
			}
			return errors.Newf(ErrCodeWorkerError, "single worker: %s error: %w", worker.GetName(), err)
		}
		return nil
	}
}

func (h *LongRunningWorkerHandler) multiWorkerRunner(ctx context.Context, worker LongRunningWorker) (err error) {
	defer func() {
		if err != nil {
			err = errors.Newf(ErrCodeWorkerError, "multi worker: %s error: %w", worker.GetName(), err)
		}
	}()
	defer panics.DeferredPanicToError(&err, "worker: %s panicked", worker.GetName())
	logger := log.GetLoggerFromCtx(ctx)
	logger.Infof("Running worker: %s, with concurrency: %+v", worker.GetName(), worker.GetConcurrency())
	err = worker.Run(ctx)
	return
}

func (h *LongRunningWorkerHandler) shutdownFxApp(err error) {
	select {
	case <-h.ctx.Done():
		return
	default:
		h.logger.Errorf("Starting graceful shutdown because of: %s", err)
		shErr := h.sd.Shutdown(fx.ExitCode(1))
		if shErr != nil {
			panic(errors.Newf(
				errors.ErrCodeUnknown,
				"failed to start graceful shutdown: %w, panicking because of: %w",
				shErr,
				err,
			))
		}
	}
}

func (h *LongRunningWorkerHandler) Stop(ctx context.Context) {
	h.logger.Infof("Stop: long running worker handler, workers: %d", len(h.workers))
	h.cancelCauseFunc(errWorkerHandlerStopped)

	if _, ok := ctx.Deadline(); ok {
		select {
		case <-h.closedChn:
			h.logger.Infof("Stop: long running worker handler, workers: %d, done", len(h.workers))
		case <-ctx.Done():
			err := context.Cause(ctx)
			if errors.Is(err, context.DeadlineExceeded) {
				h.logger.Warnf("Stop: long running worker handler, workers: %d, not finished in time", len(h.workers))
			}
		}
	} else {
		<-h.closedChn
		h.logger.Infof("Stop: long running worker handler, workers: %d, done", len(h.workers))
	}
}

func ProvideAsLongRunningWorker(provider any, anns ...fx.Annotation) fx.Option {
	return di.FxProvideAs[LongRunningWorker](provider, anns, []fx.Annotation{fx.ResultTags(`group:"long_running_workers"`)})
}

var ModuleWorkerHandler = di.FxProvideAs[Handler](
	NewLongRunningWorkerHandlerFx,
	nil,
	[]fx.Annotation{fx.ResultTags(`group:"worker_handlers"`)},
)
