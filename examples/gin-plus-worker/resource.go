package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/southernlabs-io/go-fw/rest"
)

type SimpleResource struct {
}

func NewSimpleResource() *SimpleResource {
	return &SimpleResource{}
}

func (sr *SimpleResource) Setup(httpHandler rest.HTTPHandler) {
	httpHandler.Root.Group("simples").GET("hello", sr.Get)
}

func (sr *SimpleResource) Get(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, "Hello World!")
}
