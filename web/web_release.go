//go:build release

// Package web holds the production web build, embedded only in release builds.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist is the output of bun run build in web/.
var Dist, _ = fs.Sub(dist, "dist")
