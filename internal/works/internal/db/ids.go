package db

import "github.com/jatassi/slipstream/internal/storage"

type (
	WorkID  struct{ storage.UUID }
	LevelID struct{ storage.UUID }
	ItemID  struct{ storage.UUID }
)

// TypeID is a Content Type's id as works_work.content_type holds it.
type TypeID string

// Key is a node's identity at its metadata source.
type Key string
