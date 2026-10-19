package db

import "github.com/jatassi/slipstream/internal/storage"

type RootID struct{ storage.UUID }

// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
type Singleton int64

// VolumeID is a volume's id as its Node reports it: a filesystem id or a
// volume serial.
type VolumeID string
