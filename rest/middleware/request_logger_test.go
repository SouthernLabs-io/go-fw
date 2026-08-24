package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/southernlabs-io/go-fw/config"
	fwcontext "github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest/middleware"
)

func TestRequestLoggerMiddleware_ResponseWriterGuard(t *testing.T) {
	conf := config.Config{}
	lf := log.NewLoggerFactory(config.RootConfig{})
	mw := middleware.NewRequestLogger(conf, lf)

	// Case 1: Multiple WriteHeader calls — only the first status should be recorded
	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.WriteHeader(http.StatusInternalServerError) // Should be guarded & ignored
		_, _ = w.Write([]byte("not found"))
	})

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/test1", nil)
	mw.Handle(handler1).ServeHTTP(rec1, req1)

	require.Equal(t, http.StatusNotFound, rec1.Code)
	require.Equal(t, "not found", rec1.Body.String())

	// Case 2: Write called without WriteHeader — implicit 200 OK should be set
	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
		w.WriteHeader(http.StatusBadRequest) // Superfluous after Write, should be ignored
	})

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/test2", nil)
	mw.Handle(handler2).ServeHTTP(rec2, req2)

	require.Equal(t, http.StatusOK, rec2.Code)
	require.Equal(t, "hello world", rec2.Body.String())
}

func TestRequestLoggerMiddleware_SecureRequestMetadataByDefault(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, &output)
	mw := middleware.NewRequestLogger(conf, lf)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "request-secret", r.URL.Query().Get("token"))
		require.Equal(t, "visible", r.URL.Query().Get("safe"))
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/upload?token=request-secret&secret-value=unknown-secret&safe=visible", nil)
	req.Header.Set("Referer", "https://example.com/editor?token=referer-secret&view=graph")
	req = req.WithContext(lf.AddToCtx(req.Context()))
	mw.Handle(handler).ServeHTTP(httptest.NewRecorder(), req)

	logs := output.String()
	for _, secret := range []string{"request-secret", "unknown-secret", "referer-secret"} {
		require.NotContains(t, logs, secret)
	}

	var requestLog map[string]any
	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		if entry["msg"] == "Req End: /upload" {
			requestLog = entry
		}
	}
	require.NotNil(t, requestLog)

	httpAttrs := requestLog["http"].(map[string]any)
	require.Equal(t, "/upload", httpAttrs["url"])
	require.NotContains(t, httpAttrs, "referer")

	urlDetails := httpAttrs["url_details"].(map[string]any)
	require.Equal(t, "/upload", urlDetails["path"])
	require.NotContains(t, urlDetails, "queryString")
}

func TestRequestLoggerMiddleware_AllowsConfiguredQueryParameters(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	conf.HttpServer.RequestLogger.QueryParameterAllowlist = []string{"view"}
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, &output)
	mw := middleware.NewRequestLogger(conf, lf)

	req := httptest.NewRequest(http.MethodGet, "/upload?view=graph&token=request-secret&view=table", nil)
	req = req.WithContext(lf.AddToCtx(req.Context()))
	mw.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(httptest.NewRecorder(), req)

	requestLog := requestEndLog(t, output.String())
	httpAttrs := requestLog["http"].(map[string]any)
	urlDetails := httpAttrs["url_details"].(map[string]any)
	query := urlDetails["queryString"].(map[string]any)
	require.Equal(t, []any{"graph", "table"}, query["view"])
	require.NotContains(t, query, "token")
	require.NotContains(t, output.String(), "request-secret")
	require.Equal(t, "/upload", httpAttrs["url"])
}

func TestRequestLoggerMiddleware_MalformedQueryIsOmitted(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	conf.HttpServer.RequestLogger.QueryParameterAllowlist = []string{"view"}
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, &output)
	mw := middleware.NewRequestLogger(conf, lf)

	req := httptest.NewRequest(http.MethodGet, "/upload?view=%ZZ&secret-value=unknown-secret", nil)
	req = req.WithContext(lf.AddToCtx(req.Context()))
	mw.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(httptest.NewRecorder(), req)

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
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, &output)
	mw := middleware.NewRequestLogger(conf, lf)

	req := httptest.NewRequest(http.MethodGet, "/upload", nil)
	req.Header.Set("Referer", "https://example.com/editor?token=referer-secret#details")
	req = req.WithContext(lf.AddToCtx(req.Context()))
	mw.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(httptest.NewRecorder(), req)

	requestLog := requestEndLog(t, output.String())
	httpAttrs := requestLog["http"].(map[string]any)
	require.Equal(t, "https://example.com/editor", httpAttrs["referer"])
	require.NotContains(t, output.String(), "referer-secret")
}

func TestRequestLoggerMiddleware_MalformedReferrerIsOmitted(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	conf.HttpServer.RequestLogger.ReferrerMode = config.RequestLoggerReferrerOriginPath
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, &output)
	mw := middleware.NewRequestLogger(conf, lf)

	req := httptest.NewRequest(http.MethodGet, "/upload", nil)
	req.Header.Set("Referer", "https://example.com/%ZZ?token=referer-secret")
	req = req.WithContext(lf.AddToCtx(req.Context()))
	mw.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(httptest.NewRecorder(), req)

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
	span, spanCtx := tracer.StartSpanFromContext(fwcontext.Background(), "request")
	t.Cleanup(func() { span.Finish() })

	calledWithRequestID := ""
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		calledWithRequestID = fwcontext.GetRequestIDFromCtx(r.Context())
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/items/42", nil)
	req.Header.Set("Request-ID", "request-id-123")
	req = req.WithContext(lf.AddToCtx(spanCtx))
	recorder := httptest.NewRecorder()
	mw.Handle(mux).ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Equal(t, "request-id-123", calledWithRequestID)

	requestLog := requestEndLogForPath(t, output.String(), "/items/42")
	require.Equal(t, float64(http.StatusCreated), requestLog["http.status_code"])
	require.Equal(t, "GET /items/{id}", requestLog["http.url_details.pattern"])
	require.Equal(t, strconv.FormatUint(span.Context().TraceID(), 10), requestLogNumber(t, requestLog, "dd.trace_id"))
	require.Equal(t, strconv.FormatUint(span.Context().SpanID(), 10), requestLogNumber(t, requestLog, "dd.span_id"))
}

func TestRequestLoggerMiddleware_ExcludedRouteStillExecutesWithoutRequestLog(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	conf.HttpServer.ReqLoggerExcludes = []string{"/health"}
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, &output)
	mw := middleware.NewRequestLogger(conf, lf)

	handlerCalled := false
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req = req.WithContext(lf.AddToCtx(req.Context()))
	recorder := httptest.NewRecorder()
	mw.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, req)

	require.True(t, handlerCalled)
	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.NotContains(t, output.String(), "Req End: /health")
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
