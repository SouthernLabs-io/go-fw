package requestlog_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/internal/requestlog"
)

func TestPolicy_DefaultOmitsQueryAndReferrer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/download?token=secret&view=summary", nil)
	req.Header.Set("Referer", "https://example.com/editor?token=referer-secret#details")

	metadata := (requestlog.Policy{}).Metadata(req)

	require.Equal(t, "/download", metadata.Path)
	require.Nil(t, metadata.Query)
	require.Empty(t, metadata.Referrer)
}

func TestPolicy_AllowlistReturnsOnlyApprovedQueryParameters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search?view=summary&view=details&token=secret&encoded=hello%20world", nil)

	metadata := requestlog.NewPolicy(config.RequestLoggerConfig{
		QueryParameterAllowlist: []string{"view", "encoded"},
	}).Metadata(req)

	require.Equal(t, "/search", metadata.Path)
	require.Equal(t, []string{"summary", "details"}, metadata.Query["view"])
	require.Equal(t, []string{"hello world"}, metadata.Query["encoded"])
	require.NotContains(t, metadata.Query, "token")
}

func TestPolicy_MalformedQueryOmitsAllQueryData(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/download?view=summary&token=secret%ZZ", nil)

	metadata := requestlog.NewPolicy(config.RequestLoggerConfig{
		QueryParameterAllowlist: []string{"view", "token"},
	}).Metadata(req)

	require.Nil(t, metadata.Query)
}

func TestPolicy_OriginPathReferrerOmitsSensitiveURLParts(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/editor", nil)
	req.Header.Set("Referer", "https://user:password@example.com:8443/editor%2Fdraft?token=secret#details")

	metadata := requestlog.NewPolicy(config.RequestLoggerConfig{
		ReferrerMode: config.RequestLoggerReferrerOriginPath,
	}).Metadata(req)

	require.Equal(t, "https://example.com:8443/editor%2Fdraft", metadata.Referrer)
}

func TestPolicy_InvalidReferrerModeOmitsReferrer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/editor", nil)
	req.Header.Set("Referer", "https://example.com/editor?token=secret")

	metadata := requestlog.NewPolicy(config.RequestLoggerConfig{
		ReferrerMode: "unsupported",
	}).Metadata(req)

	require.Empty(t, metadata.Referrer)
}

func TestPolicy_MalformedReferrerOmitsReferrer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/editor", nil)
	req.Header.Set("Referer", "https://example.com/%ZZ?token=referer-secret")

	metadata := requestlog.NewPolicy(config.RequestLoggerConfig{
		ReferrerMode: config.RequestLoggerReferrerOriginPath,
	}).Metadata(req)

	require.Empty(t, metadata.Referrer)
}
