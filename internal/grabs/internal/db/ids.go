package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/downloadclients"
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

// InFlight is the one answer to whether a Grab at s holds its cells: from
// fetching until it is imported or failed (catalog 2.4).
func (s Stage) InFlight() bool { return s != Imported && s != Failed }

// Use is the stage the client of a Grab's Torrent sees. A fetching Grab
// has no Torrent, so asking panics.
func (s Stage) Use() downloadclients.UseStage {
	switch s {
	case Fetching:
	case Downloading:
		return downloadclients.UseDownloading
	case Probing, Transferring, Importing:
		return downloadclients.UseImporting
	case Imported:
		return downloadclients.UseImported
	case Failed:
		return downloadclients.UseFailed
	}
	panic("grabs: a Grab " + string(s) + " has no Torrent")
}

// Settled is how a Grab's cell settled: a file placed into it, no file of
// the download filled it, or the Admin declined it.
type Settled string

const (
	SettledPlaced   Settled = "placed"
	SettledNone     Settled = "none"
	SettledDeclined Settled = "declined"
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
