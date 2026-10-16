package db

// TaskID is a Scheduled task's stable id: its row key and URL segment.
type TaskID string

type RunID int64

// UnitKey names the object a unit acts on, such as "node:<uuid>".
type UnitKey string

// Overdue is what a task does when the clock finds its slot passed.
type Overdue string

// Trigger is what opened a run.
type Trigger string

// Outcome is how one unit of a run ended.
type Outcome string

// Result is how a run ended.
type Result string

// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
type Singleton int64
