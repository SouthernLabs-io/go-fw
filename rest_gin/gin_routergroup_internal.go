package rest

import (
	"net/http"
	"path"
	"slices"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/maps"
)

type _GinRouterGroup struct {
	*gin.RouterGroup
	byPath        MetaMapping
	groupMetadata []any
}

var _ GinRouterGroup = new(_GinRouterGroup)

func (g *_GinRouterGroup) middleware(ctx *gin.Context) {
	ctx.Set(routeMetadataCtxKey.(string), g.GetMetaByFullPath(URLPath(ctx.FullPath()), HTTPMethod(ctx.Request.Method)))
}

func (g *_GinRouterGroup) addMeta(relativePath string, httpMethod HTTPMethod, metadata any) {
	fullPath := URLPath(path.Join(g.BasePath(), relativePath))

	pathMapping := g.byPath[fullPath]
	if pathMapping == nil {
		pathMapping = map[HTTPMethod][]any{}
		g.byPath[fullPath] = pathMapping
	}

	// Group metadata first
	if pathMapping[httpMethod] == nil {
		pathMapping[httpMethod] = append([]any{}, g.groupMetadata...)
	}

	if metadata != nil {
		// Append new metadata
		pathMapping[httpMethod] = append(pathMapping[httpMethod], metadata)
	}
}

func (g *_GinRouterGroup) GetMetaByFullPath(fullPath URLPath, httpMethod HTTPMethod) []any {
	return g.byPath[fullPath][httpMethod]
}

func (g *_GinRouterGroup) MetaMapping() MetaMapping {
	return maps.Clone(g.byPath)
}

func (g *_GinRouterGroup) Use(handlers ...gin.HandlerFunc) GinRouterGroup {
	g.RouterGroup.Use(handlers...)
	return g
}
func (g *_GinRouterGroup) Handle(httpMethod string, relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup {
	g.addMeta(relativePath, HTTPMethod(httpMethod), nil)
	g.RouterGroup.Handle(httpMethod, relativePath, handlers...)
	return g
}

func (g *_GinRouterGroup) HandleWithMeta(httpMethod string, relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup {
	g.addMeta(relativePath, HTTPMethod(httpMethod), metadata)
	g.RouterGroup.Handle(httpMethod, relativePath, handlers...)
	return g
}

func (g *_GinRouterGroup) Match(methods []string, relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup {
	g.RouterGroup.Match(methods, relativePath, handlers...)
	return g
}

func (g *_GinRouterGroup) MatchWithMeta(methods []string, relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup {
	for _, method := range methods {
		return g.HandleWithMeta(method, relativePath, metadata, handlers...)
	}
	return g
}

func (g *_GinRouterGroup) Group(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.GroupWithMeta(relativePath, nil, handlers...)
}

func (g *_GinRouterGroup) GroupWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup {
	newG := &_GinRouterGroup{
		RouterGroup:   g.RouterGroup.Group(relativePath, handlers...),
		byPath:        g.byPath,
		groupMetadata: slices.Clone(g.groupMetadata),
	}
	if metadata != nil {
		newG.groupMetadata = append(newG.groupMetadata, metadata)
	}

	return newG
}

// Sugar funcs

func (g *_GinRouterGroup) GETWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.HandleWithMeta(http.MethodGet, relativePath, metadata, handlers...)
}
func (g *_GinRouterGroup) POSTWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.HandleWithMeta(http.MethodPost, relativePath, metadata, handlers...)
}

func (g *_GinRouterGroup) PUTWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.HandleWithMeta(http.MethodPut, relativePath, metadata, handlers...)
}
func (g *_GinRouterGroup) PATCHWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.HandleWithMeta(http.MethodPatch, relativePath, metadata, handlers...)
}
func (g *_GinRouterGroup) DELETEWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.HandleWithMeta(http.MethodDelete, relativePath, metadata, handlers...)
}
func (g *_GinRouterGroup) OPTIONSWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.HandleWithMeta(http.MethodOptions, relativePath, metadata, handlers...)
}

func (g *_GinRouterGroup) HEADWithMeta(relativePath string, metadata any, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.HandleWithMeta(http.MethodHead, relativePath, metadata, handlers...)
}

// GET is a shortcut for router.Handle("GET", path, handlers).
func (g *_GinRouterGroup) GET(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.Handle(http.MethodGet, relativePath, handlers...)
}

func (g *_GinRouterGroup) POST(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.Handle(http.MethodPost, relativePath, handlers...)
}

func (g *_GinRouterGroup) PUT(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.Handle(http.MethodPut, relativePath, handlers...)
}

func (g *_GinRouterGroup) PATCH(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.Handle(http.MethodPatch, relativePath, handlers...)
}

func (g *_GinRouterGroup) DELETE(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.Handle(http.MethodDelete, relativePath, handlers...)
}

func (g *_GinRouterGroup) OPTIONS(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.Handle(http.MethodOptions, relativePath, handlers...)
}

func (g *_GinRouterGroup) HEAD(relativePath string, handlers ...gin.HandlerFunc) GinRouterGroup {
	return g.Handle(http.MethodHead, relativePath, handlers...)
}
