package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	fw_context "github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

type HTTPServer struct {
	StdServer *http.Server
	StdMux    *http.ServeMux
	BasePath  string
}

// NewHTTPServer creates a new http server with an empty Mux.
func NewHTTPServer(
	conf config.Config,
	lf log.LoggerFactory,
	lc fx.Lifecycle,
) HTTPServer {
	logger := lf.GetLoggerForType(HTTPServer{})

	basePath := conf.HttpServer.BasePath
	mux := http.NewServeMux()
	srv := &http.Server{
		Handler: mux,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			// Use a custom key/value context
			return fw_context.NewKeyValueContext(ctx)
		},
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

	return HTTPServer{
		StdServer: srv,
		StdMux:    mux,
		BasePath:  basePath,
	}
}

func (srv HTTPServer) Handle(verb string, pathPattern string, handler http.Handler) {
	if srv.BasePath != "" {
		pathPattern = srv.BasePath + pathPattern
	}
	srv.StdMux.Handle(verb+" "+pathPattern, handler)
}

func (srv HTTPServer) HandleGet(pathPattern string, handler http.Handler) {
	srv.Handle("GET", pathPattern, handler)
}

func (srv HTTPServer) HandleGetFunc(pathPattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.HandleGet(pathPattern, http.HandlerFunc(handler))
}

func (srv HTTPServer) HandlePost(pathPattern string, handler http.Handler) {
	srv.Handle("POST", pathPattern, handler)
}

func (srv HTTPServer) HandlePostFunc(pathPattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.HandlePost(pathPattern, http.HandlerFunc(handler))
}

func (srv HTTPServer) HandlePatch(pathPattern string, handler http.Handler) {
	srv.Handle("PATCH", pathPattern, handler)
}

func (srv HTTPServer) HandlePatchFunc(pathPattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.HandlePatch(pathPattern, http.HandlerFunc(handler))
}

func (srv HTTPServer) HandlePut(pathPattern string, handler http.Handler) {
	srv.Handle("PUT", pathPattern, handler)
}

func (srv HTTPServer) HandlePutFunc(pathPattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.HandlePut(pathPattern, http.HandlerFunc(handler))
}

func (srv HTTPServer) HandleDelete(pathPattern string, handler http.Handler) {
	srv.Handle("DELETE", pathPattern, handler)
}

func (srv HTTPServer) HandleDeleteFunc(pathPattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.HandleDelete(pathPattern, http.HandlerFunc(handler))
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
