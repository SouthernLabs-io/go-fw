package core

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/gin-contrib/cors"

	"github.com/southernlabs-io/go-fw/errors"
)

type DatabaseConfig struct {
	Host string
	Port int
	User string
	Pass string
}

type HttpServerConfig struct {
	BindAddress       string
	Port              int
	ReqLoggerExcludes []string
	BasePath          string
	CORS              cors.Config
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

type EnvConfig struct {
	Name string
	Type EnvType
}

type LogConfigWriter string

const (
	LogConfigWriterStdout LogConfigWriter = "stdout"
	LogConfigWriterStderr LogConfigWriter = "stderr"
	LogConfigWriterBuffer LogConfigWriter = "buffer"
)

// UnmarshalText parses a logger writer from a string
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

//go:generate stringer -type=LogLevel -linecomment
type LogLevel slog.Level

const (
	LogLevelTrace = LogLevel(slog.LevelDebug - 4) // TRACE
	LogLevelDebug = LogLevel(slog.LevelDebug)     // DEBUG
	LogLevelInfo  = LogLevel(slog.LevelInfo)      // INFO
	LogLevelWarn  = LogLevel(slog.LevelWarn)      // WARN
	LogLevelError = LogLevel(slog.LevelError)     // ERROR
)

// UnmarshalText parses a log level from a string
//
//goland:noinspection GoMixedReceiverTypes
func (l *LogLevel) UnmarshalText(text []byte) error {
	levelStr := string(text)
	switch strings.ToUpper(levelStr) {
	case traceAsSlogStr, LogLevelTrace.String():
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
	// The 'S' in 'URLS' is capitalized to avoid the corresponding env variables being named
	// something like 'SLACK_WEBHOOK_UR_LS_INFO'
	WebhookURLS struct {
		Info  string
		Warn  string
		Error string
	}
}

type CoreConfig struct {
	Name    string
	Secrets SecretsConfig
	Env     EnvConfig
	Log     LogConfig
	Datadog DataDogConfig
}

type Config struct {
	CoreConfig

	Database DatabaseConfig

	Redis RedisConfig

	HttpServer HttpServerConfig

	JWT JWTConfig

	Slack SlackConfig
}

func NewConfig(core CoreConfig) Config {
	conf := Config{
		CoreConfig: core,
	}
	LoadConfig(core, &conf)
	return conf
}

func NewCoreConfig() CoreConfig {
	var conf CoreConfig
	loadConfig(&conf, nil)
	return conf
}

func LoadConfig(core CoreConfig, dst any) {
	loadConfig(dst, loadSecrets(core))
}
