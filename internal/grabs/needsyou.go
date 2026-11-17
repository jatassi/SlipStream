package grabs

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// needsYouCap bounds the count History's sentence words: it reads every
// failed Grab only until this many need the Admin.
const needsYouCap = 100

// needsYouBatch is how many failed Grabs one pass reads the cells of.
const needsYouBatch = 500

// NeedsYou counts the failed Grabs Activity lists under Needs you: each
// failed in a way the Admin didn't choose, isn't searched again, and a cell
// still reads from. It reads failed Grabs only, newest first, and stops at
// 100.
func NeedsYou(ctx context.Context, r storage.Reader[file]) (int, error) {
	rows, err := db.New(r).ListFailed(ctx)
	if err != nil {
		return 0, err
	}
	var failed []Grab
	for _, row := range rows {
		g := Grab{ID: row.ID, WorkID: row.WorkID, Stage: Failed, Failed: &Failure{Code: *row.FailureCode}, Source: row.Source}
		if g.NeedsYou() {
			failed = append(failed, g)
		}
	}
	n := 0
	for batch := range slices.Chunk(failed, needsYouBatch) {
		current, again, err := grabsCellsReadFrom(ctx, r, batch)
		if err != nil {
			return 0, err
		}
		for _, g := range batch {
			if current[g.ID] && !again[g.ID] {
				n++
			}
		}
		if n >= needsYouCap {
			return needsYouCap, nil
		}
	}
	return n, nil
}
