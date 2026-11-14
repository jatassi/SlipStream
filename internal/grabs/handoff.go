package grabs

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/works"
)

// Importer is the import pipeline as grabs reaches it: a Grab's download
// is its Import's from the Write that moves it to importing. Each method
// but Wake runs inside a grabs Write, writes only the Import's rows, does
// no IO, and reports back through Follow in the same Write.
type Importer interface {
	// Start creates the Import of h's Grab. A second Start is nothing to
	// do.
	Start(ctx context.Context, tx *storage.Tx[storage.Library], h Handoff) error
	// End ends g's Import for why: each file not yet placed or placing is
	// declined, and a placing file finishes. With cancel, it refuses with
	// *GrabBusyError while a file is placing.
	End(ctx context.Context, tx *storage.Tx[storage.Library], g GrabID, why Failure, cancel bool) error
	// Wake has the pipeline look at its Imports soon. It doesn't block.
	Wake()
}

// Register sets the Importer, before Run.
func (gr *Grabs) Register(im Importer) {
	if gr.importer != nil {
		panic("grabs: an Importer is registered already")
	}
	gr.importer = im
}

// Handoff is a Grab as its Import reads it.
type Handoff struct {
	Grab GrabID
	Work works.WorkID
	// Edition is the Edition the Admin grabbed it as, nil for its
	// Release's own cut.
	Edition *release.CutKey
	// Items is every Item its Release covers in the Work, in tree order.
	Items []works.ItemID
	// Cells is its cells in plan order, each with how it settled.
	Cells    []Cell
	Release  string
	Stage    Stage
	Node     role.NodeID
	InfoHash bencode.InfoHash
	Torrent  downloadclients.ID
	// Files is the download's files from its Completion.
	Files []engine.File
}

// Handed reads Grab id's Handoff in r, with each cell as it settled now.
func Handed(ctx context.Context, r storage.Reader[storage.Library], id GrabID) (Handoff, error) {
	g, err := grabIn(ctx, r, id)
	if err != nil {
		return Handoff{}, err
	}
	h := Handoff{
		Grab: g.ID, Work: g.WorkID, Items: g.Items, Cells: g.Targets, Release: g.Release.Title, Stage: g.Stage, Node: g.Node,
		InfoHash: g.Torrent.InfoHash, Torrent: g.Torrent.ID, Files: g.Torrent.Files,
	}
	if g.Overrides.Edition != "" {
		h.Edition = new(release.CutKey(g.Edition))
	}
	return h, nil
}

// Report is what a Grab's Import tells it: facts, not commands.
type Report struct {
	// Settled is how each cell the Import is done with settled.
	Settled map[files.CellKey]Settled
	// Placed is whether any file of the Import was placed.
	Placed bool
	// Ended is how the Import ended, once it places nothing more.
	Ended *Failure
}

// Followed is what a Follow changed, to publish once its Write commits.
type Followed struct {
	grab    Grab
	settled bool
	next    Stage
}

// Follow brings importing Grab id in line with r in tx. Each cell r
// settles is settled for good. Then, once something was placed and every
// cell settled, the Grab is imported; once nothing was placed and the
// Import ended, it fails as the Import ended. Else it stays importing.
// A Grab gone or past importing is nothing to do.
func (gr *Grabs) Follow(ctx context.Context, tx *storage.Tx[storage.Library], id GrabID, r Report) (Followed, error) {
	g, err := grabIn(ctx, tx, id)
	if errors.Is(err, ErrNoGrab) {
		return Followed{}, nil
	}
	if err != nil || g.Stage != Importing {
		return Followed{}, err
	}
	q, now := db.New(tx), storage.Now()
	out, open := Followed{grab: g}, 0
	for _, c := range g.Targets {
		if c.Settled != nil {
			continue
		}
		s, ok := r.Settled[c.Key]
		if !ok {
			open++
			continue
		}
		p := db.SettleTargetParams{GrabID: g.ID, ItemID: c.Key.Item, TargetID: c.Key.Target, Settled: &s, SettledAt: now}
		if _, err := q.SettleTarget(ctx, p); err != nil {
			return Followed{}, err
		}
		out.settled = true
	}
	var o outcome
	switch {
	case r.Placed && open == 0:
		o = advance{to: Imported}
	case !r.Placed && r.Ended != nil:
		o = fail{r.Ended.Code, r.Ended.Message}
	default:
		return out, nil
	}
	params, err := gr.move(ctx, tx, g, o)
	out.next = params.Next
	return out, err
}

// Followed publishes what each Follow changed, once its Write committed.
// A Follow that neither moved nor settled its Grab still changed the
// Import its view reads.
func (gr *Grabs) Followed(ctx context.Context, fs ...Followed) {
	woken := false
	for _, f := range fs {
		switch {
		case f.next != "":
			gr.entered(ctx, f.grab, f.next)
		case f.settled:
			gr.cfg.Changed.Publish(ctx, Changed{GrabID: f.grab.ID.String(), WorkID: f.grab.WorkID.String()})
		case !f.grab.ID.IsZero():
			gr.cfg.Changed.Publish(ctx, Changed{GrabID: f.grab.ID.String(), WorkID: f.grab.WorkID.String()})
			continue
		default:
			continue
		}
		woken = true
	}
	if woken {
		gr.wake()
	}
}

// GoneFailure is how an importing Grab fails when its Torrent was removed
// for g's reason before anything was placed.
func GoneFailure(g downloadclients.Gone) Failure {
	f := failOfGone(g)
	return Failure{Stage: Importing, Code: f.code, Message: f.msg}
}

// Carry is a Grab this release found past its download, as grabs 00012
// kept it: what its old stage chose, for imports to carry into an Import.
type Carry struct {
	Grab GrabID
	// Was is probing, transferring or importing. A probing Grab chose
	// nothing yet, and the fields below are zero.
	Was       string
	FileIndex int
	Probe     media.Record
	Root      roots.ID
	// Path is the file's path in Root, '/' separated.
	Path  string
	Fills []targets.TargetID
}

// Carried reads every Grab still to carry, in r.
func Carried(ctx context.Context, r storage.Reader[storage.Library]) ([]Carry, error) {
	rows, err := db.New(r).ListCarried(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Carry, len(rows))
	for i, row := range rows {
		c := Carry{Grab: row.GrabID, Was: row.Was, Root: row.RootID}
		var hexes []string
		if err := json.Unmarshal([]byte(row.Fills), &hexes); err != nil {
			return nil, fmt.Errorf("carry %s: fills: %w", row.GrabID, err)
		}
		for _, h := range hexes {
			raw, err := hex.DecodeString(h)
			if err != nil {
				return nil, fmt.Errorf("carry %s: fill %q: %w", row.GrabID, h, err)
			}
			u, err := storage.UUIDFromBytes(raw)
			if err != nil {
				return nil, fmt.Errorf("carry %s: fill %q: %w", row.GrabID, h, err)
			}
			c.Fills = append(c.Fills, targets.TargetID{UUID: u})
		}
		if row.Was != "probing" {
			c.FileIndex, c.Path = int(*row.FileIndex), *row.Path
			if err := json.Unmarshal([]byte(*row.Probe), &c.Probe); err != nil {
				return nil, fmt.Errorf("carry %s: probe: %w", row.GrabID, err)
			}
		}
		out[i] = c
	}
	return out, nil
}

// Uncarry records in tx that g was carried.
func Uncarry(ctx context.Context, tx *storage.Tx[storage.Library], g GrabID) error {
	return db.New(tx).DeleteCarried(ctx, g)
}
