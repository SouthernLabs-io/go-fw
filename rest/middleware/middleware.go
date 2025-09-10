package middleware

//go:generate mockery --all --with-expecter=true --keeptree=false --case=underscore

import (
	"net/http"
	"reflect"
	"slices"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/log"
)

type MiddlewarePriority int

const (
	MiddlewarePriorityHighest MiddlewarePriority = iota * 1_000

	MiddlewarePriorityBeforeMux

	MiddlewarePriorityAfterMux
	MiddlewarePriorityAuthN
	MiddlewarePriorityAuthZ
	MiddlewarePriorityDefault

	MiddlewarePriorityLowest = MiddlewarePriorityDefault + 1_000
)

type Middleware interface {
	Priority() MiddlewarePriority
	Handle(http.Handler) http.Handler
}

type BaseMiddleware struct {
	Conf   config.Config
	Logger log.Logger
}

func ProvideAsMiddleware(provider any, anns ...fx.Annotation) fx.Option {
	return di.FxProvideAs[Middleware](provider, anns, []fx.Annotation{fx.ResultTags(`group:"rest_middlewares"`)})
}

type Middlewares []Middleware

func NewMiddlewares(deps struct {
	fx.In

	LF          log.LoggerFactory
	Middlewares []Middleware `group:"rest_middlewares"`
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
	return deps.Middlewares
}

var Module = fx.Options(
	fx.Provide(NewMiddlewares),

	//Default providers
	ProvideAsMiddleware(NewRequestLogger),
)
