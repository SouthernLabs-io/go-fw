package rest

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	gintrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
)

type HTTPHandler struct {
	Engine   *gin.Engine
	Root     GinRouterGroup
	BasePath string
}

// NewHTTPHandler creates a new request handler
func NewHTTPHandler(
	conf core.Config,
	lf *core.LoggerFactory,
	lc fx.Lifecycle,
) HTTPHandler {
	logger := lf.GetLoggerForType(HTTPHandler{})
	ginLogger := lf.GetLoggerForType(gin.Engine{})
	gin.DefaultWriter = core.NewDefaultGinWriter(ginLogger)
	gin.DefaultErrorWriter = core.NewDefaultErrorGinWriter(ginLogger)
	if conf.Env.Type == core.EnvTypeProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	engine := gin.New()
	engine.ContextWithFallback = true

	modules := []gin.HandlerFunc{
		// CORS
		cors.New(conf.HttpServer.CORS),
	}

	if conf.Datadog.Tracing {
		modules = append(modules, gintrace.Middleware(conf.Name))
	}

	engine.Use(modules...)

	basePath := conf.HttpServer.BasePath
	srv := &http.Server{Handler: engine.Handler()}

	lc.Append(fx.StartStopHook(
		func() error {
			bindAddress := fmt.Sprintf("%s:%d", conf.HttpServer.BindAddress, conf.HttpServer.Port)
			ln, err := net.Listen("tcp", bindAddress)
			if err != nil {
				panic(errors.NewUnknownf("failed to run gin server on: %s, error: %w", bindAddress, err))
			}
			logger.Infof("Running gin server on: %s", bindAddress)
			go func() {
				err := srv.Serve(ln)
				if !errors.Is(err, http.ErrServerClosed) {
					panic(errors.Newf(errors.ErrCodeBadState, "failed to run gin server, error: %w", err))
				}
			}()
			return nil
		},
		func(ctx context.Context) {
			err := srv.Shutdown(ctx)
			if err != nil {
				logger.Errorf("Error while shutting down gin: %s", err)
			}
		},
	))
	root := NewGinRouterGroup(engine.Group(basePath))

	return HTTPHandler{
		Engine:   engine,
		Root:     root,
		BasePath: basePath,
	}
}
