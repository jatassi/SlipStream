package db

import "github.com/jatassi/slipstream/internal/storage"

type (
	GrabID   struct{ storage.UUID }
	SearchID struct{ storage.UUID }
)

// Stage is where a Grab is in the pipeline.
type Stage string

const (
	Fetching     Stage = "fetching"
	Downloading  Stage = "downloading"
	Probing      Stage = "probing"
	Transferring Stage = "transferring"
	Importing    Stage = "importing"
	Imported     Stage = "imported"
	Failed       Stage = "failed"
)

// Code is why a Grab failed.
type Code string

// Outcome is how a search run ended.
type Outcome string
