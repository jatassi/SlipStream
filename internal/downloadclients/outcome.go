package downloadclients

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// Outcome is what a downloading Grab does about its Torrent: Pending
// waits, Completed moves on, Failed and Gone fail it.
type Outcome interface{ outcome() }

type (
	Pending   struct{}
	Completed struct{}
	// Failed is the latest report saying the engine failed the Torrent.
	Failed struct{ Error string }
	// Gone is a removed Torrent, Removing or Parted.
	Gone struct{ Reason RemoveReason }
)

func (Pending) outcome()   {}
func (Completed) outcome() {}
func (Failed) outcome()    {}
func (Gone) outcome()      {}

// Attached is a Torrent as the Grabs attached to it read it.
type Attached struct {
	InfoHash bencode.InfoHash
	// Files are its files from its first complete report, nil before.
	// They stay once it is removed, for a Grab already past downloading.
	Files   []engine.File
	Outcome Outcome
	// Progress is nil before the Node's first report.
	Progress *Progress
	// Fault is the Torrent's release fault, nil for none.
	Fault *Fault
}

// Progress is a Torrent's bytes done, as its Node last reported them, of
// its size.
type Progress struct{ Done, Size int64 }

// Outcomes is each of ids' Torrents as its Grabs read it, in r's
// snapshot. The precedence is StateOf's: Gone, then Failed, then
// Completed, so a removal that lands before a Grab advances cancels it.
func Outcomes(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID]Attached, error) {
	rows, err := db.New(r).ListAttached(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[ID]Attached, len(rows))
	for _, row := range rows {
		a, err := attachedOf(row)
		if err != nil {
			return nil, fmt.Errorf("torrent %s: %w", row.ID, err)
		}
		out[row.ID] = a
	}
	return out, nil
}

func attachedOf(row db.ListAttachedRow) (Attached, error) {
	a := Attached{InfoHash: bencode.InfoHash(row.InfoHash), Outcome: Pending{}}
	if row.Completion != nil {
		var c Completion
		if err := json.Unmarshal([]byte(*row.Completion), &c); err != nil {
			return Attached{}, fmt.Errorf("completion: %w", err)
		}
		a.Files, a.Outcome = c.Files, Completed{}
	}
	rep, err := reportOf(row.Report)
	if err != nil {
		return Attached{}, err
	}
	if rep != nil {
		a.Progress = &Progress{Done: rep.done(row.Size), Size: row.Size}
		if rep.Engine == EngineFailed {
			a.Outcome = Failed{Error: rep.Error}
		}
	}
	if row.RemovedReason != nil {
		a.Outcome = Gone{Reason: RemoveReason(*row.RemovedReason)}
	}
	if row.Fault != nil {
		a.Fault = &Fault{Kind: FaultKind(*row.Fault), At: row.FaultAt.Time()}
	}
	return a, nil
}
