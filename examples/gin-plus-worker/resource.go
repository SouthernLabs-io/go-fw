package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	lib "github.com/southernlabs-io/go-fw/core"
)

type SimpleResource struct {
	logger lib.Logger
}

func NewSimpleResource(logger lib.Logger) *SimpleResource {
	return &SimpleResource{
		logger: logger,
	}
}

func (sr *SimpleResource) Setup(httpHandler lib.HTTPHandler) {
	httpHandler.Root.Group("simples").GET("hello", sr.Get)
}

func (sr *SimpleResource) Get(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, "Hello World!")
}
