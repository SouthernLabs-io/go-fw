package middlewares

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/functional/predicates"
	"github.com/southernlabs-io/go-fw/functional/slices"
	"github.com/southernlabs-io/go-fw/version"
)

const ErrCodeHealthCheckFailed = "HEALTH_CHECK_FAILED"

type _HealthResponse struct {
	Status    string           `json:"status"`
	Time      time.Time        `json:"time"`
	Version   string           `json:"version,omitempty"`
	Commit    string           `json:"commit,omitempty"`
	BuildTime string           `json:"build_time,omitempty"`
	Errors    map[string]error `json:"errors,omitempty"`
}

type HealthCheckProvider interface {
	GetName() string
	HealthCheck() error
}

type HealthCheckMiddleware struct {
	BaseMiddleware

	healthChecks []HealthCheckProvider
}

type HealthCheckMiddlewareParams struct {
	di.BaseParams

	HealthChecks []HealthCheckProvider `group:"health_checks"`
}

func NewHealthCheckFx(params HealthCheckMiddlewareParams) *HealthCheckMiddleware {
	return NewHealthCheck(params.Conf, params.LF, params.HealthChecks)
}

func NewHealthCheck(conf core.Config, lf *core.LoggerFactory, healthChecks []HealthCheckProvider) *HealthCheckMiddleware {
	return &HealthCheckMiddleware{
		BaseMiddleware{conf, lf.GetLoggerForType(HealthCheckMiddleware{})},
		slices.Filter(healthChecks, predicates.Not(predicates.Nil[HealthCheckProvider])),
	}
}

func (m *HealthCheckMiddleware) Setup(httpHandler core.HTTPHandler) {
	rel, err := filepath.Rel(httpHandler.BasePath, "/health")
	if err != nil {
		panic(errors.NewUnknownf("failed to get relative path, error: %w", err))
	}
	httpHandler.Root.GET(rel, m.HealthCheck)

	if len(m.healthChecks) == 0 {
		m.Logger.Warnf("No health checks provided")
	}
}

func (m *HealthCheckMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityProbes
}

func (m *HealthCheckMiddleware) HealthCheck(ctx *gin.Context) {
	failed := map[string]error{}
	for _, check := range m.healthChecks {
		if err := check.HealthCheck(); err != nil {
			failed[check.GetName()] = err
		}
	}
	if len(failed) > 0 {
		for s, err := range failed {
			_ = ctx.Error(errors.Newf(ErrCodeHealthCheckFailed, "health check failed: %s with error: %w", s, err))
		}
		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			_HealthResponse{
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
		_HealthResponse{
			Status:    "pass",
			Time:      time.Now(),
			Version:   version.Release,
			Commit:    version.Commit,
			BuildTime: version.BuildTime,
		},
	)
}
