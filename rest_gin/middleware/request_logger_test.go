package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/southernlabs-io/go-fw/config"
	fwcontext "github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest_gin/middleware"
)

func TestRequestLoggerMiddleware_SecureRequestMetadataByDefault(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	engine := newRequestLoggerEngine(conf, &output, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/upload?token=request-secret&secret-value=unknown-secret&safe=visible", nil)
	req.Header.Set("Referer", "https://example.com/editor?token=referer-secret&view=graph")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, req)

	requestLog := requestEndLog(t, output.String())
	httpAttrs := requestLog["http"].(map[string]any)
	require.Equal(t, "/upload", httpAttrs["url"])
	require.NotContains(t, httpAttrs, "referer")

	urlDetails := httpAttrs["url_details"].(map[string]any)
	require.Equal(t, "/upload", urlDetails["path"])
	require.NotContains(t, urlDetails, "queryString")
	require.NotContains(t, output.String(), "request-secret")
	require.NotContains(t, output.String(), "unknown-secret")
	require.NotContains(t, output.String(), "referer-secret")
}

func TestRequestLoggerMiddleware_AllowsConfiguredQueryParameters(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	conf.HttpServer.RequestLogger.QueryParameterAllowlist = []string{"view"}
	engine := newRequestLoggerEngine(conf, &output, http.StatusCreated)

	req := httptest.NewRequest(http.MethodGet, "/upload?view=graph&token=request-secret&view=table", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, req)

	requestLog := requestEndLog(t, output.String())
	httpAttrs := requestLog["http"].(map[string]any)
	urlDetails := httpAttrs["url_details"].(map[string]any)
	query := urlDetails["queryString"].(map[string]any)
	require.Equal(t, []any{"graph", "table"}, query["view"])
	require.NotContains(t, query, "token")
	require.Equal(t, "/upload", httpAttrs["url"])
	require.Equal(t, float64(http.StatusCreated), requestLog["http.status_code"])
	require.Equal(t, "/upload", requestLog["http.url_details.pattern"])
}

func TestRequestLoggerMiddleware_MalformedQueryIsOmitted(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	conf.HttpServer.RequestLogger.QueryParameterAllowlist = []string{"view"}
	engine := newRequestLoggerEngine(conf, &output, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/upload?view=%ZZ&secret-value=unknown-secret", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, req)

	requestLog := requestEndLog(t, output.String())
	httpAttrs := requestLog["http"].(map[string]any)
	urlDetails := httpAttrs["url_details"].(map[string]any)
	require.NotContains(t, urlDetails, "queryString")
	require.NotContains(t, output.String(), "unknown-secret")
}

func TestRequestLoggerMiddleware_SanitizesConfiguredReferrer(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	conf.HttpServer.RequestLogger.ReferrerMode = config.RequestLoggerReferrerOriginPath
	engine := newRequestLoggerEngine(conf, &output, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/upload", nil)
	req.Header.Set("Referer", "https://example.com/editor?token=referer-secret#details")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, req)

	requestLog := requestEndLog(t, output.String())
	httpAttrs := requestLog["http"].(map[string]any)
	require.Equal(t, "https://example.com/editor", httpAttrs["referer"])
	require.NotContains(t, output.String(), "referer-secret")
}

func TestRequestLoggerMiddleware_MalformedReferrerIsOmitted(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	conf.HttpServer.RequestLogger.ReferrerMode = config.RequestLoggerReferrerOriginPath
	engine := newRequestLoggerEngine(conf, &output, http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/upload", nil)
	req.Header.Set("Referer", "https://example.com/%ZZ?token=referer-secret")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, req)

	requestLog := requestEndLog(t, output.String())
	httpAttrs := requestLog["http"].(map[string]any)
	require.NotContains(t, httpAttrs, "referer")
	require.NotContains(t, output.String(), "referer-secret")
}

func TestRequestLoggerMiddleware_PreservesRequestIDTracingRoutePatternAndStatus(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	conf.Datadog.Tracing = true
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, &output)
	mw := middleware.NewRequestLogger(conf, lf)

	mt := mocktracer.Start()
	t.Cleanup(mt.Stop)
	span := tracer.StartSpan("request")
	spanCtx := tracer.ContextWithSpan(fwcontext.Background(), span)
	t.Cleanup(func() { span.Finish() })

	calledWithRequestID := ""
	engine := gin.New()
	engine.ContextWithFallback = true
	engine.Use(mw.Run)
	engine.GET("/items/:id", func(ctx *gin.Context) {
		value, exists := ctx.Get(fwcontext.RequestIDCtxKey.(string))
		require.True(t, exists)
		calledWithRequestID, exists = value.(string)
		require.True(t, exists)
		ctx.Status(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodGet, "/items/42", nil).WithContext(spanCtx)
	req.Header.Set("Request-ID", "request-id-123")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Equal(t, "request-id-123", calledWithRequestID)

	requestLog := requestEndLogForPath(t, output.String(), "/items/42")
	require.Equal(t, float64(http.StatusAccepted), requestLog["http.status_code"])
	require.Equal(t, "/items/:id", requestLog["http.url_details.pattern"])
	require.Equal(t, strconv.FormatUint(span.Context().TraceID(), 10), requestLogNumber(t, requestLog, "dd.trace_id"))
	require.Equal(t, strconv.FormatUint(span.Context().SpanID(), 10), requestLogNumber(t, requestLog, "dd.span_id"))
}

func TestRequestLoggerMiddleware_ExcludedRouteSkipsDownstreamAndRequestLog(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	conf.HttpServer.ReqLoggerExcludes = []string{"/health"}
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, &output)
	mw := middleware.NewRequestLogger(conf, lf)

	handlerCalled := false
	engine := gin.New()
	engine.ContextWithFallback = true
	engine.Use(mw.Run)
	engine.GET("/health", func(ctx *gin.Context) {
		handlerCalled = true
		ctx.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	require.True(t, handlerCalled)
	require.Equal(t, http.StatusNoContent, response.Code)
	require.NotContains(t, output.String(), "Req End: /health")
}

func newRequestLoggerEngine(conf config.Config, output *bytes.Buffer, status int) *gin.Engine {
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, output)
	mw := middleware.NewRequestLogger(conf, lf)
	engine := gin.New()
	engine.ContextWithFallback = true
	engine.Use(mw.Run)
	engine.GET("/upload", func(ctx *gin.Context) {
		ctx.Status(status)
	})
	return engine
}

func requestEndLog(t *testing.T, logs string) map[string]any {
	return requestEndLogForPath(t, logs, "/upload")
}

func requestEndLogForPath(t *testing.T, logs string, path string) map[string]any {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry["msg"] == "Req End: "+path {
			entry["_raw"] = line
			return entry
		}
	}
	t.Fatal("request end log not found")
	return nil
}

func requestLogNumber(t *testing.T, entry map[string]any, key string) string {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(entry["_raw"].(string)))
	decoder.UseNumber()
	var decoded map[string]any
	require.NoError(t, decoder.Decode(&decoded))
	value, ok := decoded[key].(json.Number)
	require.Truef(t, ok, "log attribute %q not found", key)
	return value.String()
}
