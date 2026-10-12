package db

import "github.com/jatassi/slipstream/internal/storage"

type AccountID struct{ storage.UUID }

type SessionID struct{ storage.UUID }

// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
type Singleton int64
