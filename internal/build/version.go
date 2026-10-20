// Package build holds what is decided when the binary is built.
package build

// Version is set at build time:
// go build -ldflags "-X github.com/kaelence-01/draco-hq/internal/build.Version=v0.1.0"
var Version = "dev"
