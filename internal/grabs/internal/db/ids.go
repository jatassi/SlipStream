package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
)

type (
	GrabID   struct{ storage.UUID }
	SearchID struct{ storage.UUID }
	ManualID struct{ storage.UUID }
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

// ReleaseOutcome is what a run did with one Release it listed.
type ReleaseOutcome string

// Step is one step of the ranking chain.
type Step string

// Source is how a Grab was chosen: by a search's decision, or by an Admin
// in a Manual search.
type Source string

// BlockCause is why a Release is on the Blocklist.
type BlockCause string

// Reading is a listed Release's title as release.Read read it when it was
// heard, kept as JSON.
type Reading struct{ release.Read }

func (r Reading) Value() (driver.Value, error) {
	b, err := json.Marshal(r.Read)
	return string(b), err
}

func (r *Reading) Scan(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("scan a title read from %T", v)
	}
	if err := json.Unmarshal([]byte(s), &r.Read); err != nil {
		return fmt.Errorf("scan a title read: %w", err)
	}
	return nil
}
