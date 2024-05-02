package rest

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/mitchellh/mapstructure"

	lib "github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
)

// BindJSONBody binds the body to the dst parameter expecting the body to be JSON. It will abort with error
// http.StatusBadRequest if the JSON deserialization fails.
//
// It will run validations if the dst struct implements validation.Validatable interface. It will abort with
// http.StatusUnprocessableEntity if validation fails
func BindJSONBody(ctx *gin.Context, dst any) error {
	if err := ctx.ShouldBindJSON(dst); err != nil {
		_ = ctx.Error(
			errors.Newf(
				errors.ErrCodeBadArgument,
				"could not parse body in to: %T, binding error: %w",
				dst,
				err,
			),
		).SetType(gin.ErrorTypeBind)
		ctx.Abort()
		return err
	}

	if v, is := dst.(validation.Validatable); is {
		if err := v.Validate(); err != nil {
			_ = ctx.Error(errors.Newf(
				errors.ErrCodeBadArgument,
				"could not parse body into: %T, validation error: %w",
				dst,
				err,
			)).SetType(gin.ErrorTypeBind)
			ctx.Abort()
			return err
		}
	}

	return nil
}

// ParseQueryDeepObject will parse the url.URL.Query() expecting it to follow [openapi3 deepObject] serialization.
// [openapi3 deepObject]: https://swagger.io/docs/specification/serialization/#:~:text=b%7Cc.-,deepObject,-%E2%80%93%20simple%20non%2Dnested
func ParseQueryDeepObject(ctx *gin.Context) map[string]any {
	resp := map[string]any{}
	for key, values := range ctx.Request.URL.Query() {
		if i := strings.IndexByte(key, '['); i > 0 {
			mapKey := key[0:i]
			resp[mapKey] = ctx.QueryMap(mapKey)
		} else {
			if len(values) > 1 {
				resp[key] = values
			} else {
				resp[key] = values[0]
			}
		}
	}

	return resp
}

// BindDeepObjectQuery binds the url.URL.Query() to the dst parameter expecting the query to follow [openapi3 deepObject]
// serialization. The binding will be done using mapstructure.WeakDecode. It will abort with error http.StatusBadRequest
// if the binding fails.
//
// It will run validations if the dst struct implements validation.Validatable interface. It will abort with
// error http.StatusUnprocessableEntity if validation fails.
//
// [openapi3 deepObject]: https://swagger.io/docs/specification/serialization/#:~:text=b%7Cc.-,deepObject,-%E2%80%93%20simple%20non%2Dnested
func BindDeepObjectQuery(ctx *gin.Context, dst any) error {
	values := ParseQueryDeepObject(ctx)
	err := mapstructure.WeakDecode(values, dst)
	if err != nil {
		_ = ctx.AbortWithError(
			http.StatusBadRequest,
			errors.Newf(
				errors.ErrCodeBadArgument,
				"could not parse query parameters into: %T, validation error: %w",
				dst,
				err,
			),
		)
		return err
	}

	if v, is := dst.(validation.Validatable); is {
		if err = v.Validate(); err != nil {
			_ = ctx.Error(errors.Newf(
				errors.ErrCodeBadArgument,
				"could not validate query parameters for: %T, validation error: %w",
				dst,
				err,
			),
			).SetType(gin.ErrorTypeBind)
			ctx.Abort()
			return err
		}
	}

	return nil
}

func HandleError(ctx *gin.Context, conf lib.Config, err error, nonFWFormat string, args ...any) {
	var fwErr *errors.Error
	if errors.As(err, &fwErr) {
		_ = ctx.Error(err)
	} else {
		args = append(args, err)
		ginErr := ctx.Error(errors.NewUnknownf(nonFWFormat+": %w", args...))
		// Make error public on non prod envs
		if conf.Env.Type != lib.EnvTypeProd {
			_ = ginErr.SetType(gin.ErrorTypePublic)
		}
	}
	ctx.Abort()
}
