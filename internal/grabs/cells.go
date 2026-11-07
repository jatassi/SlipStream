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

// GrabCells is files.Inflight over the Grabs: each cell takes its latest
// Grab, downloading once past fetching and failed once failed. An
// unfinished Grab whose Node is out of step waits on it, node_away, and
// one whose Torrent's client is away while its Node is in step waits on
// the client, client_away.
type GrabCells struct {
	InStep func(role.NodeID) bool
}

const (
	// NodeAway is the Waiting code of a Grab whose Node is out of step.
	NodeAway = "node_away"
	// ClientAway is the Waiting code of a Grab whose Torrent's client
	// isn't responding, refused the sign-in or needs attention.
	ClientAway = "client_away"
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
	away, err := g.awayClients(ctx, r)
	if err != nil {
		return nil, err
	}
	out := map[files.CellKey]files.GrabCell{}
	for _, row := range rows {
		key := files.CellKey{Item: row.ItemID, Edition: row.Edition, Target: row.TargetID}
		cell := files.GrabCell{Grab: row.ID.UUID}
		switch row.Stage {
		case Imported:
			delete(out, key)
			continue
		case Failed:
			cell.State = files.GrabFailed
		case Downloading, Probing, Transferring, Importing:
			cell.State = files.GrabDownloading
		case Fetching:
		}
		switch {
		case row.Stage == Failed:
		case !g.InStep(row.NodeID):
			cell.Waiting = NodeAway
		case away[row.ID]:
			cell.Waiting = ClientAway
		}
		out[key] = cell
	}
	return out, nil
}

// awayClients is each unfinished Grab whose Torrent an away client holds.
func (g GrabCells) awayClients(ctx context.Context, r storage.Reader[storage.Library]) (map[GrabID]bool, error) {
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
	out := map[GrabID]bool{}
	for _, gr := range open {
		if away[gr.TorrentID] {
			out[gr.ID] = true
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
