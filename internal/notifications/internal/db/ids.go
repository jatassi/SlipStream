package db

import "github.com/jatassi/slipstream/internal/storage"

type ConnectionID struct{ storage.UUID }

// Seq is a bell row's place in the list, taken when it is shown.
type Seq int64

// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
type Singleton int64
