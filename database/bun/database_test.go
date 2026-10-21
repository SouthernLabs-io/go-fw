package databasebun

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func writeTestCA(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "root.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	return path
}

func TestValidateTLS(t *testing.T) {
	validCA := writeTestCA(t)
	notPEM := filepath.Join(t.TempDir(), "not-pem.txt")
	require.NoError(t, os.WriteFile(notPEM, []byte("not a certificate"), 0o600))

	tests := []struct {
		name    string
		conf    config.DatabaseConfig
		wantErr string
	}{
		{name: "default is TLS off", conf: config.DatabaseConfig{}},
		{name: "verify-full without CA", conf: config.DatabaseConfig{SSLMode: "verify-full"}},
		{name: "verify-full with valid CA", conf: config.DatabaseConfig{SSLMode: "verify-full", SSLRootCert: validCA}},
		{name: "unsupported mode", conf: config.DatabaseConfig{SSLMode: "verify_full"}, wantErr: "unsupported database.sslMode"},
		{name: "CA with empty mode", conf: config.DatabaseConfig{SSLRootCert: validCA}, wantErr: "TLS would not be used"},
		{name: "CA with disable", conf: config.DatabaseConfig{SSLMode: "disable", SSLRootCert: validCA}, wantErr: "TLS would not be used"},
		{
			name:    "missing CA file",
			conf:    config.DatabaseConfig{SSLMode: "verify-full", SSLRootCert: filepath.Join(t.TempDir(), "missing.pem")},
			wantErr: "could not read database.sslRootCert",
		},
		{name: "CA without certificate", conf: config.DatabaseConfig{SSLMode: "verify-full", SSLRootCert: notPEM}, wantErr: "contains no PEM certificate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTLS(tt.conf)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
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
