package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	rest "github.com/southernlabs-io/go-fw/rest_gin"
	"github.com/stretchr/testify/require"
)

// HTTPHandlerClientGin is a client for testing HTTPHandler implementations.
type HTTPHandlerClientGin struct {
	t           *testing.T
	httpHandler rest.HTTPHandler
	headers     http.Header
}

func NewHTTPClientGin(t *testing.T, httpHandler rest.HTTPHandler) *HTTPHandlerClientGin {
	return &HTTPHandlerClientGin{
		t:           t,
		httpHandler: httpHandler,
	}
}

func (c *HTTPHandlerClientGin) SetHeaders(headers http.Header) *HTTPHandlerClientGin {
	c.headers = headers
	return c
}

func (c *HTTPHandlerClientGin) GET(
	urlFormat string,
	args ...any,
) *ResponseGin {
	return c.Do(http.MethodGet, fmt.Sprintf(urlFormat, args...), nil)
}

func (c *HTTPHandlerClientGin) POST(
	body any,
	urlFormat string,
	args ...any,
) *ResponseGin {
	return c.Do(http.MethodPost, fmt.Sprintf(urlFormat, args...), body)
}

func (c *HTTPHandlerClientGin) PATCH(
	body any,
	urlFormat string,
	args ...any,
) *ResponseGin {
	return c.Do(http.MethodPatch, fmt.Sprintf(urlFormat, args...), body)
}

func (c *HTTPHandlerClientGin) DELETE(
	urlFormat string,
	args ...any,
) *ResponseGin {
	return c.Do(http.MethodDelete, fmt.Sprintf(urlFormat, args...), nil)
}

func (c *HTTPHandlerClientGin) Do(method, url string, body any) *ResponseGin {
	var reader io.Reader
	if body != nil {
		if bodyReader, ok := body.(io.Reader); ok {
			reader = bodyReader
		} else {
			jsonBytes, err := json.Marshal(body)
			require.NoError(c.t, err)
			reader = bytes.NewReader(jsonBytes)
		}
	}

	req, err := http.NewRequest(
		method,
		url,
		reader,
	)
	require.NoError(c.t, err)

	req.Header = c.headers

	res := &ResponseGin{
		t:  c.t,
		rr: httptest.NewRecorder(),
	}
	c.httpHandler.Engine.ServeHTTP(res.rr, req)
	return res
}

type ResponseGin struct {
	t  *testing.T
	rr *httptest.ResponseRecorder
}

// RequireJSONBodyAs decodes the response body as JSON into the given body.
// Target must be a reference to store the deserialized body.
func (r *ResponseGin) RequireJSONBodyAs(target any) {
	require.Equal(r.t, "application/json; charset=utf-8", r.rr.Header().Get("Content-Type"))
	require.Greater(r.t, r.rr.Body.Len(), 0)
	err := json.NewDecoder(r.rr.Body).Decode(target)
	require.NoError(r.t, err)
}

func (r *ResponseGin) RequireStatus(status int) *ResponseGin {
	require.Equalf(r.t, status, r.rr.Code, "expected status %d, got %d, body: %s", status, r.rr.Code, r.rr.Body.String())
	return r
}

func (r *ResponseGin) RequireHeader(header, value string) *ResponseGin {
	require.Contains(r.t, r.rr.Header().Values(header), value)
	return r
}

func (r *ResponseGin) RequireEmptyBody() *ResponseGin {
	require.Empty(r.t, r.rr.Body)
	return r
}
