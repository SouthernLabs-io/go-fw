package databasegorm

import (
	"io"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/log"
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

// localPostgresConfig points at the fw_postgres service of test/testdata/docker-compose.yaml (no TLS).
func localPostgresConfig(sslMode string) config.Config {
	return config.Config{Database: config.DatabaseConfig{
		Host: "localhost", Port: 5433, User: "postgres", Pass: "postgres", SSLMode: sslMode,
	}}
}

func TestMustOpenGORM_ConnectsToLocalPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	lf := log.NewLoggerFactoryWithWriter(config.GetRootConfig(), io.Discard)
	// An empty sslMode keeps the pgx default (prefer), which falls back to plaintext on a server without TLS.
	for _, sslMode := range []string{"", "disable", "prefer"} {
		t.Run("sslMode="+sslMode, func(t *testing.T) {
			db := MustOpenGORM(localPostgresConfig(sslMode), "postgres", lf)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			defer sqlDB.Close()
			var one int
			require.NoError(t, sqlDB.QueryRow("SELECT 1").Scan(&one))
			require.Equal(t, 1, one)
		})
	}
}

func TestMustOpenGORM_VerifiedTLSPanicsAgainstServerWithoutTLS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	lf := log.NewLoggerFactoryWithWriter(config.GetRootConfig(), io.Discard)
	defer func() {
		err, ok := recover().(error)
		require.True(t, ok, "MustOpenGORM should panic with an error")
		require.NotContains(t, err.Error(), "postgres:postgres")
		require.NotContains(t, err.Error(), "password='postgres'")
		require.Contains(t, err.Error(), "could not connect to DB")
	}()
	MustOpenGORM(localPostgresConfig("verify-full"), "postgres", lf)
}
