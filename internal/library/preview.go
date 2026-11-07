package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Previewed is the Work an add would make, each Item's availability, and
// what each preset would monitor of it as an add would make it, with no
// file. Stored is whether the Work is in the library already.
type Previewed struct {
	Tree    works.Tree
	Timings map[works.ItemID]targets.Timing
	Stored  bool
	Counts  works.PresetCounts
}

// Preview is the Work an add of the record t's source knows by key would
// make, with what each preset would monitor of it, from works.Preview's
// rolled-back Write.
func (l *Library) Preview(ctx context.Context, by identity.Admin, t works.TypeID, key works.Key) (Previewed, error) {
	var out Previewed
	tree, stored, err := l.cfg.Works.Preview(ctx, by, t, key, func(ctx context.Context, r storage.Reader[storage.Library], tree works.Tree) error {
		among := works.Only([]works.WorkID{tree.ID})
		var err error
		out.Timings, err = l.cfg.Targets.TimingsIn(ctx, r, tree.ContentType, among)
		out.Counts = works.CountPresets(tree, l.cfg.Files.UnfilledFacts(tree, out.Timings))
		return err
	})
	if err != nil {
		return Previewed{}, err
	}
	out.Tree, out.Stored = tree, stored
	return out, nil
}
