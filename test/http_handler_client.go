package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/core"
)

// HTTPHandlerClient is a client for testing HTTPHandler implementations.
type HTTPHandlerClient struct {
	t           *testing.T
	httpHandler core.HTTPHandler
	headers     http.Header
}

func NewHTTPClient(t *testing.T, httpHandler core.HTTPHandler) *HTTPHandlerClient {
	return &HTTPHandlerClient{
		t:           t,
		httpHandler: httpHandler,
	}
}

func (c *HTTPHandlerClient) SetHeaders(headers http.Header) *HTTPHandlerClient {
	c.headers = headers
	return c
}

func (c *HTTPHandlerClient) GET(
	urlFormat string,
	args ...any,
) *Response {
	return c.Do(http.MethodGet, fmt.Sprintf(urlFormat, args...), nil)
}

func (c *HTTPHandlerClient) POST(
	body any,
	urlFormat string,
	args ...any,
) *Response {
	return c.Do(http.MethodPost, fmt.Sprintf(urlFormat, args...), body)
}

func (c *HTTPHandlerClient) PATCH(
	body any,
	urlFormat string,
	args ...any,
) *Response {
	return c.Do(http.MethodPatch, fmt.Sprintf(urlFormat, args...), body)
}

func (c *HTTPHandlerClient) DELETE(
	urlFormat string,
	args ...any,
) *Response {
	return c.Do(http.MethodDelete, fmt.Sprintf(urlFormat, args...), nil)
}

func (c *HTTPHandlerClient) Do(method, url string, body any) *Response {
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

	res := &Response{
		t:  c.t,
		rr: httptest.NewRecorder(),
	}
	c.httpHandler.Engine.ServeHTTP(res.rr, req)
	return res
}

type Response struct {
	t  *testing.T
	rr *httptest.ResponseRecorder
}

// RequireJSONBodyAs decodes the response body as JSON into the given body.
// Target must be a reference to store the deserialized body.
func (r *Response) RequireJSONBodyAs(target any) {
	require.Equal(r.t, "application/json; charset=utf-8", r.rr.Header().Get("Content-Type"))
	require.Greater(r.t, r.rr.Body.Len(), 0)
	err := json.NewDecoder(r.rr.Body).Decode(target)
	require.NoError(r.t, err)
}

func (r *Response) RequireStatus(status int) *Response {
	require.Equalf(r.t, status, r.rr.Code, "expected status %d, got %d, body: %s", status, r.rr.Code, r.rr.Body.String())
	return r
}

func (r *Response) RequireHeader(header, value string) *Response {
	require.Contains(r.t, r.rr.Header().Values(header), value)
	return r
}

func (r *Response) RequireEmptyBody() {
	require.Empty(r.t, r.rr.Body)
}
