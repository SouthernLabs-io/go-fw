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
	s := dbNameReplacer.Replace(strings.ToLower(conf.Name))
	parts := strings.Split(s, "_")
	// Account for the extra "_" between parts
	maxCutLen := (maxLen - len(parts)) / len(parts)
	s = ""
	for _, part := range parts {
		if len(part) > maxCutLen {
			s += part[0:maxCutLen]
		} else {
			s += part
		}
		s += "_"
	}

	return fmt.Sprintf(
		"%s%.*x_%s",
		s,
		// Plus one to account for the final "_"
		// Each byte uses 2 characters, so we need to divide by 2
		(maxLen-(len(s)+len(conf.Env.Name)+1))/2,
		sha256.Sum256([]byte(conf.Name)),
		conf.Env.Name,
	)
}
