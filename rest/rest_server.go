package rest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest/middleware"
)

type HandleOptions struct {
	// The priority starting range for the middlewares to be applied before the handler. It must be equal or higher than middleware.MiddlewarePriorityAfterMux. If no set, it defaults to middleware.MiddlewarePriorityAfterMux.
	MiddlewarePriorityFrom middleware.MiddlewarePriority

	// The priority ending range for the middlewares to be applied before the handler. It must be equal or higher than MiddlewarePriorityFrom. If not set, it defaults to middleware.MiddlewarePriorityLowest.
	MiddlewarePriorityTo middleware.MiddlewarePriority
}

type StdMux interface {
	// Handle registers the handler for the given pattern. If the pattern is already registered, Handle panics. This is a low level method that doesn't prepend the BasePath to the pattern. Use Register or RegisterFunc instead.
	Handle(pattern string, handler http.Handler)

	// See Handle
	HandleWithOptions(pattern string, handler http.Handler, options HandleOptions)

	// HandleFunc registers the handler function for the given pattern. If the pattern is already registered, HandleFunc panics. This is a low level method that doesn't prepend the BasePath to the pattern. Use Register or RegisterFunc instead.
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))

	// See HandleFunc
	HandleFuncWithOptions(pattern string, handler func(http.ResponseWriter, *http.Request), options HandleOptions)

	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type StdServer interface {
	StdMux

	GetBasePath() string

	// Register registers a handler for the given HTTP verb and path pattern. Use one of the http.Method* constants for the verb. Example: http.MethodGet, http.MethodPost, etc. The BasePath will be prepended to the path pattern.
	Register(verb string, pathPattern string, handler http.Handler)

	// See Register
	RegisterWithOptions(verb string, pathPattern string, handler http.Handler, options HandleOptions)

	// RegisterFunc registers a handler for the given HTTP verb and path pattern. Use one of the http.Method* constants for the verb. Example: http.MethodGet, http.MethodPost, etc. The BasePath will be prepended to the path pattern.
	RegisterFunc(verb string, pathPattern string, handler func(http.ResponseWriter, *http.Request))

	// See RegisterFunc
	RegisterFuncWithOptions(verb string, pathPattern string, handler func(http.ResponseWriter, *http.Request), options HandleOptions)

	Close() error
	Shutdown(ctx context.Context) error
}

type _StdServer struct {
	logger log.Logger

	httpSrv       *http.Server
	basePath      string
	middlewares   middleware.Middlewares
	preMuxHandler http.Handler
	mux           *http.ServeMux
}

// Make sure _StdServer implements StdServer
var _ StdServer = (*_StdServer)(nil)

func NewStdServer(deps struct {
	fx.In

	Lc fx.Lifecycle
	Sd fx.Shutdowner

	LF          log.LoggerFactory
	Conf        config.Config
	Middlewares middleware.Middlewares
	Resources   Resources
}) StdServer {
	conf := deps.Conf
	lf := deps.LF
	middlewares := deps.Middlewares
	resources := deps.Resources
	lc := deps.Lc
	sd := deps.Sd
	logger := lf.GetLoggerForType(_StdServer{})

	basePath := conf.HttpServer.BasePath
	mux := http.NewServeMux()
	preMuxHandler := middlewares.Apply(middleware.MiddlewarePriorityHighest, middleware.MiddlewarePriorityBeforeMuxInclusive, mux) // Include BeforeMux middlewares
	srv := &http.Server{
		Handler: preMuxHandler,
	}

	stdServer := &_StdServer{
		logger:        logger,
		httpSrv:       srv,
		basePath:      basePath,
		middlewares:   middlewares,
		preMuxHandler: preMuxHandler,
		mux:           mux,
	}

	// Register Resources
	for _, r := range resources {
		logger.Infof("Registering resource: %s", reflect.TypeOf(r).String())
		r.Register(stdServer)
	}

	lc.Append(fx.StartStopHook(
		func() error {
			bindAddress := fmt.Sprintf("%s:%d", conf.HttpServer.BindAddress, conf.HttpServer.Port)
			ln, err := net.Listen("tcp", bindAddress)
			if err != nil {
				return errors.NewUnknownf("failed to run http server on: %s, error: %w", bindAddress, err)
			}
			logger.Infof("Running http server on: %s", bindAddress)
			go func() {
				err := srv.Serve(ln)
				if !errors.Is(err, http.ErrServerClosed) {
					logger.Errorf("Http server failed with error: %s", err)
					sd.Shutdown(fx.ExitCode(-1))
				}
			}()
			return nil
		},
		func(ctx context.Context) {
			err := srv.Shutdown(ctx)
			if err != nil {
				logger.Errorf("Error while shutting down http server: %s", err)
			}
		},
	))

	return stdServer
}

func (srv *_StdServer) GetBasePath() string {
	return srv.basePath
}

func (srv *_StdServer) handleWithOptions(pattern string, handler http.Handler, options HandleOptions) {

	from := middleware.MiddlewarePriorityAfterMux
	to := middleware.MiddlewarePriorityLowest

	// Clamp left side of the range to MiddlewarePriorityAfterMux
	from = max(from, options.MiddlewarePriorityFrom)

	if options.MiddlewarePriorityTo != middleware.MiddlewarePriorityNotSet {
		to = max(from, options.MiddlewarePriorityTo)
	}

	// Apply after-mux middlewares
	handler = srv.middlewares.Apply(from, to, handler)
	srv.mux.Handle(pattern, handler)
}

func (srv *_StdServer) Handle(pattern string, handler http.Handler) {
	srv.logger.Infof("Registering handler for pattern: %s", pattern)
	srv.handleWithOptions(pattern, handler, HandleOptions{})
}

func (srv *_StdServer) HandleWithOptions(pattern string, handler http.Handler, options HandleOptions) {
	srv.logger.Infof("Registering handler with options for pattern: %s", pattern)
	srv.handleWithOptions(pattern, handler, options)
}

func (srv *_StdServer) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.Handle(pattern, http.HandlerFunc(handler))
}

func (srv *_StdServer) HandleFuncWithOptions(pattern string, handler func(http.ResponseWriter, *http.Request), options HandleOptions) {
	srv.HandleWithOptions(pattern, http.HandlerFunc(handler), options)
}

func (srv *_StdServer) Register(verb string, pathPattern string, handler http.Handler) {
	if srv.basePath != "" {
		pathPattern = srv.basePath + pathPattern
	}
	srv.Handle(verb+" "+pathPattern, handler)
}

func (srv *_StdServer) RegisterWithOptions(verb string, pathPattern string, handler http.Handler, options HandleOptions) {
	if srv.basePath != "" {
		pathPattern = srv.basePath + pathPattern
	}
	srv.HandleWithOptions(verb+" "+pathPattern, handler, options)
}

func (srv *_StdServer) RegisterFunc(verb string, pathPattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.Register(verb, pathPattern, http.HandlerFunc(handler))
}

func (srv *_StdServer) RegisterFuncWithOptions(verb string, pathPattern string, handler func(http.ResponseWriter, *http.Request), options HandleOptions) {
	srv.RegisterWithOptions(verb, pathPattern, http.HandlerFunc(handler), options)
}

func (srv *_StdServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.mux.ServeHTTP(w, r)
}

func (srv *_StdServer) Close() error {
	return srv.httpSrv.Close()
}

func (srv *_StdServer) Shutdown(ctx context.Context) error {
	return srv.httpSrv.Shutdown(ctx)
}
