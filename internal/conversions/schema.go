package conversions

import (
	"embed"

	"github.com/jatassi/slipstream/internal/storage"
)

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is conversions' library.db schema set. Its rows name Works,
// Targets, Profiles, roots, Nodes and files, and imports' rows name its
// Conversions, so it is listed after files and before imports.
func Schema() storage.Schema[storage.Library] {
	return storage.NewSchema[storage.Library]("conversions", schemaFiles)
}
