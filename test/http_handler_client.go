package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// HTTPHandlerClient is a client for testing HTTPHandler implementations.
type HTTPHandlerClient struct {
	t       *testing.T
	baseURL string
	headers http.Header
	client  *http.Client
}

func NewHTTPClient(t *testing.T, baseURL string, client *http.Client) *HTTPHandlerClient {
	return &HTTPHandlerClient{
		t:       t,
		baseURL: baseURL,
		client:  client,
	}
}

func (c *HTTPHandlerClient) SetHeaders(headers http.Header) *HTTPHandlerClient {
	c.headers = headers
	return c
}

type BodyType string

const (
	BodyTypeText   BodyType = "text"
	BodyTypeJSON   BodyType = "json"
	BodyTypeYAML   BodyType = "yaml"
	BodyTypeBinary BodyType = "binary"
)

type AcceptType string

const (
	AcceptTypeJSON   AcceptType = "json"
	AcceptTypeYAML   AcceptType = "yaml"
	AcceptTypeBinary AcceptType = "binary"
	AcceptTypeText   AcceptType = "text"
	AcceptTypeSSE    AcceptType = "sse"
)

type RequestArgs struct {
	Body      any
	BodyType  BodyType
	UrlFormat string
	Headers   http.Header
	Accept    AcceptType
}

func (c *HTTPHandlerClient) GET(req RequestArgs, urlArgs ...any) *Response {
	return c.Do(http.MethodGet, req, urlArgs...)
}

func (c *HTTPHandlerClient) POST(req RequestArgs, urlArgs ...any) *Response {
	return c.Do(http.MethodPost, req, urlArgs...)
}

func (c *HTTPHandlerClient) PATCH(req RequestArgs, urlArgs ...any) *Response {
	return c.Do(http.MethodPatch, req, urlArgs...)
}

func (c *HTTPHandlerClient) PUT(req RequestArgs, urlArgs ...any) *Response {
	return c.Do(http.MethodPut, req, urlArgs...)
}

func (c *HTTPHandlerClient) DELETE(req RequestArgs, urlArgs ...any) *Response {
	return c.Do(http.MethodDelete, req, urlArgs...)
}

func (c *HTTPHandlerClient) Do(method string, reqArgs RequestArgs, urlArgs ...any) *Response {
	// Prepare body
	var reader io.Reader
	if reqArgs.Body != nil {
		switch reqArgs.BodyType {
		case BodyTypeText:
			if reqArgs.Body != nil {
				if bodyStr, ok := reqArgs.Body.(string); ok {
					reader = bytes.NewReader([]byte(bodyStr))
				} else if bodyBytes, ok := reqArgs.Body.([]byte); ok {
					reader = bytes.NewReader(bodyBytes)
				} else {
					require.Fail(c.t, "Body must be string or []byte for BodyTypeText")
				}
			}
		case BodyTypeJSON:
			if reqArgs.Body != nil {
				jsonBytes, err := json.Marshal(reqArgs.Body)
				require.NoError(c.t, err)
				reader = bytes.NewReader(jsonBytes)
			}
		case BodyTypeYAML:
			if reqArgs.Body != nil {
				yamlBytes, err := yaml.Marshal(reqArgs.Body)
				require.NoError(c.t, err)
				reader = bytes.NewReader(yamlBytes)
			}
		case BodyTypeBinary:
			if reqArgs.Body != nil {
				if bodyReader, ok := reqArgs.Body.(io.Reader); ok {
					reader = bodyReader
				} else if bodyBytes, ok := reqArgs.Body.([]byte); ok {
					reader = bytes.NewReader(bodyBytes)
				} else {
					require.Fail(c.t, "Body must be io.Reader or []byte for BodyTypeBinary")
				}
			}
		default:
			require.Fail(c.t, "Unsupported BodyType", reqArgs.BodyType)
		}
	}

	// Format URL
	var url string
	if len(urlArgs) > 0 {
		url = fmt.Sprintf("%s"+reqArgs.UrlFormat, append([]any{c.baseURL}, urlArgs...)...)
	} else {
		url = c.baseURL + reqArgs.UrlFormat
	}

	// Create request
	req, err := http.NewRequest(
		method,
		url,
		reader,
	)
	require.NoError(c.t, err)

	// Set Content-Type header
	if reqArgs.Body != nil &&
		(reqArgs.Headers == nil || len(reqArgs.Headers.Values("Content-Type")) == 0) {
		switch reqArgs.BodyType {
		case BodyTypeText:
			req.Header.Set("Content-Type", "text/plain")
		case BodyTypeJSON:
			req.Header.Set("Content-Type", "application/json")
		case BodyTypeYAML:
			req.Header.Set("Content-Type", "application/x-yaml")
		case BodyTypeBinary:
			req.Header.Set("Content-Type", "application/octet-stream")
		}
	}

	// Set client default headers
	for key, values := range c.headers {
		for _, value := range values {
			req.Header.Set(key, value)
		}
	}

	// Set request specific headers
	for key, values := range reqArgs.Headers {
		for _, value := range values {
			req.Header.Set(key, value)
		}
	}
	// Set Accept header
	if reqArgs.Accept != "" {
		acceptVal := ""
		switch reqArgs.Accept {
		case AcceptTypeJSON:
			acceptVal = "application/json"
		case AcceptTypeYAML:
			acceptVal = "application/yaml"
		case AcceptTypeBinary:
			acceptVal = "application/octet-stream"
		case AcceptTypeText:
			acceptVal = "text/plain"
		case AcceptTypeSSE:
			acceptVal = "text/event-stream"
		default:
			require.Fail(c.t, "Unsupported AcceptType", reqArgs.Accept)
		}
		req.Header.Set("Accept", acceptVal)
	}

	// Do request
	res, err := c.client.Do(req)
	require.NoError(c.t, err)
	return NewResponse(c.t, res)
}

type Response struct {
	t    *testing.T
	rr   *http.Response
	body *bytes.Buffer
}

func NewResponse(t *testing.T, rr *http.Response) *Response {
	return &Response{
		t:  t,
		rr: rr,
	}
}

// RequireJSONBodyAs decodes the response body as JSON into the given body.
// Target must be a reference to store the deserialized body.
func (r *Response) RequireJSONBodyAs(target any) {
	require.Equal(r.t, "application/json", r.rr.Header.Get("Content-Type"))
	require.Greater(r.t, len(r.BodyBytes()), 0)
	err := json.NewDecoder(r.Body()).Decode(target)
	require.NoError(r.t, err)
}

func (r *Response) RequireStatus(status int) {
	require.Equalf(r.t, status, r.rr.StatusCode, "expected status %d, got %d, url: %s, body: %s", status, r.rr.StatusCode, r.rr.Request.RequestURI, r.BodyString())
}

func (r *Response) RequireHeader(header, value string) {
	require.Contains(r.t, r.rr.Header.Values(header), value)
}

func (r *Response) RequireEmptyBody() {
	require.Empty(r.t, r.BodyBytes())
}

func (r *Response) BodyBytes() []byte {
	if r.body == nil {
		buff := &bytes.Buffer{}
		_, err := io.Copy(buff, r.rr.Body)
		require.NoError(r.t, err)
		err = r.rr.Body.Close()
		require.NoError(r.t, err)
		r.body = buff
	}
	return r.body.Bytes()
}

func (r *Response) BodyString() string {
	bytes := r.BodyBytes()
	if bytes == nil {
		return "<nil>"
	}
	return string(bytes)
}

func (r *Response) Body() io.ReadCloser {
	if r.body == nil {
		return r.rr.Body
	}
	return io.NopCloser(r.body)
}

func (r *Response) StatusCode() int {
	return r.rr.StatusCode
}
