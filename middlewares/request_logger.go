package middlewares

import (
	"log/slog"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/exp/maps"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	lib "github.com/southernlabs-io/go-fw/core"
)

type RequestLoggerMiddleware struct {
	BaseMiddleware

	lf         *lib.LoggerFactory
	excludeMap map[string]bool
}

func NewRequestLogger(conf lib.Config, lf *lib.LoggerFactory) *RequestLoggerMiddleware {
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

func (m *RequestLoggerMiddleware) Setup(httpHandler lib.HTTPHandler) {
	httpHandler.Root.Use(m.Run)
}

func (m *RequestLoggerMiddleware) Priority() MiddlewarePriority {
	return MiddlewarePriorityHighest
}

func (m *RequestLoggerMiddleware) Run(ctx *gin.Context) {
	m.lf.SetCtx(ctx)

	urlPath := ctx.Request.URL.Path
	start := time.Now()
	requestID := ctx.GetHeader("Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}
	ctx.Set(lib.RequestIDCtxKey.(string), requestID)

	// Parse the host and port by using URL struct
	hostPortURL := url.URL{Host: ctx.Request.Host}
	hostname := hostPortURL.Hostname()
	portStr := hostPortURL.Port()
	portAttr := slog.Attr{}
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err == nil {
			portAttr = slog.Int("port", port)
		}
	}

	attrs := []slog.Attr{
		slog.Group("http",
			slog.String("method", ctx.Request.Method),
			slog.String("url", ctx.Request.RequestURI),
			slog.String("request_id", requestID),
			slog.String("referer", ctx.Request.Referer()),
			slog.String("useragent", ctx.Request.UserAgent()),
			slog.String("version", ctx.Request.Proto),
			slog.Group("url_details",
				slog.String("host", hostname),
				portAttr,
				slog.String("path", urlPath),
				slog.Any("queryString", ctx.Request.URL.Query()),
			),
		),
		slog.String("network.client.ip", ctx.ClientIP()),
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
			logger := lib.GetLoggerFromCtx(ctx).WithAttrs(attrs...)
			logger.Errorf("tracing is enabled but there is no span in the context!")
		}
	}

	lib.CtxAppendLoggerAttrs(ctx, attrs...)

	if m.excludeMap[ctx.FullPath()] {
		return
	}

	logger := lib.GetLoggerFromCtxForType(ctx, m)
	logger.Debugf("Req Start: %s", urlPath)

	ctx.Next()

	latency := time.Since(start)
	logger = lib.GetLoggerFromCtx(ctx)
	status := ctx.Writer.Status()
	level := lib.LogLevelInfo
	if status >= 500 {
		level = lib.LogLevelError
	} else if status >= 400 {
		level = lib.LogLevelWarn
	}
	logger.Log(level, "Req End: "+urlPath,
		slog.Int("http.status_code", status),
		// Using "duration" to follow DataDog expectations
		slog.Duration("duration", latency),
	)
}

var RequestLoggerModule = ProvideAsMiddleware(NewRequestLogger)
