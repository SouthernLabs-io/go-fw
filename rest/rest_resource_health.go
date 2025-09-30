package rest

import (
	"net/http"
	"time"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/functional/predicates"
	"github.com/southernlabs-io/go-fw/functional/slices"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest/healthcheck"
	"github.com/southernlabs-io/go-fw/rest/middleware"
	"github.com/southernlabs-io/go-fw/version"
	"go.uber.org/fx"
)

type _HealthResponse struct {
	Status    string           `json:"status"`
	Time      time.Time        `json:"time"`
	Version   string           `json:"version,omitempty"`
	Commit    string           `json:"commit,omitempty"`
	BuildTime string           `json:"build_time,omitempty"`
	Errors    map[string]error `json:"errors,omitempty"`
}

type HealthCheckResource struct {
	logger       log.Logger
	healthChecks []healthcheck.Provider
}

type HealthCheckParams struct {
	fx.In

	Conf         config.Config
	LF           log.LoggerFactory
	HealthChecks []healthcheck.Provider `group:"rest_healthcheck_providers"`
}

func NewHealthCheckFx(params HealthCheckParams) *HealthCheckResource {
	return NewHealthCheck(params.Conf, params.LF, params.HealthChecks)
}

func NewHealthCheck(conf config.Config, lf log.LoggerFactory, healthChecks []healthcheck.Provider) *HealthCheckResource {
	return &HealthCheckResource{
		logger:       lf.GetLoggerForType(HealthCheckResource{}),
		healthChecks: slices.Filter(healthChecks, predicates.Not(predicates.Nil[healthcheck.Provider])),
	}
}

var _ Resource = new(HealthCheckResource)

func (m *HealthCheckResource) Register(srv StdServer) {
	if len(m.healthChecks) == 0 {
		m.logger.Warn("No health checks provided, health endpoint will always return OK")
	}

	srv.RegisterFuncWithOptions(http.MethodGet, "/health", m.HealthCheck, HandleOptions{
		MiddlewarePriorityTo: middleware.MiddlewarePriorityBeforeMux,
	})
}

func (m *HealthCheckResource) HealthCheck(w http.ResponseWriter, r *http.Request) {
	resp := _HealthResponse{
		Status:    "pass",
		Time:      time.Now(),
		Version:   version.Release,
		Commit:    version.Commit,
		BuildTime: version.BuildTime,
	}

	if len(m.healthChecks) != 0 {
		failed := map[string]error{}
		for _, check := range m.healthChecks {
			if err := check.HealthCheck(); err != nil {
				failed[check.GetName()] = err
			}
		}
		if len(failed) > 0 {
			resp.Status = "fail"
			resp.Errors = failed
		}
	}
	statusCode := http.StatusOK
	if resp.Status != "pass" {
		statusCode = http.StatusInternalServerError
	}
	ctx := r.Context()
	err := WriteJSON(ctx, w, statusCode, resp)
	if err != nil {
		log.GetLoggerFromCtx(ctx).Errorf("Failed to write health check response: %v", resp)
	}
}
