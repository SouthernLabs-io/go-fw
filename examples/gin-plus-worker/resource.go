package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/southernlabs-io/go-fw/core"
)

type SimpleResource struct {
	logger core.Logger
}

func NewSimpleResource(logger core.Logger) *SimpleResource {
	return &SimpleResource{
		logger: logger,
	}
}

func (sr *SimpleResource) Setup(httpHandler core.HTTPHandler) {
	httpHandler.Root.Group("simples").GET("hello", sr.Get)
}

func (sr *SimpleResource) Get(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, "Hello World!")
}
