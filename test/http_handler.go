package test

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest"
)

func NewTestHTTPHandler(conf config.Config, lf *log.LoggerFactory) rest.HTTPHandler {
	ginLogger := lf.GetLoggerForType(&gin.Engine{})
	gin.DefaultWriter = rest.NewDefaultGinWriter(ginLogger)
	gin.DefaultErrorWriter = rest.NewDefaultErrorGinWriter(ginLogger)
	gin.SetMode(gin.DebugMode) //There is a TestMode, but it doesn't print logs, so it is not useful

	engine := gin.New()
	engine.ContextWithFallback = true

	basePath := conf.HttpServer.BasePath
	root := rest.NewGinRouterGroup(engine.Group(basePath))

	return rest.HTTPHandler{
		Engine:   engine,
		Root:     root,
		BasePath: "",
	}
}

var TestModuleHTTPHandler = fx.Provide(NewTestHTTPHandler)
