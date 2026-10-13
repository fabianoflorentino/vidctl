// Package version holds the version baked into the binary at build time. The
// Makefile and the release workflow pass it through -ldflags -X; local builds
// (go run, tests, CI on branches) fall back to "dev".
package version

// Version is the semantic version of this binary, e.g. "v2.5.0".
var Version = "dev"
