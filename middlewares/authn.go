//go:generate mockery --all --with-expecter=true --keeptree=false --case=underscore
package middlewares

import (
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/syncmap"
)

var ErrInvalidToken = errors.Newf("AUTHN_TOKEN_NOT_VALID", "token is not valid")

const PrincipalCtxKey = "authn_principal"
const AuthNExcludedCtxKey = "authn_excluded"

type PrincipalType string

type Principal interface {
	GetID() any
	GetName() string
	GetEmail() string
	GetType() PrincipalType
}

func GetPrincipal(ctx *gin.Context) (principal Principal, present bool) {
	if value, exists := ctx.Get(PrincipalCtxKey); exists {
		return value.(Principal), true
	}
	return
}
func MustGetPrincipal(ctx *gin.Context) Principal {
	if principal, present := GetPrincipal(ctx); present {
		return principal
	}

	ctx.AbortWithStatus(http.StatusUnauthorized)
	panic(errors.Newf(errors.ErrCodeBadState, "no principal in context"))
}

func SetPrincipal(ctx *gin.Context, principal Principal) {
	ctx.Set(PrincipalCtxKey, principal)
	attrs := core.GetLoggerAttrsFromCtx(ctx)
	principalAttr := slog.Group("usr",
		slog.Any("id", principal.GetID()),
		slog.Any("type", principal.GetType()),
	)
	for idx, attr := range attrs {
		if attr.Key == principalAttr.Key {
			attrs[idx] = principalAttr
			return
		}
	}
	core.CtxAppendLoggerAttrs(ctx, principalAttr)
}

type AuthNProvider interface {
	Authenticate(ctx *gin.Context) (Principal, error)
}

type _PathMethod struct {
	path   string
	method string
}

// AuthNMiddleware The exclusions are implemented as a map from path prefix to list of methods. If the list of methods
// is empty, then all methods are excluded for the path prefix acting as map key
type AuthNMiddleware struct {
	BaseMiddleware
	provider      AuthNProvider
	excludes      map[string][]string
	excludesCache *syncmap.Map[_PathMethod, bool]
}

func NewAuthN(
	conf core.Config,
	lf *core.LoggerFactory,
	provider AuthNProvider,
) *AuthNMiddleware {
	return &AuthNMiddleware{
		BaseMiddleware{
			conf, lf.GetLoggerForType(AuthNMiddleware{}),
		},
		provider,
		map[string][]string{},
		syncmap.New[_PathMethod, bool](),
	}
}

func (m *AuthNMiddleware) Setup(httpHandler core.HTTPHandler) {
	httpHandler.Root.Use(m.Require)
}

func (m *AuthNMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityAuthN
}

// ExcludePrefix excludes the given path prefix from authentication for all methods. Internally, this method attempts to
// add the empty list to the exclusion map. If the path prefix was already excluded for specific methods, then the function
// will make the list nil to exclude all methods.
func (m *AuthNMiddleware) ExcludePrefix(pathPrefix string) {
	m.excludes[pathPrefix] = nil
}

// ExcludePrefixAndMethods excludes the given path prefix from authentication for the given methods. There must be at least
// one given method. Internally, this method attempts to append the given methods to the methods list already present in the
// exclusion map for the given path prefix. If the already present method list is of length zero, it means the path prefix
// was already excluded for all methods, and the function doesn't do anything further.
func (m *AuthNMiddleware) ExcludePrefixAndMethods(pathPrefix string, methods ...string) {
	if len(methods) == 0 {
		panic(errors.Newf(errors.ErrCodeBadArgument, "path: %s, methods list must have at least one element", pathPrefix))
	}
	previousMethods, found := m.excludes[pathPrefix]
	if found && len(previousMethods) == 0 {
		return
	}
	m.excludes[pathPrefix] = append(previousMethods, methods...)
}

func (m *AuthNMiddleware) Require(ctx *gin.Context) {
	excluded := m.excludesCache.LoadOrStore(_PathMethod{path: ctx.FullPath(), method: ctx.Request.Method}, func(pathMethod _PathMethod) bool {
		// exclude paths that are not under the base path, like /health
		if !strings.HasPrefix(pathMethod.path, m.Conf.HttpServer.BasePath) {
			core.GetLoggerFromCtx(ctx).Infof("Excluded path: %s", pathMethod.path)
			return true
		}
		for excludePath, methods := range m.excludes {
			if strings.HasPrefix(pathMethod.path, path.Join(m.Conf.HttpServer.BasePath, excludePath)) &&
				(len(methods) == 0 || slices.Contains(methods, pathMethod.method)) {
				core.GetLoggerFromCtx(ctx).Infof("Excluded path: %s for method: %s", pathMethod.path, pathMethod.method)
				return true
			}
		}

		return false
	})

	if excluded {
		ctx.Set(AuthNExcludedCtxKey, true)
		return
	}
	principal, err := m.provider.Authenticate(ctx)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			ctx.AbortWithStatus(http.StatusUnauthorized)
		} else {
			core.GetLoggerFromCtx(ctx).Errorf("Failed to authenticate, error: %s", err)
			ctx.AbortWithStatus(http.StatusInternalServerError)
		}
	} else {
		SetPrincipal(ctx, principal)
	}
}

var AuthNModule = ProvideAsMiddleware(NewAuthN)
