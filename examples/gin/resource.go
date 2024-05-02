package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	lib "github.com/southernlabs-io/go-fw/core"
)

type MyResource struct {
	httpHandler lib.HTTPHandler
}

func NewMyResource(httpHandler lib.HTTPHandler) MyResource {
	return MyResource{httpHandler: httpHandler}
}

func (r MyResource) Setup(httpHandler lib.HTTPHandler) {
	router := httpHandler.Root
	router.GET("check", r.Check)
	getHandler := r.Get
	postHandler := r.Post
	patchHandler := r.Patch

	router.Group("orgs").GET("-/projects", getHandler)

	//router.GroupWithMeta("orgs", "root_domain").GET("", getHandler)
	router.Group("orgs").GETWithMeta("", "root_domain", getHandler)

	router.GroupWithMeta("orgs", "org_domain").
		GET(":id", getHandler).
		POST(":id", postHandler).
		PATCH(":id", postHandler).
		GET(":id/users", getHandler).
		GET(":id/projects", getHandler)

	router.GroupWithMeta("users", "user_domain").
		GET(":id", getHandler).
		PATCH(":id", postHandler).
		GET(":id/roles", getHandler)

	router.
		GroupWithMeta("resources", "first_level").
		GET(":id", getHandler). // ["first_level"]
		GroupWithMeta("sub-resources", "second_level").
		GETWithMeta("", "list", getHandler). // ["first_level", "second_level", "list"]
		GET(":id", getHandler).              // ["first_level", "second_level"]
		GroupWithMeta(":id/lowest-resources", "third_level").
		POSTWithMeta(":id", "create", postHandler) // ["first_level", "second_level","third_level", "create"]
	router.
		Group("resources").
		POST(":id", postHandler). // []
		GroupWithMeta("", "first_level").
		PATCHWithMeta(":id", "update", patchHandler) // ["first_level", "update"]
}

func (r MyResource) Middleware(ctx *gin.Context) {
	lib.GetLoggerFromCtx(ctx).Infof("middleware!!")
}

func (r MyResource) Get(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, lib.GetPathMetaFromCtx(ctx))
}

func (r MyResource) Post(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, lib.GetPathMetaFromCtx(ctx))
}
func (r MyResource) Patch(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, lib.GetPathMetaFromCtx(ctx))
}

func (r MyResource) Check(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, r.httpHandler.Root.MetaMapping())
}
