package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded frontend filesystem.
// In development, dist/ may not exist; callers should handle this gracefully.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
