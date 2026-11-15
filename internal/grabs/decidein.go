package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

// decisionIn is what one decision reads: the run its Releases belong to,
// and those Releases, deduped and assessed, before the Blocklist gate.
// Only its constructors build it.
type decisionIn struct {
	run Search
	as  []assessed
}

// runInput is the decision of run, a search run, over what its lines
// listed.
func runInput(run Search, as []assessed) decisionIn { return decisionIn{run: run, as: as} }

// decisionOut is what a decision took and why: the Grab, nil when it took
// none, the notes of its pick, every Release with its outcome, how many
// passed the Blocklist, and when it decided.
type decisionOut struct {
	grab       *Grab
	notes      []Reason
	rows       []row
	candidates int
	at         storage.Time
}

// decideIn is the one decision, in the deciding Write: the Blocklist read
// and gate, sightings, Format timeouts, the judging and the take.
func (gr *Grabs) decideIn(ctx context.Context, tx *storage.Tx[file], in decisionIn) (decisionOut, error) {
	bl, err := blocksOf(ctx, tx, in.as)
	if err != nil {
		return decisionOut{}, err
	}
	as := bl.gate(in.as)
	if err := sight(ctx, tx, as); err != nil {
		return decisionOut{}, err
	}
	results := make([]formats.Result, len(as))
	for i, a := range as {
		results[i] = a.formats
	}
	if err := targets.AddTimeouts(ctx, tx, results); err != nil {
		return decisionOut{}, err
	}
	d, c, err := gr.decision(ctx, tx, in.run, as)
	if err != nil {
		return decisionOut{}, err
	}
	at := storage.Now()
	grab, pick, err := gr.take(ctx, tx, in.run, d, c, nil, at)
	if err != nil {
		return decisionOut{}, err
	}
	return decisionOut{grab: grab, notes: pick.Notes, rows: outcomes(d, c, pick.at), candidates: len(as), at: at}, nil
}
