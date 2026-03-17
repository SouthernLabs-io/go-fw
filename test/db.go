package test

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/southernlabs-io/go-fw/config"
)

var dbNameReplacer = strings.NewReplacer(
	" ", "_",
	"-", "_",
	"test", "",
)

func CreateTestDBName(conf config.Config) string {
	// Postgres max length for db name is 63
	const maxLen = 63
	prefix := strings.Trim(dbNameReplacer.Replace(strings.ToLower(conf.Name)), "_")
	envName := strings.Trim(dbNameReplacer.Replace(strings.ToLower(conf.Env.Name)), "_")
	if envName == "" {
		envName = "test"
	}

	const hashLen = 16
	hashStr := fmt.Sprintf("%x", sha256.Sum256([]byte(conf.Name)))
	availablePrefixLen := maxLen - len(envName) - hashLen - 2
	if availablePrefixLen < 0 {
		availablePrefixLen = 0
	}
	if len(prefix) > availablePrefixLen {
		prefix = strings.Trim(prefix[:availablePrefixLen], "_")
	}

	if prefix == "" {
		return fmt.Sprintf("%s_%s", hashStr[:hashLen], envName)
	}

	return fmt.Sprintf("%s_%s_%s", prefix, hashStr[:hashLen], envName)
}
