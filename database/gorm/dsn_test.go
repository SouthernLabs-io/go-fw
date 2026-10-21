package databasegorm

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
)

func TestPostgresDSN_DefaultKeepsDriverSSLMode(t *testing.T) {
	cfg, err := pgx.ParseConfig(postgresDSN(config.DatabaseConfig{
		Host: "localhost",
		Port: 5432,
		User: "postgres",
		Pass: "password",
	}, "app", "password"))
	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.Host)
	require.Equal(t, "app", cfg.Database)
	// pgx's default sslmode is prefer: it tries TLS first and falls back to plaintext.
	require.NotNil(t, cfg.TLSConfig)
	require.Len(t, cfg.Fallbacks, 1)
	require.Nil(t, cfg.Fallbacks[0].TLSConfig)
}

func TestPostgresDSN_EncodesCredentials(t *testing.T) {
	password := `p'ss\ word "x"=y`
	user := `user'name`
	cfg, err := pgx.ParseConfig(postgresDSN(config.DatabaseConfig{
		Host:    "2001:db8::1",
		Port:    5432,
		User:    user,
		Pass:    password,
		SSLMode: "verify-full",
	}, "app", password))
	require.NoError(t, err)
	require.Equal(t, "2001:db8::1", cfg.Host)
	require.Equal(t, uint16(5432), cfg.Port)
	require.Equal(t, user, cfg.User)
	require.Equal(t, password, cfg.Password)
	require.Equal(t, "app", cfg.Database)
	require.NotNil(t, cfg.TLSConfig)
	require.Equal(t, "2001:db8::1", cfg.TLSConfig.ServerName)
	require.False(t, cfg.TLSConfig.InsecureSkipVerify)
	require.Empty(t, cfg.Fallbacks)
}

func TestPostgresDSN_SetsSSLRootCert(t *testing.T) {
	rootCert := `/etc/postgres/certs/root ca/#1's.pem`
	dsn := postgresDSN(config.DatabaseConfig{
		Host: "localhost", Port: 5432, User: "postgres", SSLMode: "verify-full", SSLRootCert: rootCert,
	}, "app", "password")
	require.Contains(t, dsn, ` sslmode='verify-full' sslrootcert='/etc/postgres/certs/root ca/#1\'s.pem'`)
}

func TestQuoteDSNValue(t *testing.T) {
	require.Equal(t, `'plain'`, quoteDSNValue("plain"))
	require.Equal(t, `'it\'s'`, quoteDSNValue("it's"))
	require.Equal(t, `'back\\slash'`, quoteDSNValue(`back\slash`))
}

func TestPostgresDSN_RedactedPasswordIsNotInDSN(t *testing.T) {
	password := `p'ssw0rd\secret`
	dsn := postgresDSN(config.DatabaseConfig{Host: "localhost", Port: 5432, User: "postgres", Pass: password}, "app", "*")
	require.NotContains(t, dsn, "ssw0rd")
	cfg, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	require.Equal(t, "*", cfg.Password)
}
