package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"reflect"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	fw_context "github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest/middleware"
)

type StdMux interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type StdServer interface {
	StdMux

	GetBasePath() string

	Register(verb string, pathPattern string, handler http.Handler)
	RegisterFunc(verb string, pathPattern string, handler func(http.ResponseWriter, *http.Request))

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

	LC fx.Lifecycle

	LF          log.LoggerFactory
	Conf        config.Config
	Middlewares middleware.Middlewares
	Resources   Resources
}) StdServer {
	conf := deps.Conf
	lf := deps.LF
	middlewares := deps.Middlewares
	resources := deps.Resources
	lc := deps.LC
	logger := lf.GetLoggerForType(_StdServer{})

	basePath := conf.HttpServer.BasePath
	mux := http.NewServeMux()
	preMuxHandler := middlewares.Handle(middleware.MiddlewarePriorityHighest, middleware.MiddlewarePriorityBeforeMux, mux)
	srv := &http.Server{
		Handler: preMuxHandler,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			// Use a custom key/value context
			return fw_context.NewContextWithStore(ctx)
		},
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
				panic(errors.NewUnknownf("failed to run http server on: %s, error: %w", bindAddress, err))
			}
			logger.Infof("Running http server on: %s", bindAddress)
			go func() {
				err := srv.Serve(ln)
				if !errors.Is(err, http.ErrServerClosed) {
					panic(errors.Newf(errors.ErrCodeBadState, "failed to run http server, error: %w", err))
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

// Handle registers the handler for the given pattern. If the pattern is already registered, Handle panics. This is a low level method that doesn't prepend the BasePath to the pattern. Use Register or RegisterFunc instead.
func (srv *_StdServer) Handle(pattern string, handler http.Handler) {
	srv.logger.Infof("Registering handler for pattern: %s", pattern)
	// Apply after-mux middlewares
	handler = srv.middlewares.Handle(middleware.MiddlewarePriorityAfterMux, middleware.MiddlewarePriorityLowest, handler)
	srv.mux.Handle(pattern, handler)
}

// HandleFunc registers the handler function for the given pattern. If the pattern is already registered, HandleFunc panics. This is a low level method that doesn't prepend the BasePath to the pattern. Use Register or RegisterFunc instead.
func (srv *_StdServer) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.Handle(pattern, http.HandlerFunc(handler))
}

// Register registers a handler for the given HTTP verb and path pattern. Use one of the http.Method* constants for the verb. Example: http.MethodGet, http.MethodPost, etc. The BasePath will be prepended to the path pattern.
func (srv *_StdServer) Register(verb string, pathPattern string, handler http.Handler) {
	if srv.basePath != "" {
		pathPattern = srv.basePath + pathPattern
	}
	srv.Handle(verb+" "+pathPattern, handler)
}

// RegisterFunc registers a handler for the given HTTP verb and path pattern. Use one of the http.Method* constants for the verb. Example: http.MethodGet, http.MethodPost, etc. The BasePath will be prepended to the path pattern.
func (srv *_StdServer) RegisterFunc(verb string, pathPattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.Register(verb, pathPattern, http.HandlerFunc(handler))
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

func WriteJSON(ctx context.Context, resp http.ResponseWriter, statusCode int, obj any) {
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(statusCode)
	err := json.NewEncoder(resp).Encode(obj)
	if err != nil {
		log.GetLoggerFromCtx(ctx).Errorf("failed to write json response, error: %s", err)
	}
}

func AbortWithStatus(ctx context.Context, resp http.ResponseWriter, statusCode int) {
	resp.WriteHeader(statusCode)
}
