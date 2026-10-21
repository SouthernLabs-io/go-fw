package database

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
)

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
		{name: "CA with empty mode", conf: config.DatabaseConfig{SSLRootCert: validCA}, wantErr: "requires database.sslMode to enable TLS"},
		{name: "CA with disable", conf: config.DatabaseConfig{SSLMode: "disable", SSLRootCert: validCA}, wantErr: "requires database.sslMode to enable TLS"},
		{
			name:    "missing CA file",
			conf:    config.DatabaseConfig{SSLMode: "verify-full", SSLRootCert: filepath.Join(t.TempDir(), "missing.pem")},
			wantErr: "could not read database.sslRootCert",
		},
		{name: "CA without certificate", conf: config.DatabaseConfig{SSLMode: "verify-full", SSLRootCert: notPEM}, wantErr: "contains no PEM certificate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLS(tt.conf)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
