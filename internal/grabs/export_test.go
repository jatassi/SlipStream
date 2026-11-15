package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// DecideRun records a press's run of items of work, as if its lines had
// listed rs, and decides it in one Write, as a group run whose every line
// answered does.
func (gr *Grabs) DecideRun(ctx context.Context, work works.WorkID, items []works.ItemID, rs []indexers.Release) (SearchID, error) {
	var j judging
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		tree, err := gr.cfg.Works.TreeIn(ctx, r, work)
		if err != nil {
			return err
		}
		expect, err := gr.cfg.Terms[tree.ContentType].Expect(ctx, r, work)
		if err != nil {
			return err
		}
		j, err = gr.judgingOf(ctx, r, tree.ContentType, work, works.ItemID{}, expect)
		return err
	})
	if err != nil {
		return SearchID{}, err
	}
	as := assessAll(j, dedupe(listedOf(rs)))
	run := Search{ID: SearchID{UUID: storage.NewUUID()}, WorkID: work, Group: indexers.ItemGroup(items[0]), Trigger: pressTrigger, Items: items}
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		started := storage.Now()
		run.Started = started.Time()
		lane := pressLane
		p := db.InsertSearchParams{ID: run.ID, WorkID: work, GroupKey: &run.Group, Lane: &lane, Trigger: new(string(pressTrigger)), IncludeGone: true, StartedAt: started}
		if err := q.InsertSearch(ctx, p); err != nil {
			return err
		}
		for _, it := range items {
			if err := q.InsertSearchItem(ctx, db.InsertSearchItemParams{SearchID: run.ID, ItemID: it}); err != nil {
				return err
			}
		}
		_, err := gr.finish(ctx, tx, run, nil, nil, as)
		return err
	})
	return run.ID, err
}
