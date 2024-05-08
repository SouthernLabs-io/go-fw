package middleware

//go:generate mockery --all --with-expecter=true --keeptree=false --case=underscore

import (
	"reflect"
	"slices"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest"
)

type MiddlewarePriority int

const (
	MiddlewarePriorityHighest MiddlewarePriority = iota * 1_000

	MiddlewarePriorityProbes
	MiddlewarePriorityAuthN
	MiddlewarePriorityAuthZ
	MiddlewarePriorityHeader
	MiddlewarePriorityBody

	MiddlewarePriorityDefault
	MiddlewarePriorityLowest
)

type Middleware interface {
	Setup(httpHandler rest.HTTPHandler)
	Priority() MiddlewarePriority
	GetLogger() log.Logger
}

type BaseMiddleware struct {
	Conf   config.Config
	Logger log.Logger
}

func (m *BaseMiddleware) GetLogger() log.Logger {
	return m.Logger
}

func ProvideAsMiddleware(provider any, anns ...fx.Annotation) fx.Option {
	return di.FxProvideAs[Middleware](provider, anns, []fx.Annotation{fx.ResultTags(`group:"middlewares"`)})
}

type Middlewares []Middleware

func NewMiddlewares(deps struct {
	fx.In

	LF          *log.LoggerFactory
	Middlewares []Middleware `group:"middlewares"`
	HTTPHandler rest.HTTPHandler
}) Middlewares {
	// We want a stable order
	slices.SortFunc(deps.Middlewares, func(a, b Middleware) int {
		if a.Priority() < b.Priority() {
			return -1
		} else if a.Priority() == b.Priority() {
			// Same priority, try with type name
			tA := reflect.TypeOf(a).Elem()
			tB := reflect.TypeOf(b).Elem()
			if tA.Name() < tB.Name() {
				return -1
			} else if tA.Name() == tB.Name() {
				// Same name, finally try with package name
				if tA.PkgPath() < tB.PkgPath() {
					return -1
				} else if tA.PkgPath() == tB.PkgPath() {
					deps.LF.GetLoggerForType(Middlewares{}).Warnf(
						"Not stable sort on middlewares, you registered the same middleware twice: %s.%s",
						tA.PkgPath(),
						tA.Name(),
					)
					return 0
				}
			}
		}
		return 1
	})
	Middlewares(deps.Middlewares).Setup(deps.HTTPHandler)
	return deps.Middlewares
}

func (ms Middlewares) Setup(httpHandler rest.HTTPHandler) {
	for _, middleware := range ms {
		t := reflect.TypeOf(middleware).Elem()
		middleware.GetLogger().Debugf("Setting up middleware %s.%s", t.PkgPath(), t.Name())
		middleware.Setup(httpHandler)
	}
}

var Module = fx.Options(
	fx.Invoke(NewMiddlewares),

	//Default providers
	ProvideAsMiddleware(NewHealthCheckFx),
	ProvideAsMiddleware(NewReadyCheckFx),
	ProvideAsMiddleware(NewPanicRecovery),
	ProvideAsMiddleware(NewErrorHandlerFx),
)
