package version_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"

	"github.com/southernlabs-io/go-fw/version"
)

func TestVersionIsSemVer(t *testing.T) {
	// It should be a valid semver
	require.True(t, semver.IsValid(version.SemVer))
	require.Equal(t, "v0", semver.Major(version.SemVer))
	require.Equal(t, "v0.1", semver.MajorMinor(version.SemVer))
	require.Equal(t, "v0.1.0", semver.Canonical(version.SemVer))
	require.Equal(t, "", semver.Prerelease(version.SemVer))
	require.Equal(t, "+dirty", semver.Build(version.SemVer))
}
