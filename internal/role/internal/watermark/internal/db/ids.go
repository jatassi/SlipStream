package db

import "github.com/jatassi/slipstream/internal/storage"

// NodeID is role.NodeID, which this package cannot import.
type NodeID struct{ storage.UUID }
