package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/errors"
)

type DatabaseConfig struct {
	Host string
	Port int
	User string
	Pass string
}

type CORS struct {
	cors.Config
}

func (c CORS) MarshalJSON() ([]byte, error) {
	// These are all the JSON compatible field of cors.Config@v1.7.2
	m := map[string]interface{}{
		"AllowAllOrigins":           c.AllowAllOrigins,
		"AllowOrigins":              c.AllowOrigins,
		"AllowMethods":              c.AllowMethods,
		"AllowPrivateNetwork":       c.AllowPrivateNetwork,
		"AllowHeaders":              c.AllowHeaders,
		"AllowCredentials":          c.AllowCredentials,
		"ExposeHeaders":             c.ExposeHeaders,
		"MaxAge":                    c.MaxAge,
		"AllowWildcard":             c.AllowWildcard,
		"AllowBrowserExtensions":    c.AllowBrowserExtensions,
		"CustomSchemas":             c.CustomSchemas,
		"AllowWebSockets":           c.AllowWebSockets,
		"AllowFiles":                c.AllowFiles,
		"OptionsResponseStatusCode": c.OptionsResponseStatusCode,
	}
	return json.Marshal(m)
}

type HttpServerConfig struct {
	BindAddress       string
	Port              int
	ReqLoggerExcludes []string
	BasePath          string
	CORS              CORS
}

type RedisConfig struct {
	URL string
}

type TLSConfig struct {
	CertFolder   string
	CACertBase64 string
	CAKeyBase64  string
}

type JWTConfig struct {
	SigningKey string
}

type DataDogConfig struct {
	Profiling bool
	Tracing   bool
	Agent     string
}

type EnvType string

const EnvTypeProd EnvType = "prod"
const EnvTypeSandbox EnvType = "sandbox"
const EnvTypeLocal EnvType = "local"
const EnvTypeTest EnvType = "test"

type Host string

func (h *Host) MarshalText() ([]byte, error) {
	return []byte(*h), nil
}

func (h *Host) UnmarshalText(text []byte) error {
	*h = Host(text)
	if *h == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return errors.NewUnknownf("failed to get hostname: %w", err)
		}
		*h = Host(hostname)
	}
	return nil
}

type EnvConfig struct {
	Name string
	Type EnvType
	Host Host
}

type LogConfigWriter string

const (
	LogConfigWriterStdout LogConfigWriter = "stdout"
	LogConfigWriterStderr LogConfigWriter = "stderr"
	LogConfigWriterBuffer LogConfigWriter = "buffer"
)

// UnmarshalText parses a logger writer from a string
//
//goland:noinspection GoMixedReceiverTypes
func (w *LogConfigWriter) UnmarshalText(text []byte) error {
	writerStr := string(text)
	switch LogConfigWriter(writerStr) {
	case LogConfigWriterStdout, LogConfigWriterStderr, LogConfigWriterBuffer:
		*w = LogConfigWriter(writerStr)
	case "":
		*w = LogConfigWriterStdout
	default:
		return fmt.Errorf("invalid logger writer: %s", writerStr)
	}
	return nil
}

// MarshalText returns the string representation of a logger writer
//
//goland:noinspection GoMixedReceiverTypes
func (w LogConfigWriter) MarshalText() ([]byte, error) {
	return []byte(w), nil
}

//go:generate stringer -type=LogLevel -linecomment
type LogLevel slog.Level

const (
	LogLevelTrace = LogLevel(slog.LevelDebug - 4) // TRACE
	LogLevelDebug = LogLevel(slog.LevelDebug)     // DEBUG
	LogLevelInfo  = LogLevel(slog.LevelInfo)      // INFO
	LogLevelWarn  = LogLevel(slog.LevelWarn)      // WARN
	LogLevelError = LogLevel(slog.LevelError)     // ERROR
)

// TraceAsSlogStr is the string representation of the TRACE log level within the slog package: "DEBUG-4"
var TraceAsSlogStr = slog.Level(LogLevelTrace).String()

// UnmarshalText parses a log level from a string
//
//goland:noinspection GoMixedReceiverTypes
func (l *LogLevel) UnmarshalText(text []byte) error {
	levelStr := string(text)
	switch strings.ToUpper(levelStr) {
	case TraceAsSlogStr, LogLevelTrace.String():
		*l = LogLevelTrace
	case "", LogLevelDebug.String():
		*l = LogLevelDebug
	case LogLevelInfo.String():
		*l = LogLevelInfo
	case LogLevelWarn.String():
		*l = LogLevelWarn
	case LogLevelError.String():
		*l = LogLevelError
	default:
		return errors.Newf(errors.ErrCodeBadArgument, "invalid log level: %s", levelStr)
	}
	return nil
}

// MarshalText returns the string representation of a log level
//
//goland:noinspection GoMixedReceiverTypes
func (l LogLevel) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

type LogConfig struct {
	Level  LogLevel
	Levels map[string]LogLevel
	Writer LogConfigWriter
}

type SecretsConfig struct {
	PrefixFmt string
	KeyFmt    string
}

type SlackConfig struct {
	Enabled            bool
	HTTPTimeoutSeconds int
	WebhookURLs        struct {
		Info  string
		Warn  string
		Error string
	}
}

type RootConfig struct {
	Name    string
	Secrets SecretsConfig
	Env     EnvConfig
	Log     LogConfig
	Datadog DataDogConfig
}

type Config struct {
	RootConfig

	Database DatabaseConfig

	Redis RedisConfig

	HttpServer HttpServerConfig

	JWT JWTConfig

	Slack SlackConfig
}

func NewConfig(root RootConfig, secretsMgr SecretsManager) Config {
	conf := Config{
		RootConfig: root,
	}
	LoadConfig(root, &conf, secretsMgr)
	return conf
}

var rootConfig RootConfig

func init() {
	loadConfig(&rootConfig, nil)
}

func GetCoreConfig() RootConfig {
	return rootConfig
}

func LoadConfig[T any](root RootConfig, dst *T, secretsMgr SecretsManager) {
	if secretsMgr == nil {
		secretsMgr = PanicSecretsManager{}
	}
	loadConfig(dst, loadSecrets(root, secretsMgr))
}

// Module exports dependency
var Module = fx.Options(
	fx.Provide(fx.Annotate(NewConfig, fx.ParamTags("", `optional:"true"`))),
	fx.Provide(GetCoreConfig),
)
