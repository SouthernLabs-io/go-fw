package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
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

func newRequestLoggerEngine(conf config.Config, output *bytes.Buffer, status int) *gin.Engine {
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, output)
	mw := middleware.NewRequestLogger(conf, lf)
	engine := gin.New()
	engine.Use(mw.Run)
	engine.GET("/upload", func(ctx *gin.Context) {
		ctx.Status(status)
	})
	return engine
}

func requestEndLog(t *testing.T, logs string) map[string]any {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry["msg"] == "Req End: /upload" {
			return entry
		}
	}
	t.Fatal("request end log not found")
	return nil
}
