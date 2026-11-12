package downloadclients

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// Outcome is what a downloading Grab does about its Torrent: Pending
// waits, Completed moves on, Failed, Errored and Gone fail it.
type Outcome interface{ outcome() }

type (
	Pending   struct{}
	Completed struct{}
	// Failed is the latest report saying the engine failed the Torrent.
	Failed struct{ Error string }
	// Errored is an external client's error that its reports kept
	// carrying for longer than After, the ClientErrorFail threshold.
	Errored struct{ After time.Duration }
	// Gone is a removed Torrent, Removing or Parted. Client is its
	// client's name for ByClient, ClientDeleted and ReleaseFaulted, "" for
	// the built-in client and the other reasons. By is the name of the
	// Admin who removed it for ByAdmin and ByMarkFailed, "" for the API key
	// and the other reasons. Fault is its release fault for
	// ReleaseFaulted, nil otherwise.
	Gone struct {
		Reason RemoveReason
		Client string
		By     string
		Fault  *FaultKind
	}
)

func (Pending) outcome()   {}
func (Completed) outcome() {}
func (Failed) outcome()    {}
func (Errored) outcome()   {}
func (Gone) outcome()      {}

// Attached is a Torrent as the Grabs attached to it read it.
type Attached struct {
	InfoHash bencode.InfoHash
	// Client is nil for the built-in client.
	Client *ClientRef
	// Files are its files from its first complete report, nil before.
	// They stay once it is removed, for a Grab already past downloading.
	Files   []engine.File
	Outcome Outcome
	// Progress is nil before the Node's first report.
	Progress *Progress
	// Fault is the Torrent's release fault, nil for none.
	Fault *Fault
	// ClientError is the error its external client reports, nil while the
	// last report had none.
	ClientError *ClientError
	// ImportDelayUntil is when its external client's Import delay ends,
	// while it seeds short of its Completion; zero otherwise.
	ImportDelayUntil time.Time
}

// ClientError is an external client's error on a Torrent. Node names the
// client's Node. FailsAt is when its Grab fails if reports keep carrying
// the error: the clock counts only while they do.
type ClientError struct {
	Message string
	Node    string
	FailsAt time.Time
}

// Progress is a Torrent's bytes done, as its Node last reported them, of
// its size.
type Progress struct{ Done, Size int64 }

// Outcomes is each of ids' Torrents as its Grabs read it, in r's
// snapshot. The precedence is StateOf's: Gone, then Failed, then
// Completed, so a removal that lands before a Grab advances cancels it. A
// client error that outlasted ClientErrorFail is Errored, in place of
// Pending.
func Outcomes(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID]Attached, error) {
	rows, err := db.New(r).ListAttached(ctx, ids)
	if err != nil {
		return nil, err
	}
	th, err := ThresholdsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	var names map[role.NodeID]string
	if slices.ContainsFunc(rows, func(row db.ListAttachedRow) bool { return !row.ClientErrorAt.IsZero() }) {
		if names, err = namesIn(ctx, r); err != nil {
			return nil, err
		}
	}
	out := make(map[ID]Attached, len(rows))
	for _, row := range rows {
		a, err := attachedOf(row, names[row.NodeID], th.ClientErrorFail)
		if err != nil {
			return nil, fmt.Errorf("torrent %s: %w", row.ID, err)
		}
		out[row.ID] = a
	}
	return out, nil
}

// reported reads what rep, row's latest report, says: progress, an engine
// failure, the Import delay and the client's error on node, which fails
// a once its clock passed fail.
func (a *Attached) reported(row db.ListAttachedRow, rep *Report, node string, fail time.Duration) {
	a.Progress = &Progress{Done: rep.done(row.Size), Size: row.Size}
	if rep.Engine == EngineFailed {
		a.Outcome = Failed{Error: rep.Error}
	}
	if row.Completion == nil {
		a.ImportDelayUntil = rep.ImportAt
	}
	if row.ClientErrorAt.IsZero() {
		return
	}
	counted := time.Duration(row.ClientErrorMs) * time.Millisecond
	a.ClientError = &ClientError{Message: rep.Error, Node: node, FailsAt: row.ClientErrorAt.Time().Add(fail - counted)}
	if counted >= fail && a.Outcome == Outcome(Pending{}) {
		a.Outcome = Errored{After: fail}
	}
}

// attachedOf is row as its Grabs read it, its client's error on node
// failing it once the error outlasted fail.
func attachedOf(row db.ListAttachedRow, node string, fail time.Duration) (Attached, error) {
	a := Attached{InfoHash: bencode.InfoHash(row.InfoHash), Client: refOf(row.ClientID, row.ClientKind, row.ClientName), Outcome: Pending{}}
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
		a.reported(row, rep, node, fail)
	}
	if row.RemovedReason != nil {
		gone := Gone{Reason: RemoveReason(*row.RemovedReason)}
		if gone.Reason == ByClient || gone.Reason == ClientDeleted || gone.Reason == ReleaseFaulted {
			gone.Client = row.ClientName
		}
		gone.By = row.RemovedByName.String
		if gone.Reason == ReleaseFaulted {
			gone.Fault = (*FaultKind)(row.Fault)
		}
		a.Outcome = gone
	}
	if row.Fault != nil {
		a.Fault = &Fault{Kind: FaultKind(*row.Fault), At: row.FaultAt.Time()}
	}
	return a, nil
}
