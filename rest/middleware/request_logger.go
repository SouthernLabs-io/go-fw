package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/log"
	resterrors "github.com/southernlabs-io/go-fw/rest/errors"
)

type RequestLoggerMiddleware struct {
	BaseMiddleware

	lf         log.LoggerFactory
	excludeMap map[string]bool
}

const redactedQueryValue = "[REDACTED]"

var sensitiveQueryParameters = map[string]bool{
	"access_token":  true,
	"api_key":       true,
	"apikey":        true,
	"id_token":      true,
	"passwd":        true,
	"password":      true,
	"refresh_token": true,
	"secret":        true,
	"sig":           true,
	"signature":     true,
	"token":         true,
}

func NewRequestLogger(conf config.Config, lf log.LoggerFactory) *RequestLoggerMiddleware {
	excludes := conf.HttpServer.ReqLoggerExcludes
	excludeMap := make(map[string]bool, len(excludes))
	for _, exclude := range excludes {
		if path.IsAbs(exclude) {
			excludeMap[exclude] = true
		} else {
			excludeMap[path.Join(conf.HttpServer.BasePath, exclude)] = true
		}
	}
	logger := lf.GetLoggerForType(RequestLoggerMiddleware{})
	logger.Infof("Excluded paths: %+v", maps.Keys(excludeMap))

	return &RequestLoggerMiddleware{
		BaseMiddleware{conf, logger},
		lf,
		excludeMap,
	}
}

func (m *RequestLoggerMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityHighest + 1
}

type responseWriter struct {
	http.ResponseWriter
	http.Flusher
	statusCode  int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(p)
}

func (rw *responseWriter) Status() int {
	if rw.statusCode == 0 {
		return http.StatusOK
	}
	return rw.statusCode
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func redactURLQuery(original *url.URL) (*url.URL, url.Values) {
	query := original.Query()
	didRedact := false
	for key, values := range query {
		if !sensitiveQueryParameters[strings.ToLower(key)] {
			continue
		}
		didRedact = true
		for i := range values {
			values[i] = redactedQueryValue
		}
	}
	if !didRedact {
		return original, query
	}
	redacted := *original
	redacted.RawQuery = query.Encode()
	return &redacted, query
}

func redactURLString(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "[INVALID URL]"
	}
	redacted, _ := redactURLQuery(parsed)
	if redacted == parsed {
		return rawURL
	}
	return redacted.String()
}

func (m *RequestLoggerMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		headers := r.Header

		urlPath := r.URL.Path
		start := time.Now()
		requestID := headers.Get("Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		ctx = context.WithValue(ctx, context.RequestIDCtxKey.(string), requestID)

		// Parse the host and port by using URL struct
		hostPortURL := url.URL{Host: r.Host}
		hostname := hostPortURL.Hostname()
		portStr := hostPortURL.Port()
		portAttr := slog.Attr{}
		if portStr != "" {
			port, err := strconv.Atoi(portStr)
			if err == nil {
				portAttr = slog.Int("port", port)
			}
		}

		clientIP := headers.Get("X-Forwarded-Id")
		if clientIP == "" {
			clientIP = headers.Get("X-Real-Ip")
		}
		if clientIP == "" {
			addr, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
			if err == nil {
				clientIP = addr
			}
		}

		_, redactedQuery := redactURLQuery(r.URL)

		attrs := []slog.Attr{
			slog.GroupAttrs("http",
				slog.String("method", r.Method),
				slog.String("url", redactURLString(r.RequestURI)),
				slog.String("request_id", requestID),
				slog.String("referer", redactURLString(r.Referer())),
				slog.String("useragent", r.UserAgent()),
				slog.String("version", r.Proto),
				slog.GroupAttrs("url_details",
					slog.String("host", hostname),
					portAttr,
					slog.String("path", urlPath),
					slog.Any("queryString", redactedQuery),
				),
			),
			slog.String("network.client.ip", clientIP),
		}

		if m.Conf.Datadog.Tracing {
			span, spanFound := tracer.SpanFromContext(ctx)
			if spanFound {
				spanCtx := span.Context()
				attrs = append(attrs,
					// Use flat dd to avoid classing with previous/later dd groups.
					slog.Uint64("dd.trace_id", spanCtx.TraceID()),
					slog.Uint64("dd.span_id", spanCtx.SpanID()),
				)
			} else {
				// Should not happen!
				logger := log.GetLoggerFromCtx(ctx).WithAttrs(attrs...)
				logger.Errorf("tracing is enabled but there is no span in the context!")
			}
		}

		ctx = log.CtxAppendLoggerAttrs(ctx, attrs...)
		logger := log.GetLoggerFromCtxForType(ctx, m)
		logReq := !m.excludeMap[r.URL.Path]
		if logReq {
			// Log request started
			logger.Debugf("Req Start: %s", urlPath)
		}
		var rw *responseWriter
		if flusher, ok := w.(http.Flusher); ok {
			// Wrap the writer to capture status code
			rw = &responseWriter{ResponseWriter: w, Flusher: flusher}
		} else {
			rw = &responseWriter{ResponseWriter: w}
		}

		if ctx != r.Context() {
			// Only update the request if the context changed
			logger.Warn("Request context was modified by previous middleware, this is not recommended")
			r = r.WithContext(ctx)
		}

		// Continue the chain
		next.ServeHTTP(rw, r)

		latency := time.Since(start)
		logger = log.GetLoggerFromCtx(ctx)
		status := rw.Status()
		level := config.LogLevelInfo
		if status >= 500 {
			level = config.LogLevelError
		} else if status == resterrors.StatusClientClosed {
			// Client closed the request (499): normal browser/client behavior, not a server error.
			level = config.LogLevelDebug
		} else if status >= 400 {
			level = config.LogLevelWarn
		}

		// Log the request if asked or if level is higher than info
		if logReq || level > config.LogLevelInfo {
			logger.Log(level, "Req End: "+urlPath,
				slog.Int("http.status_code", status),
				// Using "duration" to follow DataDog expectations
				slog.Duration("duration", latency),
				// r.Pattern should be populated at this point
				slog.String("http.url_details.pattern", r.Pattern),
			)
		}
	})
}
