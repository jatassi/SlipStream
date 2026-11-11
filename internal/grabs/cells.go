package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// GrabCells is files.Inflight over the Grabs: each cell is in flight while
// any unfinished Grab holds it, fetching included, and reads from the
// latest such Grab, whatever finished after it. Else it reads from its
// latest Grab, failed once failed. A failure that doesn't need the Admin,
// such as a cancel, leaves the cell as no Grab would, as an import does.
// An unfinished Grab whose Node is out of step waits on it, node_away, as
// does a fetching one whose route names a client on a Node out of step.
// A fetching one pinned to a client that can't take it waits on that
// client, client_off. One whose Torrent's client is away while its Node is
// in step waits on the client, client_away.
type GrabCells struct {
	InStep func(role.NodeID) bool
	// Router chooses the client each Grab's download goes to.
	Router Router
}

const (
	// NodeAway is the Waiting code of a Grab whose Node is out of step.
	NodeAway = "node_away"
	// ClientAway is the Waiting code of a Grab whose Torrent's client
	// isn't responding, refused the sign-in or needs attention.
	ClientAway = "client_away"
	// ClientOff is the Waiting code of a fetching Grab pinned to a client
	// that can't take it: it is off or deleted, or its Node doesn't hold
	// Downloading.
	ClientOff = "client_off"
)

func (g GrabCells) GrabCells(ctx context.Context, r storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[files.CellKey]files.GrabCell, error) {
	q := db.New(r)
	rows, err := works.Read(ws, func() ([]db.ListTargetsOfWorksRow, error) {
		rows, err := q.ListEveryGrabTarget(ctx)
		out := make([]db.ListTargetsOfWorksRow, len(rows))
		for i, row := range rows {
			out[i] = db.ListTargetsOfWorksRow(row)
		}
		return out, err
	}, func(ids []works.WorkID) ([]db.ListTargetsOfWorksRow, error) { return q.ListTargetsOfWorks(ctx, ids) })
	if err != nil {
		return nil, err
	}
	waits, err := g.waits(ctx, r)
	if err != nil {
		return nil, err
	}
	out := map[files.CellKey]files.GrabCell{}
	for _, row := range rows {
		key := files.CellKey{Item: row.ItemID, Edition: row.Edition, Target: row.TargetID}
		cell := files.GrabCell{Grab: row.ID.UUID}
		switch row.Stage {
		case Imported, Failed:
			if out[key].InFlight() {
				continue
			}
			if row.Stage == Imported || !(Grab{Stage: Failed, Failed: &Failure{Code: *row.FailureCode}, Source: row.Source}).NeedsYou() {
				delete(out, key)
				continue
			}
			cell.State, cell.FailedAt = files.GrabFailed, string(*row.FailedStage)
		case Fetching, Downloading, Probing, Transferring, Importing:
			cell.State = files.GrabInFlight
			if g.InStep(row.NodeID) {
				cell.Waiting = waits[row.ID]
			} else {
				cell.Waiting = NodeAway
			}
		}
		out[key] = cell
	}
	return out, nil
}

func (GrabCells) ReleaseTitles(ctx context.Context, r storage.Reader[storage.Library]) (map[storage.UUID]string, error) {
	rows, err := db.New(r).ListReleaseTitles(ctx)
	out := make(map[storage.UUID]string, len(rows))
	for _, row := range rows {
		out[row.ID.UUID] = row.ReleaseTitle
	}
	return out, err
}

// waits is what each unfinished Grab waits on while its own Node is in
// step: a fetching one's pinned client that can't take it, client_off,
// or its route's Node out of step, node_away, or its Torrent's away
// client, client_away.
func (g GrabCells) waits(ctx context.Context, r storage.Reader[storage.Library]) (map[GrabID]string, error) {
	open, err := db.New(r).ListOpenGrabs(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]downloadclients.ID, len(open))
	for i, gr := range open {
		ids[i] = gr.TorrentID
	}
	away, err := downloadclients.AwayIn(ctx, r, ids, g.InStep)
	if err != nil {
		return nil, err
	}
	out := map[GrabID]string{}
	for _, row := range open {
		if row.Stage == Fetching {
			grab, err := load(ctx, r, row, related{})
			if err != nil {
				return nil, err
			}
			w, err := clientWait(ctx, r, g.Router, g.InStep, grab)
			switch {
			case err != nil:
				return nil, err
			case w != nil && w.Off:
				out[row.ID] = ClientOff
				continue
			case w != nil:
				out[row.ID] = NodeAway
				continue
			}
		}
		if away[row.TorrentID] {
			out[row.ID] = ClientAway
		}
	}
	return out, nil
}

// Unfinished lists every Grab not yet imported or failed.
func (GrabCells) Unfinished(ctx context.Context, r storage.Reader[storage.Library]) ([]storage.UUID, error) {
	rows, err := db.New(r).ListOpenGrabs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]storage.UUID, len(rows))
	for i, row := range rows {
		out[i] = row.ID.UUID
	}
	return out, nil
}

// Settle records u on each unfinished Grab of work all of whose cells are
// in gone. Settling one twice changes nothing.
func (GrabCells) Settle(ctx context.Context, tx *storage.Tx[storage.Library], work works.WorkID, gone map[files.CellKey]bool, u files.Unwanted) (int, error) {
	q := db.New(tx)
	rows, err := q.ListTargetsOfWork(ctx, work)
	if err != nil {
		return 0, err
	}
	var order []GrabID
	left := map[GrabID]bool{}
	for _, row := range rows {
		if row.Stage == Imported || row.Stage == Failed {
			continue
		}
		if _, seen := left[row.ID]; !seen {
			order = append(order, row.ID)
			left[row.ID] = false
		}
		if !gone[files.CellKey{Item: row.ItemID, Edition: row.Edition, Target: row.TargetID}] {
			left[row.ID] = true
		}
	}
	n := 0
	for _, id := range order {
		if left[id] {
			continue
		}
		if err := q.SettleGrab(ctx, db.SettleGrabParams{ID: id, Unwanted: &u}); err != nil {
			return 0, err
		}
		n++
	}
	return n, nil
}
