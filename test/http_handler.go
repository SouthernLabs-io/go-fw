package test

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/rest"
)

func NewTestHTTPHandler(conf core.Config, lf *core.LoggerFactory) rest.HTTPHandler {
	ginLogger := lf.GetLoggerForType(&gin.Engine{})
	gin.DefaultWriter = core.NewDefaultGinWriter(ginLogger)
	gin.DefaultErrorWriter = core.NewDefaultErrorGinWriter(ginLogger)
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
