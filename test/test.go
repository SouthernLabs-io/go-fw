package test

import "testing"

// IntegrationTest will skip if this is a testing.Short test run
func IntegrationTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
}
