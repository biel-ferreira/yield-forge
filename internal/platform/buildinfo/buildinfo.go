// Package buildinfo exposes build metadata for the /version endpoint.
//
// The variables are overridden at build time via linker flags, e.g.:
//
//	go build -ldflags "\
//	  -X github.com/biel-ferreira/yield-forge/internal/platform/buildinfo.Version=0.1.0 \
//	  -X github.com/biel-ferreira/yield-forge/internal/platform/buildinfo.Commit=$(git rev-parse --short HEAD) \
//	  -X github.com/biel-ferreira/yield-forge/internal/platform/buildinfo.BuildTime=$(date -u +%FT%TZ)"
//
// The defaults below are used in development when no flags are passed.
package buildinfo

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

// Info is a snapshot of build metadata.
type Info struct {
	Version   string
	Commit    string
	BuildTime string
}

// Get returns the current build metadata.
func Get() Info {
	return Info{Version: Version, Commit: Commit, BuildTime: BuildTime}
}
