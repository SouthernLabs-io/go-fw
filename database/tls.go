package database

import (
	"crypto/x509"
	"os"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/errors"
)

// sslModes are the sslmode values supported by both PostgreSQL drivers (pgdriver and pgx).
var sslModes = map[string]bool{
	"disable": true, "allow": true, "prefer": true, "require": true, "verify-ca": true, "verify-full": true,
}

// ValidateTLS rejects settings the drivers would panic on or silently ignore: an unknown sslMode, an
// sslRootCert without an sslMode that enables TLS, or a CA file that is unreadable or holds no
// certificate. An empty sslMode is valid and keeps each connector's default.
func ValidateTLS(dbConf config.DatabaseConfig) error {
	if dbConf.SSLMode != "" && !sslModes[dbConf.SSLMode] {
		return errors.NewBadStatef("unsupported database.sslMode %q", dbConf.SSLMode)
	}
	if dbConf.SSLRootCert == "" {
		return nil
	}
	if dbConf.SSLMode == "" || dbConf.SSLMode == "disable" {
		return errors.NewBadStatef("database.sslRootCert requires database.sslMode to enable TLS (e.g. verify-full), got %q", dbConf.SSLMode)
	}
	pem, err := os.ReadFile(dbConf.SSLRootCert)
	if err != nil {
		return errors.NewBadStatef("could not read database.sslRootCert: %w", err)
	}
	if !x509.NewCertPool().AppendCertsFromPEM(pem) {
		return errors.NewBadStatef("database.sslRootCert %q contains no PEM certificate", dbConf.SSLRootCert)
	}
	return nil
}
