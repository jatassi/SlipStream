package works

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// Added is a Work by its id and label.
type Added struct {
	ID    WorkID
	Label string
}

// AddedSince is every Work a Library scan added after since, oldest
// first, on the partial index works_work_added_by_scan. It has no limit:
// the caller keeps the ones it wants, then counts.
func AddedSince(ctx context.Context, r storage.Reader[file], since storage.Time) ([]Added, error) {
	rows, err := db.New(r).ListAddedByScanSince(ctx, since)
	out := make([]Added, len(rows))
	for i, row := range rows {
		out[i] = Added(row)
	}
	return out, err
}

// IDsOf is the ids of added, in order.
func IDsOf(added []Added) []WorkID {
	out := make([]WorkID, len(added))
	for i, a := range added {
		out[i] = a.ID
	}
	return out
}

// LabelsOf is the labels of the first n of added that keep holds, in
// added's order.
func LabelsOf(added []Added, keep []WorkID, n int) []string {
	kept := make(map[WorkID]bool, len(keep))
	for _, id := range keep {
		kept[id] = true
	}
	var out []string
	for _, a := range added {
		if len(out) == n {
			break
		}
		if kept[a.ID] {
			out = append(out, a.Label)
		}
	}
	return out
}
