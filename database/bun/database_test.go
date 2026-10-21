package databasebun

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
)

func TestPostgresDSN_DefaultsTLSOff(t *testing.T) {
	dsn, err := url.Parse(postgresDSN(config.DatabaseConfig{
		Host: "localhost",
		Port: 5432,
		User: "postgres",
		Pass: "password",
	}, "app"))
	require.NoError(t, err)
	require.Equal(t, "disable", dsn.Query().Get("sslmode"))
	require.Empty(t, dsn.Query().Get("sslrootcert"))
}

func TestPostgresDSN_EncodesTLSAndCredentials(t *testing.T) {
	password := "p@ss:/?#%word"
	rootCert := "/etc/postgres/certs/root ca/#1.pem?version=2"
	dsn, err := url.Parse(postgresDSN(config.DatabaseConfig{
		Host:        "2001:db8::1",
		Port:        5432,
		User:        "user@domain",
		Pass:        password,
		SSLMode:     "verify-full",
		SSLRootCert: rootCert,
	}, "app"))
	require.NoError(t, err)
	gotUser := dsn.User.Username()
	gotPassword, hasPassword := dsn.User.Password()
	require.True(t, hasPassword)
	require.Equal(t, "user@domain", gotUser)
	require.Equal(t, password, gotPassword)
	require.Equal(t, "[2001:db8::1]:5432", dsn.Host)
	require.Equal(t, "verify-full", dsn.Query().Get("sslmode"))
	require.Equal(t, rootCert, dsn.Query().Get("sslrootcert"))
}

func TestSanitizedDSN_DoesNotExposePassword(t *testing.T) {
	password := "p@ss:/?#%word"
	dsn := postgresDSN(config.DatabaseConfig{
		Host: "localhost",
		Port: 5432,
		User: "postgres",
		Pass: password,
	}, "app")
	sanitized := sanitizedDSN(dsn)
	require.NotContains(t, sanitized, password)
	require.NotContains(t, sanitized, url.QueryEscape(password))
	parsed, err := url.Parse(sanitized)
	require.NoError(t, err)
	require.Equal(t, "postgres", parsed.User.Username())
	_, hasPassword := parsed.User.Password()
	require.False(t, hasPassword)
}

func TestOpenSqlDB_ErrorDoesNotExposePassword(t *testing.T) {
	password := "secret@:/?#%word"
	_, err := OpenSqlDB(config.Config{Database: config.DatabaseConfig{
		Host: "127.0.0.1",
		Port: 1,
		User: "postgres",
		Pass: password,
	}}, "app", nil)
	require.Error(t, err)
	require.False(t, strings.Contains(err.Error(), password))
	require.False(t, strings.Contains(err.Error(), url.QueryEscape(password)))
}

func TestOpenSqlDB_InvalidTLSReturnsErrorInsteadOfPanicking(t *testing.T) {
	for _, dbConf := range []config.DatabaseConfig{
		{Host: "127.0.0.1", Port: 1, User: "postgres", Pass: "password", SSLMode: "verify_full"},
		{Host: "127.0.0.1", Port: 1, User: "postgres", Pass: "password", SSLMode: "verify-full", SSLRootCert: "/does/not/exist.pem"},
		{Host: "127.0.0.1", Port: 1, User: "postgres", Pass: "password", SSLRootCert: "/does/not/exist.pem"},
	} {
		require.NotPanics(t, func() {
			_, err := OpenSqlDB(config.Config{Database: dbConf}, "app", nil)
			require.Error(t, err)
			require.NotContains(t, err.Error(), "could not connect to DB")
		})
	}
}
