package grabs

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// decisionIn is what one decision reads: the run its Releases belong to,
// those Releases, deduped and assessed, before the Blocklist gate, which
// cells it opens and whether their Targets' Holds apply. Only its
// constructors build it.
type decisionIn struct {
	run   Search
	as    []assessed
	open  openRule
	holds bool
	// only is the cells of the run's Items a rerun opens, the dropped
	// Grab's; nil opens every cell of them the rule opens.
	only map[files.CellKey]bool
	// gone is the positions it takes none of, and grabbed those another
	// Grab of the run still holds: a rerun's.
	gone    map[int]bool
	grabbed []int
}

// openRule is which wanted cells a decision opens: gone ones when gone,
// stopped ones when stopped.
type openRule struct{ gone, stopped bool }

func (o openRule) opens(c files.Cell) bool {
	return c.Wanted() && (o.gone || !c.Gone) && (o.stopped || !c.Search.Stopped)
}

// runInput is the decision of run, a search run, over what its lines
// listed. Its trigger's lane rule opens gone cells for the Admin's
// Search and stopped ones for Backlog search, and applies Holds to every
// trigger but the Admin's: a press, promoting the run, takes them off.
func runInput(run Search, as []assessed) decisionIn {
	rule := laneRules[run.Trigger]
	return decisionIn{run: run, as: as, open: openRule{gone: rule.gone, stopped: rule.paced}, holds: rule.holds}
}

// rerunInput is the decision of re's run again, over the Releases it
// kept, at the cells its dropped Grab held. It applies no Hold: the run
// already grabbed past them, and it keeps no Release for a wait.
func rerunInput(re redo) decisionIn {
	in := runInput(re.run, re.assessed)
	in.holds = false
	in.only = map[files.CellKey]bool{}
	for _, k := range re.cells {
		in.only[k] = true
	}
	in.gone, in.grabbed = re.gone, re.grabbed
	return in
}

// board is the cells in walks, out of all the Work's: open, every cell
// of the run's Items it opens, then filled, their Met cells, which a
// proper or repack fills, and others, the Work's other cells it opens,
// which a pick holds where it ranks first. Unfinished Grabs fetching or
// downloading a Release are what other Wants chose.
func (in decisionIn) board(all []files.Cell, unfinished []Grab) cells {
	scope := map[works.ItemID]bool{}
	for _, it := range in.run.Items {
		scope[it] = true
	}
	c := cells{wanted: map[works.ItemID]bool{}, chosen: takenBy{hashes: map[bencode.InfoHash]bool{}}}
	for _, cell := range all {
		opens, scoped := in.open.opens(cell), scope[cell.Item.ID] && (in.only == nil || in.only[keyOf(cell)])
		switch {
		case opens && scoped:
			c.open = append(c.open, cell)
		case opens:
			c.others = append(c.others, cell)
		case cell.State == files.Met && scoped && in.only == nil:
			c.filled = append(c.filled, cell)
		}
		if opens {
			c.wanted[cell.Item.ID] = true
		}
	}
	for _, g := range unfinished {
		if h := g.infoHash(); h != nil && g.joinable() {
			c.chosen.hashes[*h] = true
		}
	}
	return c
}

// decisionOut is what a decision took and why: the Grabs it made or
// joined, in the order it walked them, the notes of the first one's
// pick, the claims that wait, which commitWaits writes once the Releases
// are stored, every Release with its outcome, how many passed the
// Blocklist, and when it decided.
type decisionOut struct {
	grabs      []Grab
	waits      []claim
	notes      []Reason
	rows       []row
	candidates int
	at         storage.Time
}

// decideIn is the one decision, in the deciding Write: the Blocklist read
// and gate, sightings, Format timeouts, then the walk and its claims.
func (gr *Grabs) decideIn(ctx context.Context, tx *storage.Tx[file], in decisionIn) (decisionOut, error) {
	bl, err := blocksOf(ctx, tx, in.as)
	if err != nil {
		return decisionOut{}, err
	}
	in.as = bl.gate(in.as)
	if err := sight(ctx, tx, in.as); err != nil {
		return decisionOut{}, err
	}
	results := make([]formats.Result, len(in.as))
	for i, a := range in.as {
		results[i] = a.formats
	}
	if err := targets.AddTimeouts(ctx, tx, results); err != nil {
		return decisionOut{}, err
	}
	return gr.claimIn(ctx, tx, in)
}

// walkOut is a decision's walk: its board, the Work's unfinished Grabs by
// info hash, which a claim joins, and the claims.
type walkOut struct {
	board  cells
	held   map[bencode.InfoHash]Grab
	claims []claim
}

// walkIn walks in's board in tx over its gated Releases at now.
func (gr *Grabs) walkIn(ctx context.Context, tx *storage.Tx[file], in decisionIn, now time.Time) (walkOut, error) {
	all, err := gr.cfg.Files.Cells(ctx, tx, in.run.WorkID)
	if err != nil {
		return walkOut{}, err
	}
	unfinished, err := unfinishedOf(ctx, tx, in.run.WorkID)
	if err != nil {
		return walkOut{}, err
	}
	w := walkOut{board: in.board(all, unfinished), held: heldOf(unfinished)}
	w.claims, err = claimsOf(w.board, in.as, gr.taker(ctx, tx, w.held), in.gone, in.holds, now)
	return w, err
}

// claimIn walks in's board over its gated Releases, commits each claim
// and answers every Release's outcome.
func (gr *Grabs) claimIn(ctx context.Context, tx *storage.Tx[file], in decisionIn) (decisionOut, error) {
	out := decisionOut{candidates: len(in.as), at: storage.Now()}
	w, err := gr.walkIn(ctx, tx, in, out.at.Time())
	if err != nil {
		return decisionOut{}, err
	}
	c, held, cs := w.board, w.held, w.claims
	var grabbing []claim
	taken, waited := in.grabbed, []int(nil)
	for _, cl := range cs {
		if cl.until.IsZero() {
			grabbing, taken = append(grabbing, cl), append(taken, cl.pick.at)
		} else {
			out.waits, waited = append(out.waits, cl), append(waited, cl.pick.at)
		}
	}
	if out.grabs, err = gr.commitClaims(ctx, tx, in.run, grabbing, held, out.at); err != nil {
		return decisionOut{}, err
	}
	if len(grabbing) > 0 {
		out.notes = grabbing[0].pick.Notes
	}
	out.rows = outcomes(decide(c, in.as), c, taken, waited)
	return out, nil
}
