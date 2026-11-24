package db

import "github.com/jatassi/slipstream/internal/storage"

// Singleton is the id of moves_settings' one row.
type Singleton int64

type BatchID struct{ storage.UUID }

type UnitID struct{ storage.UUID }
