package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/rest/middleware"
	"github.com/southernlabs-io/go-fw/test"
)

func TestErrorHandler(t *testing.T) {
	conf := test.NewConfig(t.Name())
	lf := test.NewLoggerFactory(t, conf.RootConfig)
	ctx := test.NewContext(database.DB{}, lf)

	errMappers := []middleware.ErrorMapper{
		middleware.NewErrorCodeMapper(
			errors.ErrCodeBadArgument,
			func(err *gin.Error) (body any, status int, buildErr error) {
				return gin.H{
						"error": err.Error(),
					},
					http.StatusBadRequest,
					nil
			},
		),
		middleware.NewErrorCodeMapper(
			errors.ErrCodeBadState,
			func(err *gin.Error) (body any, status int, buildErr error) {
				return gin.H{
						"error": err.Error(),
					},
					http.StatusConflict,
					nil
			},
		),
		middleware.NewErrorAsMapper(
			validation.ErrorObject{},
			func(err *gin.Error) (body any, status int, buildErr error) {
				return gin.H{
						"error": err.Error(),
					},
					http.StatusNotAcceptable,
					nil
			},
		),
		middleware.NewErrorIsMapper(
			gorm.ErrInvalidData,
			func(err *gin.Error) (body any, status int, buildErr error) {
				return gin.H{
						"error": err.Error(),
					},
					http.StatusUnprocessableEntity,
					nil
			},
		),
	}

	errorHandler := middleware.NewErrorHandler(conf, lf, errMappers, nil)
	require.NotNil(t, errorHandler)

	setupGinTest := func() (*httptest.ResponseRecorder, *gin.Context) {
		w := httptest.NewRecorder()
		ginCtx, engine := gin.CreateTestContext(w)
		engine.ContextWithFallback = true
		ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		return w, ginCtx
	}

	// Test error code mapper
	w, ginCtx := setupGinTest()
	_ = ginCtx.Error(errors.Newf(errors.ErrCodeBadArgument, "bad argument"))
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusBadRequest, w.Code)

	// Test error code mapper
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(errors.Newf(errors.ErrCodeBadState, "bad state"))
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusConflict, w.Code)

	// Test default error mapper
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(errors.NewUnknownf("unknown error"))
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusInternalServerError, w.Code)

	// Test error mapper
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(validation.ErrInInvalid)
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusNotAcceptable, w.Code)

	// Test custom error mapper
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(gorm.ErrInvalidData)
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusUnprocessableEntity, w.Code)

	// Test default handler ErrCodeConflict
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(errors.Newf(errors.ErrCodeConflict, "conflict"))
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusConflict, w.Code)

	// Test default handler ErrCodeNotFound
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(errors.Newf(errors.ErrCodeNotFound, "not found"))
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusNotFound, w.Code)

	// Test default handler ErrCodeNotAllowed
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(errors.Newf(errors.ErrCodeNotAllowed, "not allowed"))
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusForbidden, w.Code)

	// Test default handler ErrCodeNotAuthenticated
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(errors.Newf(errors.ErrCodeNotAuthenticated, "not authenticated"))
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusUnauthorized, w.Code)

	// Test default handler ErrCodePanic
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(errors.Newf(errors.ErrCodePanic, "panic"))
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusInternalServerError, w.Code)

	// Test default handler ErrCodeUnknown
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(errors.Newf(errors.ErrCodeUnknown, "unknown"))
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusInternalServerError, w.Code)
	require.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "application/json"))
	body := w.Body.String()
	require.NotEmpty(t, body)
	require.True(t, strings.HasPrefix(body, "{\"error\":{"))
	require.Contains(t, body, "\"kind\":\"UNKNOWN\"")
	require.Contains(t, body, "\"message\":\"unknown\"")
	require.Contains(t, body, "\"stack\":\"")

	// Test default handler ErrCodeUnknown in Prod
	w, ginCtx = setupGinTest()
	_ = ginCtx.Error(errors.Newf(errors.ErrCodeUnknown, "unknown"))
	errorHandler.Conf.Env.Type = config.EnvTypeProd
	errorHandler.Run(ginCtx)
	ginCtx.Writer.Flush()
	require.EqualValues(t, http.StatusInternalServerError, w.Code)
	require.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "application/json"))
	body = w.Body.String()
	require.NotEmpty(t, body)
	require.True(t, strings.HasPrefix(body, "{\"error\":{"))
	require.Contains(t, body, "\"kind\":\"UNKNOWN\"")
	require.Contains(t, body, "\"message\":\"unknown\"")
	require.NotContains(t, body, "\"stack\":\"")
}
