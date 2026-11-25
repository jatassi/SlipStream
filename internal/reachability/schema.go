package reachability

import (
	"embed"

	"github.com/jatassi/slipstream/internal/storage"
)

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is reachability's tables. It is listed after mediaservers, whose
// rows its Exposures and dismissals cascade from.
func Schema() storage.Schema[storage.Library] {
	return storage.NewSchema[storage.Library]("reachability", schemaFiles)
}
