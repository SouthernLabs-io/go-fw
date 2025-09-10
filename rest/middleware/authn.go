package middleware

import (
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/sync"
)

var ErrInvalidToken = errors.Newf("AUTHN_TOKEN_NOT_VALID", "token is not valid")

var PrincipalCtxKey = context.CtxKey("_fw_authn_principal")
var AuthNExcludedCtxKey = context.CtxKey("authn_excluded")

type PrincipalType string

type Principal interface {
	GetID() any
	GetName() string
	GetEmail() string
	GetType() PrincipalType
}

func GetPrincipal(ctx context.Context) (principal Principal, present bool) {
	if value, exists := ctx.Value(PrincipalCtxKey).(Principal); exists {
		return value, true
	}
	return
}
func MustGetPrincipal(ctx context.Context) Principal {
	if principal, present := GetPrincipal(ctx); present {
		return principal
	}
	panic(errors.Newf(errors.ErrCodeNotAuthenticated, "no principal"))
}

func SetPrincipal(ctx context.Context, principal Principal) context.Context {
	ctx = context.WithValue(ctx, PrincipalCtxKey, principal)
	attrs := log.GetLoggerAttrsFromCtx(ctx)
	principalAttr := slog.Group("usr",
		slog.Any("id", principal.GetID()),
		slog.Any("type", principal.GetType()),
	)
	for idx, attr := range attrs {
		if attr.Key == principalAttr.Key {
			attrs[idx] = principalAttr
			return ctx
		}
	}
	ctx = log.CtxAppendLoggerAttrs(ctx, principalAttr)
	return ctx
}

type AuthNProvider interface {
	Authenticate(ctx context.Context, r *http.Request) (Principal, error)
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
	excludesCache *sync.Map[_PathMethod, bool]
}

func NewAuthN(
	conf config.Config,
	lf log.LoggerFactory,
	provider AuthNProvider,
) *AuthNMiddleware {
	return &AuthNMiddleware{
		BaseMiddleware{
			conf, lf.GetLoggerForType(AuthNMiddleware{}),
		},
		provider,
		map[string][]string{},
		sync.NewMap[_PathMethod, bool](),
	}
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

func (m *AuthNMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		excluded := m.excludesCache.LoadOrStoreFunc(_PathMethod{path: r.URL.Path, method: r.Method}, func(pathMethod _PathMethod) bool {
			// exclude paths that are not under the base path, like /health
			if !strings.HasPrefix(pathMethod.path, m.Conf.HttpServer.BasePath) {
				log.GetLoggerFromCtx(ctx).Infof("Excluded path: %s", pathMethod.path)
				return true
			}
			for excludePath, methods := range m.excludes {
				if strings.HasPrefix(pathMethod.path, path.Join(m.Conf.HttpServer.BasePath, excludePath)) &&
					(len(methods) == 0 || slices.Contains(methods, pathMethod.method)) {
					log.GetLoggerFromCtx(ctx).Infof("Excluded path: %s for method: %s", pathMethod.path, pathMethod.method)
					return true
				}
			}

			return false
		})

		if excluded {
			ctx = context.WithValue(ctx, AuthNExcludedCtxKey, true)
			if ctx != r.Context() {
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
			return
		}

		principal, err := m.provider.Authenticate(ctx, r)
		if err != nil {
			log.GetLoggerFromCtx(ctx).Errorf("failed to authenticate, error: %s", err)
			if errors.Is(err, ErrInvalidToken) {
				panic(errors.Newf(errors.ErrCodeNotAuthenticated, "no principal"))
			} else {
				panic(errors.Newf(errors.ErrCodeBadState, "failed to authenticate"))
			}
		} else {
			ctx = SetPrincipal(ctx, principal)
			log.GetLoggerFromCtx(ctx).Debugf("Authenticated principal: %s", principal.GetID())
			if ctx != r.Context() {
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		}
	})
}

var AuthNModule = ProvideAsMiddleware(NewAuthN)
