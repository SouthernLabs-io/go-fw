package version

import "fmt"

// If you want to include these values as defaults at compile time, you can use ldflags:
// -X github.com/southernlabs-io/go-fw/version.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
// -X github.com/southernlabs-io/go-fw/version.Commit=$(git rev-parse --short HEAD)
// -X github.com/southernlabs-io/go-fw/version.Release=$(git describe --tags --always --dirty)
// otherwise, they will default to "unset"
var (
	// BuildTime is a string timestamp of the binary build time.
	BuildTime = "unset"
	// Commit is a last commit hash when the binary was built.
	Commit = "unset"
	// Release is a semantic version of current build.
	Release = "unset"
)

var Full = fmt.Sprintf("%s+%s.%s", Release, Commit, BuildTime)
