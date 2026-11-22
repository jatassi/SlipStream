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
	_, n, err := needsYouWalk(ctx, r, 0)
	return n, err
}

// needsYouWalk is the first keep failed Grabs that need you, newest first,
// and how many there are, counted up to needsYouCap.
func needsYouWalk(ctx context.Context, r storage.Reader[file], keep int) (kept []db.ListFailedRow, n int, err error) {
	rows, err := db.New(r).ListFailed(ctx)
	if err != nil {
		return nil, 0, err
	}
	var failed []db.ListFailedRow
	for _, row := range rows {
		if failedOf(row).NeedsYou() {
			failed = append(failed, row)
		}
	}
	for batch := range slices.Chunk(failed, needsYouBatch) {
		grabs := make([]Grab, len(batch))
		for i, row := range batch {
			grabs[i] = failedOf(row)
		}
		current, again, err := grabsCellsReadFrom(ctx, r, grabs)
		if err != nil {
			return nil, 0, err
		}
		for _, row := range batch {
			if !current[row.ID] || again[row.ID] {
				continue
			}
			if len(kept) < keep {
				kept = append(kept, row)
			}
			if n++; n >= needsYouCap {
				return kept, needsYouCap, nil
			}
		}
	}
	return kept, n, nil
}

func failedOf(row db.ListFailedRow) Grab {
	return Grab{ID: row.ID, WorkID: row.WorkID, Stage: Failed, Failed: &Failure{Code: *row.FailureCode}, Source: row.Source}
}
