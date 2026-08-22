package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
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

func TestRequestLoggerMiddleware_RedactsSensitiveQueryParameters(t *testing.T) {
	var output bytes.Buffer
	conf := config.Config{}
	lf := log.NewLoggerFactoryWithWriter(config.RootConfig{
		Log: config.LogConfig{Level: config.LogLevelInfo, Structured: true},
	}, &output)
	mw := middleware.NewRequestLogger(conf, lf)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "request-secret", r.URL.Query().Get("token"))
		require.Equal(t, "case-secret", r.URL.Query().Get("TOKEN"))
		require.Equal(t, "visible", r.URL.Query().Get("safe"))
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/upload?token=request-secret&TOKEN=case-secret&access_token=access-secret&safe=visible", nil)
	req.Header.Set("Referer", "https://example.com/editor?token=referer-secret&view=graph")
	req = req.WithContext(lf.AddToCtx(req.Context()))
	mw.Handle(handler).ServeHTTP(httptest.NewRecorder(), req)

	logs := output.String()
	for _, secret := range []string{"request-secret", "case-secret", "access-secret", "referer-secret"} {
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
	require.Equal(t, "/upload?TOKEN=%5BREDACTED%5D&access_token=%5BREDACTED%5D&safe=visible&token=%5BREDACTED%5D", httpAttrs["url"])
	require.Equal(t, "https://example.com/editor?token=%5BREDACTED%5D&view=graph", httpAttrs["referer"])

	urlDetails := httpAttrs["url_details"].(map[string]any)
	query := urlDetails["queryString"].(map[string]any)
	require.Equal(t, []any{"[REDACTED]"}, query["token"])
	require.Equal(t, []any{"[REDACTED]"}, query["TOKEN"])
	require.Equal(t, []any{"[REDACTED]"}, query["access_token"])
	require.Equal(t, []any{"visible"}, query["safe"])
}
