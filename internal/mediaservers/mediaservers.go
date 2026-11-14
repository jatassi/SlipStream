// Package mediaservers is the Controller's side of S35 (Media Servers): the
// server records, their Path mappings, the Plex account, the durable
// per-server queue of Media Server changes, play history and the path
// index, all in library.db.
package mediaservers

import (
	"embed"

	"github.com/jatassi/slipstream/internal/storage"
)

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema runs after roots and files, whose rows it references.
func Schema() storage.Schema[storage.Library] {
	return storage.NewSchema[storage.Library]("mediaservers", schemaFiles)
}
