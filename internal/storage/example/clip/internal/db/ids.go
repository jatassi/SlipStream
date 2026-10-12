package db

import "github.com/jatassi/slipstream/internal/storage"

// ClipID is clip.id: a local integer rowid. clip exports it as clip.ID.
type ClipID int64

// ClipPublicID is clip.public_id: it appears in URLs, so it is a UUIDv7.
// Embedding promotes storage.UUID's Scan and Value.
type ClipPublicID struct{ storage.UUID }
