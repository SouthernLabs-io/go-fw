package config

import (
	"os"
	"sync"
)

// MustSetenv will set the env and panic if there was an error
func MustSetenv(key, value string) {
	err := os.Setenv(key, value)
	if err != nil {
		panic("failed to set env: " + key)
	}
}

// CachedHostname returns the hostname of the machine or "unknown-hostname" if there was an error
// It is quite slow to call os.Hostname() so we cache the result.
var CachedHostname = sync.OnceValue(func() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-hostname"
	}
	return hostname
})
