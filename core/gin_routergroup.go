package core

import (
	"github.com/gin-gonic/gin"
)

type HTTPMethod string
type URLPath string
type MetaMapping map[URLPath]map[HTTPMethod][]any

/*
GinRouterGroup is a drop-in replacement for *gin.RouterGroup, but it is not binary compatible, that also allows to
register metadata in the path with new function flavors WithMeta.

Metadata registration follows the handlers chain rules for registration. Example:

	router.
		GroupWithMeta("resources", "first_level").
		GET(":id", getHandler).                            // ["first_level"]
		GroupWithMeta("sub-resources", "second_level").
		GETWithMeta("", "list", getHandler).               // ["first_level", "second_level", "list"]
		GET(":id", getHandler).                            // ["first_level", "second_level"]
		GroupWithMeta("lowest-resource", "third_level").
		POSTWithMeta(":id", "create", postHandler)         // ["first_level", "second_level","third_level", "create"]
	router.
		Group("resources").
		POST(":id", postHandler).                          // []
		GroupWithMeta("", "first_level").
		PATCHWithMeta(":id", "update", patchHandler)       // ["first_level", "update"]

The output is:

	{
	  "GET": {
	    "/resources/subresource/:id": [
	      "first_level",
	      "second_level"
	      ]
	  }
	}
*/
type GinRouterGroup interface {
	MetaMapping() MetaMapping

	Use(handlers ...gin.HandlerFunc) GinRouterGroup

	GET(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup
	GETWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup

	POST(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup
	POSTWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup

	PUT(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup
	PUTWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup

	PATCH(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup
	PATCHWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup

	DELETE(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup
	DELETEWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup

	HEAD(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup
	HEADWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup

	OPTIONS(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup
	OPTIONSWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup

	Handle(httpMethod string, relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup
	HandleWithMeta(httpMethod string, relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup

	Group(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup
	GroupWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup
}

func NewGinRouterGroup(routerGroup *gin.RouterGroup) GinRouterGroup {
	g := &_GinRouterGroup{
		byPath: map[URLPath]map[HTTPMethod][]any{},
	}
	g.RouterGroup = routerGroup.Group("", g.middleware)
	return g
}

var routeMetadataCtxKey = CtxKey("_fw_route_metadata")

func GetPathMetaFromCtx(ctx *gin.Context) []any {
	if meta, exists := ctx.Get(routeMetadataCtxKey.(string)); exists {
		return meta.([]any)
	}
	return nil
}
