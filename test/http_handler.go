package test

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	lib "github.com/southernlabs-io/go-fw/core"
)

func NewTestHTTPHandler(conf lib.Config, lf *lib.LoggerFactory) lib.HTTPHandler {
	ginLogger := lf.GetLoggerForType(&gin.Engine{})
	gin.DefaultWriter = lib.NewDefaultGinWriter(ginLogger)
	gin.DefaultErrorWriter = lib.NewDefaultErrorGinWriter(ginLogger)
	gin.SetMode(gin.DebugMode) //There is a TestMode, but it doesn't print logs, so it is not useful

	engine := gin.New()
	engine.ContextWithFallback = true

	basePath := conf.HttpServer.BasePath
	root := lib.NewGinRouterGroup(engine.Group(basePath))

	return lib.HTTPHandler{
		Engine:   engine,
		Root:     root,
		BasePath: "",
	}
}

var TestModuleHTTPHandler = fx.Provide(NewTestHTTPHandler)
