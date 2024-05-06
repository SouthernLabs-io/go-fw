package middlewares

import (
	"reflect"
	"slices"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/core"
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
	Setup(httpHandler core.HTTPHandler)
	Priority() MiddlewarePriority
	GetLogger() core.Logger
}

type BaseMiddleware struct {
	Conf   core.Config
	Logger core.Logger
}

func (m *BaseMiddleware) GetLogger() core.Logger {
	return m.Logger
}

func ProvideAsMiddleware(provider any, anns ...fx.Annotation) fx.Option {
	return core.FxProvideAs[Middleware](provider, anns, []fx.Annotation{fx.ResultTags(`group:"middlewares"`)})
}

type Middlewares []Middleware

func NewMiddlewares(deps struct {
	fx.In

	LF          *core.LoggerFactory
	Middlewares []Middleware `group:"middlewares"`
	HTTPHandler core.HTTPHandler
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

func (ms Middlewares) Setup(httpHandler core.HTTPHandler) {
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
