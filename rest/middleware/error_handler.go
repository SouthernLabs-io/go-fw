package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest"
)

type ErrorResponseBuilder = func(*gin.Error) (body any, status int, buildErr error)

type ErrorMapper struct {
	check           func(err error) bool
	responseBuilder ErrorResponseBuilder
}

func NewErrorAsMapper[E error](mapErr E, builder ErrorResponseBuilder) ErrorMapper {
	return ErrorMapper{
		check:           func(err error) bool { return errors.As(err, &mapErr) },
		responseBuilder: builder,
	}
}

func NewErrorIsMapper(mapErr error, builder ErrorResponseBuilder) ErrorMapper {
	return ErrorMapper{
		check:           func(err error) bool { return errors.Is(err, mapErr) },
		responseBuilder: builder,
	}
}

func NewErrorCodeMapper(errCode string, builder ErrorResponseBuilder) ErrorMapper {
	return ErrorMapper{
		check:           func(err error) bool { return errors.IsCode(err, errCode) },
		responseBuilder: builder,
	}
}

type ErrorHandlerMiddleware struct {
	BaseMiddleware
	errMappers []ErrorMapper
}
type ErrorHandlerMiddlewareParams struct {
	di.BaseParams
	ErrorMappers       []ErrorMapper        `group:"error_mappers"`
	DefaultErrorMapper ErrorResponseBuilder `name:"default_error_mapper" optional:"true"`
}

func NewErrorHandlerFx(params ErrorHandlerMiddlewareParams) *ErrorHandlerMiddleware {
	return NewErrorHandler(
		params.Conf,
		params.LF,
		params.ErrorMappers,
		params.DefaultErrorMapper,
	)
}

func NewErrorHandler(
	conf config.Config,
	lf *log.LoggerFactory,
	errMappings []ErrorMapper,
	defaultErrorMapper ErrorResponseBuilder,
) *ErrorHandlerMiddleware {
	effectiveErrMappings := slices.Clone(errMappings)
	if defaultErrorMapper != nil {
		effectiveErrMappings = append(effectiveErrMappings, ErrorMapper{
			check:           func(err error) bool { return true },
			responseBuilder: defaultErrorMapper,
		})
	}
	return &ErrorHandlerMiddleware{
		BaseMiddleware{conf, lf.GetLoggerForType(ErrorHandlerMiddleware{})},
		effectiveErrMappings,
	}
}

func (m *ErrorHandlerMiddleware) Setup(httpHandler rest.HTTPHandler) {
	httpHandler.Root.Use(m.Run)
}

func (m *ErrorHandlerMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityBody
}

func (m *ErrorHandlerMiddleware) Run(ctx *gin.Context) {
	defer func() {
		var brokenPipe bool
		if errAny := recover(); errAny != nil {
			brokenPipe = handlePanic(ctx, errAny, false)
		}
		m.handleErrors(ctx, brokenPipe)
	}()

	ctx.Next()
}

func (m *ErrorHandlerMiddleware) handleErrors(ctx *gin.Context, brokenPipe bool) {
	if len(ctx.Errors) == 0 {
		return
	}

	shouldWrite := !brokenPipe && !ctx.Writer.Written()
	var mappedErr error
	var otherErrs []error
	logger := log.GetLoggerFromCtx(ctx)
	for _, ginErr := range ctx.Errors {
		if ginErr == nil {
			continue
		}

		// We already mapped an error. We just log the others.
		if mappedErr != nil {
			otherErrs = append(otherErrs, ginErr.Err)
			continue
		}
		// Error mappers
		for _, errMapper := range m.errMappers {
			if !errMapper.check(ginErr.Err) {
				continue
			}

			errBody, status, buildErr := errMapper.responseBuilder(ginErr)
			if buildErr != nil {
				// Something wrong happened. We stop trying to build a custom response for this request
				logger.Warnf("Unable to build response body for mapped Gin error %s: %s", ginErr, buildErr)
				mappedErr = ginErr.Err
				break
			}
			if errBody == nil && status == 0 {
				// Builder doesn't specify anything. Keep searching for other mappers and other errors
				continue
			}
			if status == 0 {
				defaultHandler(ctx, m.Conf.Env.Type, errBody, shouldWrite)
			} else if shouldWrite {
				if errBody != nil {
					ctx.JSON(status, errBody)
				} else {
					ctx.Status(status)
				}
			}
			mappedErr = ginErr.Err
			break
		}

		// If no error mapper matched, then add to the list
		if mappedErr == nil {
			otherErrs = append(otherErrs, ginErr.Err)
		}
	}

	// Check if there was a mapped error. If not, we use the first error
	if mappedErr == nil {
		defaultHandler(ctx, m.Conf.Env.Type, nil, shouldWrite)
		mappedErr = otherErrs[0]
		otherErrs = otherErrs[1:]
	}

	status := ctx.Writer.Status()
	level := config.LogLevelInfo
	if status >= 500 {
		level = config.LogLevelError
	} else if status >= 400 {
		level = config.LogLevelWarn
	}

	if len(otherErrs) == 0 {
		logger.LogAttrs(
			level,
			fmt.Sprintf("error: %s", mappedErr),
			slog.Any("error", mappedErr),
		)
	} else {
		logger.LogAttrs(
			level,
			fmt.Sprintf("error: %s", mappedErr),
			slog.Any("error", mappedErr),
			slog.Any("other_errors", otherErrs),
		)
	}
}

var errorCodesToStatus = map[string]int{
	errors.ErrCodeUnknown:  http.StatusInternalServerError,
	errors.ErrCodeBadState: http.StatusInternalServerError,
	errors.ErrCodePanic:    http.StatusInternalServerError,

	errors.ErrCodeNotAuthenticated: http.StatusUnauthorized,
	errors.ErrCodeNotAllowed:       http.StatusForbidden,
	errors.ErrCodeNotFound:         http.StatusNotFound,
	errors.ErrCodeConflict:         http.StatusConflict,
	errors.ErrCodeBadArgument:      http.StatusUnprocessableEntity,
	errors.ErrCodeValidationFailed: http.StatusUnprocessableEntity,
}

func defaultHandler(ctx *gin.Context, envType config.EnvType, body any, shouldWrite bool) int {
	ginErr := ctx.Errors[0]
	var status int
	var fwErr *errors.Error
	switch {
	case ginErr == nil:
		status = http.StatusInternalServerError
	case ginErr.IsType(gin.ErrorTypeBind):
		status = http.StatusUnprocessableEntity
	case errors.As(ginErr.Err, &fwErr):
		status = errorCodesToStatus[fwErr.Code]
		if status == 0 {
			status = http.StatusInternalServerError
		}
	}

	if shouldWrite {
		if body == nil && fwErr != nil {
			type ErrWrapper struct {
				Error any `json:"error"`
			}
			if envType == config.EnvTypeProd {
				// Use a copy, so we don't affect other uses of this error
				fwErr = fwErr.Copy()
				fwErr.SetStackKey("")
			}
			body = ErrWrapper{Error: fwErr}
		}
		if body != nil {
			ctx.JSON(status, body)
		} else {
			ctx.Status(status)
		}
	}
	return status
}

func RegisterErrorAsMapper(errAs error, builder ErrorResponseBuilder) fx.Option {
	return fx.Supply(AsErrorMapper(NewErrorAsMapper(errAs, builder)))
}

func RegisterErrorIsMapper(errIs error, builder ErrorResponseBuilder) fx.Option {
	return fx.Supply(AsErrorMapper(NewErrorIsMapper(errIs, builder)))
}

func RegisterErrorCodeMapper(errCode string, builder ErrorResponseBuilder) fx.Option {
	return fx.Supply(AsErrorMapper(NewErrorCodeMapper(errCode, builder)))
}

func RegisterDefaultErrorMapper(builder ErrorResponseBuilder) fx.Option {
	return fx.Supply(fx.Annotate(builder, fx.ResultTags(`name:"default_error_mapper"`)))
}

func AsErrorMapper(errMapper ErrorMapper) any {
	return fx.Annotate(
		errMapper,
		fx.ResultTags(`group:"error_mappers"`),
	)
}
