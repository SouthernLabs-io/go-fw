package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest"
)

type AuthZProvider interface {
	Authorize(ctx *gin.Context, args ...any) (bool, error)
}

type AuthZMiddleware struct {
	BaseMiddleware
	provider AuthZProvider
}

var _ Middleware = new(AuthZMiddleware)

func NewAuthZ(
	conf config.Config,
	lf *log.LoggerFactory,
	provider AuthZProvider,
) *AuthZMiddleware {
	return &AuthZMiddleware{
		BaseMiddleware{conf, lf.GetLoggerForType(AuthZMiddleware{})},
		provider,
	}
}

func (m *AuthZMiddleware) Setup(rest.HTTPHandler) {
}

func (m *AuthZMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityAuthZ
}

func (m *AuthZMiddleware) Require(args ...any) gin.HandlerFunc {
	return m.RequireCustom(m.provider, args...)
}

func (m *AuthZMiddleware) RequireCustom(provider AuthZProvider, args ...any) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		handler(ctx, provider, args...)
	}
}

func handler(ctx *gin.Context, provider AuthZProvider, args ...any) {
	if ctx.GetBool(AuthNExcludedCtxKey) {
		return
	}

	authorized, err := provider.Authorize(ctx, args)
	if err != nil {
		_ = ctx.Error(errors.NewUnknownf("failed to authorize: %w", err))
		ctx.Abort()
		return
	}
	if !authorized {
		ctx.AbortWithStatus(http.StatusForbidden)
	}
}

var AuthZModule = ProvideAsMiddleware(NewAuthZ)
