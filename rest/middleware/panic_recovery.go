package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

type PanicRecoveryMiddleware struct {
	BaseMiddleware
}

func NewPanicRecovery(
	conf config.Config,
	lf log.LoggerFactory,
) *PanicRecoveryMiddleware {
	return &PanicRecoveryMiddleware{
		BaseMiddleware{conf, lf.GetLoggerForType(PanicRecoveryMiddleware{})},
	}
}

var _ Middleware = (*PanicRecoveryMiddleware)(nil)

func (m *PanicRecoveryMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityHighest + 1
}

func (m *PanicRecoveryMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if errAny := recover(); errAny != nil {
				handlePanic(w, r, errAny, true)
			}
		}()
		next.ServeHTTP(w, r)
	})

}

func handlePanic(w http.ResponseWriter, r *http.Request, errAny any, logError bool) bool {
	// Check for a broken connection, as it is not really a
	// condition that warrants a panic stack trace.
	ctx := r.Context()
	var brokenPipe bool
	var recoveryErr error
	if err, is := errAny.(error); is {
		var se *os.SyscallError
		if errors.As(err, &se) {
			if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
				brokenPipe = true
			}
		}
		recoveryErr = errors.Newf(errors.ErrCodePanic, "recovery from panic: %w", err)
	} else {
		recoveryErr = errors.Newf(errors.ErrCodePanic, "recovery from panic: %v", errAny)
	}

	if logError {
		log.GetLoggerFromCtx(ctx).ErrorE(recoveryErr)
	}

	if brokenPipe {
		// If the connection is dead, we can't write a status to it.
		log.GetLoggerFromCtx(ctx).Warnf("Broken pipe")
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	return brokenPipe
}
