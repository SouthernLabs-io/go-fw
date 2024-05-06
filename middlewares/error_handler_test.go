package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/middlewares"
	"github.com/southernlabs-io/go-fw/test"
)

func TestErrorHandler(t *testing.T) {
	conf := test.NewConfig(t.Name())
	lf := test.NewLoggerFactory(t, conf.RootConfig)
	ctx := test.NewContext(database.DB{}, lf)

	errMappers := []middlewares.ErrorMapper{
		middlewares.NewErrorCodeMapper(
			errors.ErrCodeBadArgument,
			func(err *gin.Error) (body any, status int, buildErr error) {
				return gin.H{
						"error": err.Error(),
					},
					http.StatusBadRequest,
					nil
			},
		),
		middlewares.NewErrorCodeMapper(
			errors.ErrCodeBadState,
			func(err *gin.Error) (body any, status int, buildErr error) {
				return gin.H{
						"error": err.Error(),
					},
					http.StatusConflict,
					nil
			},
		),
		middlewares.NewErrorAsMapper(
			validation.ErrorObject{},
			func(err *gin.Error) (body any, status int, buildErr error) {
				return gin.H{
						"error": err.Error(),
					},
					http.StatusNotAcceptable,
					nil
			},
		),
		middlewares.NewErrorIsMapper(
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

	errorHandler := middlewares.NewErrorHandler(conf, lf, errMappers, nil)
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
}
