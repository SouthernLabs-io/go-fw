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
)

type RequestLoggerMiddleware struct {
	BaseMiddleware

	lf         log.LoggerFactory
	excludeMap map[string]bool
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
	return MiddlewarePriorityHighest
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
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

		attrs := []slog.Attr{
			slog.GroupAttrs("http",
				slog.String("method", r.Method),
				slog.String("url", r.RequestURI),
				slog.String("request_id", requestID),
				slog.String("referer", r.Referer()),
				slog.String("useragent", r.UserAgent()),
				slog.String("version", r.Proto),
				slog.GroupAttrs("url_details",
					slog.String("host", hostname),
					portAttr,
					slog.String("path", urlPath),
					slog.Any("queryString", r.URL.Query()),
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
		rw := &responseWriter{w, 0}
		if ctx != r.Context() {
			// Only update the request if the context changed
			logger.Warn("Request context was modified by previous middleware, this is not recommended")
			r = r.WithContext(ctx)
		}

		// Continue the chain
		next.ServeHTTP(rw, r)

		latency := time.Since(start)
		logger = log.GetLoggerFromCtx(ctx)
		status := rw.statusCode
		level := config.LogLevelInfo
		if status >= 500 {
			level = config.LogLevelError
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

var RequestLoggerModule = ProvideAsMiddleware(NewRequestLogger)
