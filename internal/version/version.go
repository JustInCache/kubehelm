// Package version exposes build-time metadata injected via -ldflags.
// Default values are used when the binary is built without ldflags (dev mode).
package version

import "runtime"

var (
	// Version is the semantic version tag (e.g. "v0.3.1").
	// Set at build time: -ldflags "-X github.com/ankushko/k8s-project-revamp/internal/version.Version=v0.3.1"
	Version = "dev"

	// Commit is the short Git SHA of the build.
	Commit = "unknown"

	// BuildDate is the UTC timestamp of the build (RFC 3339).
	BuildDate = "unknown"
)

// Info bundles all build-time metadata into a JSON-serialisable struct.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the current build info.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}
