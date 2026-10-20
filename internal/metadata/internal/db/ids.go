package db

// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
type Singleton int64

// Reason is why a metadata_due row makes its Work due.
type Reason string

// Changed is a Work its source's changes list named.
const Changed Reason = "changed"
