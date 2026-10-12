//go:build !release

// Package web holds the production web build, embedded only in release builds.
package web

import (
	"embed"
	"io/fs"
)

// Dist is empty outside release builds, so go vet and go test need no web build.
var Dist fs.FS = embed.FS{}
