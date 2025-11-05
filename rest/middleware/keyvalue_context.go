package middleware

import (
	"net/http"

	"github.com/southernlabs-io/go-fw/config"
	fw_context "github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/log"
)

type KeyValueContextMiddleware struct {
	BaseMiddleware
}

func NewKeyValueContextMiddleware(
	conf config.Config,
	lf log.LoggerFactory,
) *KeyValueContextMiddleware {
	return &KeyValueContextMiddleware{
		BaseMiddleware{conf, lf.GetLoggerForType(KeyValueContextMiddleware{})},
	}
}

var _ Middleware = (*KeyValueContextMiddleware)(nil)

func (m *KeyValueContextMiddleware) Priority() MiddlewarePriority {
	return -1 // Special value to make sure it runs first, no exceptions.
}

func (m *KeyValueContextMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := fw_context.NewContextWithStore(r.Context())
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
