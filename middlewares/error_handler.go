package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
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
	core.BaseParams
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
	conf core.Config,
	lf *core.LoggerFactory,
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

func (m *ErrorHandlerMiddleware) Setup(httpHandler core.HTTPHandler) {
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
	logger := core.GetLoggerFromCtx(ctx)
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
				defaultHandler(ctx, errBody, shouldWrite)
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
		defaultHandler(ctx, nil, shouldWrite)
		mappedErr = otherErrs[0]
		otherErrs = otherErrs[1:]
	}

	status := ctx.Writer.Status()
	level := core.LogLevelInfo
	if status >= 500 {
		level = core.LogLevelError
	} else if status >= 400 {
		level = core.LogLevelWarn
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

func defaultHandler(ctx *gin.Context, body any, shouldWrite bool) int {
	var valErrs validation.Errors
	ginErr := ctx.Errors[0]
	var status int
	if ginErr != nil && ginErr.Type == gin.ErrorTypeBind && errors.As(ginErr.Err, &valErrs) {
		status = http.StatusUnprocessableEntity
	} else {
		status = http.StatusInternalServerError
	}
	if shouldWrite {
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
