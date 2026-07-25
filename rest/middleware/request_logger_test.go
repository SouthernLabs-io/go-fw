package middleware_test

import (
	"net/http"
	"net/http/httptest"
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
