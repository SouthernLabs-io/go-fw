package middlewares

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/rest"
)

type PanicRecoveryMiddleware struct {
	BaseMiddleware
}

func NewPanicRecovery(
	conf core.Config,
	lf *core.LoggerFactory,
) *PanicRecoveryMiddleware {
	return &PanicRecoveryMiddleware{
		BaseMiddleware{conf, lf.GetLoggerForType(PanicRecoveryMiddleware{})},
	}
}

func (m *PanicRecoveryMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityHighest + 1
}

func (m *PanicRecoveryMiddleware) Setup(httpHandler rest.HTTPHandler) {
	httpHandler.Root.Use(m.Run)
}

func (m *PanicRecoveryMiddleware) Run(ctx *gin.Context) {
	defer func() {
		if errAny := recover(); errAny != nil {
			handlePanic(ctx, errAny, true)
		}
	}()
	ctx.Next()
}

func handlePanic(ctx *gin.Context, errAny any, logError bool) bool {
	// Check for a broken connection, as it is not really a
	// condition that warrants a panic stack trace.
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
		core.GetLoggerFromCtx(ctx).ErrorE(recoveryErr)
	}

	if brokenPipe {
		// If the connection is dead, we can't write a status to it.
		_ = ctx.Error(recoveryErr)
		ctx.Abort()
	} else {
		_ = ctx.AbortWithError(http.StatusInternalServerError, recoveryErr)
	}
	return brokenPipe
}
