package db

// Singleton is the id of a one-row table, fixed at 1 by a CHECK.
type Singleton int64

// StepID is a first setup step's id, as firstrun.StepID names it.
type StepID string

// Outcome is "done" or "skipped", as firstrun.Outcome names it.
type Outcome string
