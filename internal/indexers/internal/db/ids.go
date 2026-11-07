package db

import "github.com/jatassi/slipstream/internal/storage"

type (
	IndexerID struct{ storage.UUID }
	TrackerID struct{ storage.UUID }
	QueryID   struct{ storage.UUID }
)

type (
	Singleton    int64
	DefinitionID string
)
