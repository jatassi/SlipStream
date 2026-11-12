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
// any Grab in flight holds it unsettled, fetching included, and reads from
// the latest such Grab, whatever finished after it. Else it reads from its
// latest Grab, failed once failed, and from no Grab once that Grab settled
// the cell. A failure that doesn't need the Admin,
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
	for key, row := range readFrom(rows) {
		cell := files.GrabCell{Grab: row.ID.UUID}
		switch {
		case row.Settled != nil, row.Stage == Imported, row.Stage == Failed && !(Grab{Stage: Failed, Failed: &Failure{Code: *row.FailureCode}, Source: row.Source}).NeedsYou():
			continue
		case row.Stage == Failed:
			cell.State, cell.FailedAt = files.GrabFailed, string(*row.FailedStage)
		case !g.InStep(row.NodeID):
			cell.State, cell.Waiting = files.GrabInFlight, NodeAway
		default:
			cell.State, cell.Waiting = files.GrabInFlight, waits[row.ID]
		}
		out[key] = cell
	}
	return out, nil
}

func readFrom(oldestFirst []db.ListTargetsOfWorksRow) map[files.CellKey]db.ListTargetsOfWorksRow {
	out := map[files.CellKey]db.ListTargetsOfWorksRow{}
	for _, row := range oldestFirst {
		key := files.CellKey{Item: row.ItemID, Edition: row.Edition, Target: row.TargetID}
		if held, ok := out[key]; ok && holds(held) && !holds(row) {
			continue
		}
		out[key] = row
	}
	return out
}

// holds reports whether row's Grab holds its cell: the Grab is in flight
// and hasn't settled it.
func holds(row db.ListTargetsOfWorksRow) bool { return row.Stage.InFlight() && row.Settled == nil }

// inFlightStages is every Stage whose Grabs are in flight.
var inFlightStages = func() []Stage {
	var out []Stage
	for _, s := range Stages {
		if s.InFlight() {
			out = append(out, s)
		}
	}
	return out
}()

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
