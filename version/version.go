package version

// If you want to include these values as defaults at compile time, you can use ldflags:
// -X github.com/southernlabs-io/go-fw/version.BuildTime=$(date -u '+%Y-%m-%dT%H-%M-%SZ')
// -X github.com/southernlabs-io/go-fw/version.Commit=$(git rev-parse --short HEAD)
// -X github.com/southernlabs-io/go-fw/version.Release=$(git describe --tags --always --dirty)
var (
	// BuildTime is a string timestamp of the binary build time.
	BuildTime = ""

	// Commit is the git commit hash of the binary
	Commit = "dirty"

	// Release is a semantic version tag of the binary.
	Release = "v0.1.0"

	// Prerelease is a semantic version prerelease tag of the binary.
	Prerelease = ""
)

// SemVer is a valid Semantic Version string which value is: "{Release}+{Commit}.{BuildTime}"
var SemVer string

func init() {
	SemVer = Release
	if SemVer[0] != 'v' {
		SemVer = "v" + SemVer
	}

	// This can be true when the value is set at build time
	if Prerelease != "" {
		SemVer += "-" + Prerelease
	}

	// This can be true when the value is set at build time
	if Commit != "" || BuildTime != "" {
		if Commit != "" {
			SemVer += "+" + cleanMeta(Commit)
		} else {
			SemVer += "+unset"
		}
		if BuildTime != "" {
			SemVer += "." + cleanMeta(BuildTime)
		}
	}
}

func cleanMeta(str string) string {
	// Replace invalid characters
	res := str
	for i := 0; i < len(str); i++ {
		if !isIdentChar(str[i]) {
			res = res[:i] + "-" + res[i+1:]
		}
	}
	return res
}

// Copy from semver package
func isIdentChar(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '-'
}
