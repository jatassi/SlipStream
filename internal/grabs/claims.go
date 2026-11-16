package grabs

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// claim is one pick an automatic decision grabs, or waits for: the cells
// it holds, the one leading first, the Items its Release covers, the
// Ticket take readied it with, the Items the fewer-Releases step chose it
// to fill, 0 when another step did, and when its wait ends, zero for a
// Grab.
type claim struct {
	cells  []files.CellKey
	covers []works.ItemID
	pick   Pick
	ticket indexers.Ticket
	fills  int
	until  time.Time
}

// claimsOf is c's walk with take: its open cells in Want order, then its
// filled cells. Each pick holds every free cell of its Edition where it
// stands auto, and the other Items' open cells where it ranks first, so
// a pack that the fewer-Releases step puts first fills the season and a
// single fills what the pack leaves. When holds, a pick the cells'
// Holds keep past now waits.
func claimsOf(c cells, as []assessed, take taker, gone map[int]bool, holds bool, now time.Time) ([]claim, error) {
	walked := len(c.open) + len(c.filled)
	in := planIn{item: slices.Concat(c.open, c.filled, c.others), as: as, take: take, gone: gone, wanted: c.wanted, chosen: c.chosen}
	in.fill, in.scope = make([]bool, len(in.item)), make([]bool, len(in.item))
	for k := range in.item {
		if k < walked {
			in.walk, in.fill[k] = append(in.walk, k), true
		}
		in.scope[k] = k < len(c.open) || k >= walked
	}
	for i := range as {
		j := judge(cells{open: c.open, filled: c.filled}, &as[i])
		for _, v := range judge(cells{open: c.others}, &as[i]).verdicts {
			v.cell += walked
			j.verdicts = append(j.verdicts, v)
		}
		in.js, in.st = append(in.js, j), append(in.st, stances(in.item, j))
	}
	also := func(r planned) []int {
		var out []int
		for _, k := range packCells(c, in.js, r.pick) {
			out = append(out, walked+slices.IndexFunc(c.others, func(cell files.Cell) bool { return keyOf(cell) == k }))
		}
		return out
	}
	h := hooks{also: also}
	if holds {
		h.hold = holdRule(in, also, now)
	}
	rows, _, err := walk(in, h)
	if err != nil {
		return nil, err
	}
	var out []claim
	for _, r := range rows {
		if r.group < 0 {
			continue
		}
		cl := claim{covers: as[r.group].covers, pick: r.pick, ticket: r.ticket, fills: r.fills, until: r.until}
		for _, k := range r.cells {
			cl.cells = append(cl.cells, keyOf(in.item[k]))
		}
		out = append(out, cl)
	}
	return out, nil
}

// heldOf is each of unfinished with an info hash, by it.
func heldOf(unfinished []Grab) map[bencode.InfoHash]Grab {
	out := map[bencode.InfoHash]Grab{}
	for _, g := range unfinished {
		if h := g.infoHash(); h != nil {
			out[*h] = g
		}
	}
	return out
}

// taker is an automatic decision's take in tx: Indexers.Take, which
// reserves a grab, but for a Release an unfinished Grab of the Work
// holds, which it joins while that Grab fetches or downloads and passes
// over after.
func (gr *Grabs) taker(ctx context.Context, tx *storage.Tx[file], held map[bencode.InfoHash]Grab) taker {
	return func(r indexers.Release) (indexers.Ticket, error) {
		if r.InfoHash == nil {
			return gr.cfg.Indexers.Take(ctx, tx, r)
		}
		g, ok := held[*r.InfoHash]
		if !ok {
			return gr.cfg.Indexers.Take(ctx, tx, r)
		}
		if !g.joinable() {
			return indexers.Ticket{}, &indexers.NotTakenError{}
		}
		return g.Ticket, nil
	}
}

// commitClaims writes cs, run's Grab claims, in tx: a Grab of each pick,
// or the pick's cells added to the unfinished Grab of held its Release
// joins, and the run's link to each, made or joined. A Grab ends the wait
// of each cell it holds. It answers the Grabs in cs' order.
func (gr *Grabs) commitClaims(ctx context.Context, tx *storage.Tx[file], run Search, cs []claim, held map[bencode.InfoHash]Grab, now storage.Time) ([]Grab, error) {
	q := db.New(tx)
	var out []Grab
	for _, c := range cs {
		for _, k := range c.cells {
			if err := q.DeleteWait(ctx, db.DeleteWaitParams{ItemID: k.Item, Edition: k.Edition, TargetID: k.Target}); err != nil {
				return nil, err
			}
		}
		var g Grab
		var err error
		joined := false
		if h := c.pick.Release.InfoHash; h != nil {
			g, joined = held[*h]
		}
		if joined {
			g, err = gr.join(ctx, tx, g, c.cells)
		} else {
			cells := make([]Cell, len(c.cells))
			for i, k := range c.cells {
				cells[i] = Cell{Key: k}
			}
			n := newGrab{work: run.WorkID, item: c.cells[0].Item, items: c.covers, cells: cells, pick: c.pick, ticket: c.ticket, source: sourceOf(run), foundBy: foundByOf(run), fills: c.fills}
			g, err = gr.insertGrab(ctx, tx, n, now)
		}
		if err != nil {
			return nil, err
		}
		if err := q.InsertSearchGrab(ctx, db.InsertSearchGrabParams{SearchID: run.ID, GrabID: g.ID, Made: !joined}); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

// sourceOf is the source of the Grabs run's claims make.
func sourceOf(run Search) Source {
	if run.Trigger == searches.RSS {
		return FromRSS
	}
	return FromSearch
}

// foundByOf is what found the Releases run grabs.
func foundByOf(run Search) FoundBy {
	switch run.Trigger {
	case searches.RSS:
		return FoundByRSS
	case searches.Backlog:
		return FoundByBacklog
	case searches.FailedGrab:
		return FoundByFailedGrab
	case searches.Search, searches.SearchAll, searches.Add, searches.Monitor, searches.Profile, searches.Editions, searches.Retarget,
		searches.ReleaseDate, searches.Refresh:
	}
	return FoundBySearch
}

// commitWaits writes ws, run's waiting claims, once run's Releases are
// stored: each cell's wait, which a cell already waiting keeps until the
// earlier time, with the claim's Release as one more of its candidates.
func commitWaits(ctx context.Context, q *db.Queries, run Search, ws []claim, now storage.Time) error {
	for _, c := range ws {
		for _, k := range c.cells {
			w := db.UpsertWaitParams{WorkID: run.WorkID, ItemID: k.Item, Edition: k.Edition, TargetID: k.Target, UntilAt: storage.FromTime(c.until), StartedAt: now}
			if err := q.UpsertWait(ctx, w); err != nil {
				return err
			}
			cand := db.InsertWaitCandidateParams{ItemID: k.Item, Edition: k.Edition, TargetID: k.Target, SearchID: run.ID, Position: int64(c.pick.at)}
			if err := q.InsertWaitCandidate(ctx, cand); err != nil {
				return err
			}
		}
	}
	return nil
}

// join adds cells to g, the Grab a decision's pick joins, but those g
// already holds.
func (gr *Grabs) join(ctx context.Context, tx *storage.Tx[file], g Grab, cells []files.CellKey) (Grab, error) {
	q := db.New(tx)
	for _, k := range cells {
		if slices.ContainsFunc(g.Targets, func(c Cell) bool { return c.Key.Item == k.Item && c.Key.Target == k.Target }) {
			continue
		}
		if !slices.Contains(g.Items, k.Item) {
			if err := q.InsertGrabItem(ctx, db.InsertGrabItemParams{GrabID: g.ID, ItemID: k.Item, Position: int64(len(g.Items))}); err != nil {
				return Grab{}, err
			}
			g.Items = append(g.Items, k.Item)
		}
		if err := q.InsertTarget(ctx, db.InsertTargetParams{GrabID: g.ID, Position: int64(len(g.Targets)), ItemID: k.Item, TargetID: k.Target}); err != nil {
			return Grab{}, err
		}
		g.Targets = append(g.Targets, Cell{Key: files.CellKey{Item: k.Item, Edition: g.Edition, Target: k.Target}})
	}
	return g, nil
}
