package grabs

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// claim is one pick an automatic decision grabs: the cells it holds,
// the one leading first, the Items its Release covers, the Ticket take
// readied it with, and the Items the fewer-Releases step chose it to
// fill, 0 when another step did.
type claim struct {
	cells  []files.CellKey
	covers []works.ItemID
	pick   Pick
	ticket indexers.Ticket
	fills  int
}

// claimsOf is c's walk with take: its open cells in Want order, then its
// filled cells. Each pick holds every free cell of its Edition where it
// stands auto, and the other Items' open cells where it ranks first, so
// a pack that the fewer-Releases step puts first fills the season and a
// single fills what the pack leaves.
func claimsOf(c cells, as []assessed, take taker, gone map[int]bool) ([]claim, error) {
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
	rows, _, err := walk(in, hooks{also: also})
	if err != nil {
		return nil, err
	}
	var out []claim
	for _, r := range rows {
		if r.group < 0 {
			continue
		}
		cl := claim{covers: as[r.group].covers, pick: r.pick, ticket: r.ticket, fills: r.fills}
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

// commitClaims writes cs, run's claims, in tx: a Grab of each pick, or
// the pick's cells added to the unfinished Grab of held its Release
// joins, and the run's link to each, made or joined. It answers the
// Grabs in cs' order.
func (gr *Grabs) commitClaims(ctx context.Context, tx *storage.Tx[file], run Search, cs []claim, held map[bencode.InfoHash]Grab, now storage.Time) ([]Grab, error) {
	q := db.New(tx)
	var out []Grab
	for _, c := range cs {
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
			n := newGrab{work: run.WorkID, item: c.cells[0].Item, items: c.covers, cells: cells, pick: c.pick, ticket: c.ticket, source: FromSearch, fills: c.fills}
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
