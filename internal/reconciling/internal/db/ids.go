package db

// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
type Singleton int64

// RestoreID is the id of the restore whose Write began Reconciling.
type RestoreID string
