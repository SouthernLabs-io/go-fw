package requestlog

import (
	"net/http"
	"net/url"

	"github.com/southernlabs-io/go-fw/config"
)

// Metadata contains request fields that are safe for request logging.
type Metadata struct {
	Path     string
	Query    url.Values
	Referrer string
}

// Policy applies the request logger's query and referrer policy.
type Policy struct {
	queryParameterAllowlist map[string]struct{}
	referrerMode            config.RequestLoggerReferrerMode
}

func NewPolicy(conf config.RequestLoggerConfig) Policy {
	allowlist := make(map[string]struct{}, len(conf.QueryParameterAllowlist))
	for _, parameter := range conf.QueryParameterAllowlist {
		if parameter != "" {
			allowlist[parameter] = struct{}{}
		}
	}

	referrerMode := config.RequestLoggerReferrerOmit
	if conf.ReferrerMode == config.RequestLoggerReferrerOriginPath {
		referrerMode = conf.ReferrerMode
	}

	return Policy{
		queryParameterAllowlist: allowlist,
		referrerMode:            referrerMode,
	}
}

func (p Policy) Metadata(r *http.Request) Metadata {
	if r == nil {
		return Metadata{}
	}

	metadata := Metadata{}
	if r.URL != nil {
		metadata.Path = r.URL.Path
		metadata.Query = p.allowedQuery(r.URL.RawQuery)
	}
	if p.referrerMode == config.RequestLoggerReferrerOriginPath {
		metadata.Referrer = originPath(r.Referer())
	}
	return metadata
}

func (p Policy) allowedQuery(rawQuery string) url.Values {
	if rawQuery == "" || len(p.queryParameterAllowlist) == 0 {
		return nil
	}

	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return nil
	}

	allowed := make(url.Values)
	for key, values := range query {
		if _, ok := p.queryParameterAllowlist[key]; !ok {
			continue
		}
		allowed[key] = append([]string(nil), values...)
	}
	if len(allowed) == 0 {
		return nil
	}
	return allowed
}

func originPath(rawReferrer string) string {
	if rawReferrer == "" {
		return ""
	}

	referrer, err := url.Parse(rawReferrer)
	if err != nil || referrer.Scheme == "" || referrer.Host == "" {
		return ""
	}

	return (&url.URL{
		Scheme:  referrer.Scheme,
		Host:    referrer.Host,
		Path:    referrer.Path,
		RawPath: referrer.RawPath,
	}).String()
}
