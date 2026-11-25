package db

import "github.com/jatassi/slipstream/internal/storage"

// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
type Singleton int64

// AccessID is a Member's access to one Media Server.
type AccessID struct{ storage.UUID }

// ProblemCode is the last failed call on an access row, one of the codes
// mediaservers_member_access.problem_code's CHECK lists.
type ProblemCode string
