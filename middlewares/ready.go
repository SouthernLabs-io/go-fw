package middlewares

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	lib "github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/functional/predicates"
	"github.com/southernlabs-io/go-fw/functional/slices"
	"github.com/southernlabs-io/go-fw/version"
)

const ErrCodeReadyCheckFailed = "READY_CHECK_FAILED"

type _ReadyResponse _HealthResponse
type ReadyCheckProvider interface {
	GetName() string
	ReadyCheck() error
}

type ReadyCheckMiddleware struct {
	BaseMiddleware
	readyChecks []ReadyCheckProvider
}

type ReadyCheckMiddlewareParams struct {
	lib.BaseParams
	LF          *lib.LoggerFactory
	ReadyChecks []ReadyCheckProvider `group:"ready_checks"`
}

func NewReadyCheckFx(params ReadyCheckMiddlewareParams) *ReadyCheckMiddleware {
	return NewReadyCheck(params.Conf, params.LF, params.ReadyChecks)
}

func NewReadyCheck(
	conf lib.Config,
	lf *lib.LoggerFactory,
	readyChecks []ReadyCheckProvider,
) *ReadyCheckMiddleware {
	return &ReadyCheckMiddleware{
		BaseMiddleware{conf, lf.GetLoggerForType(ReadyCheckMiddleware{})},
		slices.Filter(readyChecks, predicates.Not(predicates.Nil[ReadyCheckProvider])),
	}
}

func (m *ReadyCheckMiddleware) Setup(httpHandler lib.HTTPHandler) {
	rel, err := filepath.Rel(httpHandler.BasePath, "/ready")
	if err != nil {
		panic(errors.NewUnknownf("failed to get relative path, error: %w", err))
	}
	httpHandler.Root.GET(rel, m.ReadyCheck)
}

func (m *ReadyCheckMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityProbes
}

func (m *ReadyCheckMiddleware) ReadyCheck(ctx *gin.Context) {
	failed := map[string]error{}
	for _, check := range m.readyChecks {
		if err := check.ReadyCheck(); err != nil {
			failed[check.GetName()] = err
		}
	}
	if len(failed) > 0 {
		for s, err := range failed {
			_ = ctx.Error(errors.Newf(ErrCodeReadyCheckFailed, "ready check failed: %s with error: %w", s, err))
		}
		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			_ReadyResponse{
				Status:    "fail",
				Time:      time.Now(),
				Version:   version.Release,
				Commit:    version.Commit,
				BuildTime: version.BuildTime,
				Errors:    failed,
			},
		)
		return
	}
	ctx.JSON(
		http.StatusOK,
		_ReadyResponse{
			Status:    "pass",
			Time:      time.Now(),
			Version:   version.Release,
			Commit:    version.Commit,
			BuildTime: version.BuildTime,
		},
	)
}
