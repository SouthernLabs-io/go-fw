package slack

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/iancoleman/strcase"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/version"
)

type WebhookChannelType string

const (
	WebhookChannelTypeInfo  WebhookChannelType = "ℹ️ Info"
	WebhookChannelTypeWarn  WebhookChannelType = "⚠️ Warn"
	WebhookChannelTypeError WebhookChannelType = "❗️Error"
)

type Client struct {
	conf            core.Config
	httpClient      *http.Client
	fullMsgTemplate *template.Template
}

func NewSlackClient(conf core.Config, lf *core.LoggerFactory) *Client {
	if !conf.Slack.Enabled {
		lf.GetLogger().Infof("Slack notifications are disabled")
		return nil
	}

	// Check we have at least one webhook configured
	if len(conf.Slack.WebhookURLS.Error) == 0 && len(conf.Slack.WebhookURLS.Warn) == 0 && len(conf.Slack.WebhookURLS.Info) == 0 {
		panic(errors.Newf(errors.ErrCodeBadState, "no Slack Webhook URLs configured, but it was explicitly enabled"))
	}

	// Render the context section template once
	contextSectionTemplate, err := template.New("contextSection").Parse(contextSectionTemplateSrc)
	if err != nil {
		panic(errors.Newf(errors.ErrCodeBadState, "failed to parse Slack context section template: %w", err))
	}

	buf := bytes.Buffer{}
	err = contextSectionTemplate.Execute(&buf, map[string]any{
		"env_type":   strcase.ToCamel(string(conf.Env.Type)),
		"env_name":   strcase.ToCamel(conf.Env.Name),
		"host":       core.CachedHostname(),
		"release":    version.Release,
		"commit":     version.Commit,
		"build_time": version.BuildTime,
	})
	if err != nil {
		panic(errors.Newf(errors.ErrCodeBadState, "failed to execute Slack context section template: %w", err))
	}

	// Replace context section placeholder
	fullMsgTemplate, err := template.New("fullMessage").Parse(
		strings.ReplaceAll(fullMsgTemplateSrc, "${context_section}", buf.String()),
	)
	if err != nil {
		panic(errors.Newf(errors.ErrCodeBadState, "failed to parse Slack message template: %w", err))
	}

	return &Client{
		conf:            conf,
		fullMsgTemplate: fullMsgTemplate,
		httpClient: &http.Client{
			Timeout: time.Duration(conf.Slack.HTTPTimeoutSeconds) * time.Second,
		},
	}
}

func (s *Client) Infof(message string, args ...any) error {
	return s.Send(WebhookChannelTypeInfo, message, args)
}

func (s *Client) Infob(blocks []map[string]any) error {
	return s.SendWithBlocks(WebhookChannelTypeInfo, blocks)
}

func (s *Client) Warnf(message string, args ...any) error {
	return s.Send(WebhookChannelTypeWarn, message, args)
}

func (s *Client) Warnb(blocks []map[string]any) error {
	return s.SendWithBlocks(WebhookChannelTypeWarn, blocks)
}

func (s *Client) Errorf(message string, args ...any) error {
	return s.Send(WebhookChannelTypeError, message, args)
}

func (s *Client) Errorb(blocks []map[string]any) error {
	return s.SendWithBlocks(WebhookChannelTypeError, blocks)
}

func (s *Client) Send(channelType WebhookChannelType, message string, args []any) error {
	buf := bytes.Buffer{}
	err := s.fullMsgTemplate.Execute(&buf, map[string]any{
		"service_name": s.conf.Name,
		"level_msg":    channelType,
		"main_msg":     fmt.Sprintf(message, args...),
		"env_type":     s.conf.Env.Type,
		"env_name":     s.conf.Env.Name,
		"host":         core.CachedHostname(),
		"release":      version.Release,
		"commit":       version.Commit,
		"build_time":   version.BuildTime,
	})
	if err != nil {
		return err
	}
	return s.sendRaw(channelType, buf.Bytes())
}

func (s *Client) SendWithBlocks(channelType WebhookChannelType, blocks []map[string]any) error {
	return s.send(channelType, map[string]any{
		"blocks": blocks,
	})
}

func (s *Client) send(channelType WebhookChannelType, payload map[string]any) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.NewUnknownf("error building payload: %w", err)
	}
	return s.sendRaw(channelType, bodyBytes)
}

func (s *Client) sendRaw(channelType WebhookChannelType, bodyBytes []byte) error {
	// Check if the channel type is enabled
	webhookURL := s.getWebhookURL(channelType)
	if webhookURL == "" {
		return nil
	}

	resp, err := s.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return errors.Newf(
			errors.ErrCodeUnknown,
			"failed to send HTTP request to: %s, urlSHA256: %x, error: %w",
			webhookURL[0:len(webhookURL)/2]+"...",
			sha256.Sum256([]byte(webhookURL)),
			err,
		)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	// We only care about the response body if the request was not successful, but we still need to read it fully.
	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		if err != nil {
			body = []byte("<error reading body>")
		}
		return errors.NewUnknownf("slack request not successful, code: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

// getWebhookURL returns the webhook URL for the given channel type. If the URL is not configured, it will return the
// URL for the next channel type in the order of Error -> Warn -> Info. If none of the URLs are configured, it will
// panic.
func (s *Client) getWebhookURL(channelType WebhookChannelType) string {
	urls := s.conf.Slack.WebhookURLS
	switch channelType {
	case WebhookChannelTypeInfo:
		return urls.Info
	case WebhookChannelTypeWarn:
		return firstNonEmpty(urls.Warn, urls.Info)
	case WebhookChannelTypeError:
		return firstNonEmpty(urls.Error, urls.Warn, urls.Info)
	default:
		panic(errors.Newf(errors.ErrCodeBadState, "invalid SlackWebhookChannelType: %s", channelType))
	}
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

var fullMsgTemplateSrc = `{
	"blocks": [
		{
			"type": "header",
			"text": {
				"type": "plain_text",
				"text": "{{.level_msg}}: {{.service_name}}",
				"emoji": true
			}
		},
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": "{{.main_msg | js}}"
			}
		},
		${context_section}
	]
}`

var contextSectionTemplateSrc = `{
			"type": "divider"
		},
		{
			"type": "context",
			"elements": [
				{
					"type": "mrkdwn",
					"text": "*EnvType*\t\n{{.env_type}}"
				},
				{
					"type": "mrkdwn",
					"text": "*EnvName*\t\n{{.env_name}}"
				},
				{
					"type": "mrkdwn",
					"text": "*Host*\t\n{{.host}}"
				},
				{
					"type": "mrkdwn",
					"text": "*Release*\t\n{{.release}}"
				},
				{
					"type": "mrkdwn",
					"text": "*Commit*\t\n{{.commit}}"
				},
				{
					"type": "mrkdwn",
					"text": "*BuildTime*\t\n{{.build_time}}"
				}
			]
		},
		{
			"type": "divider"
		}`
