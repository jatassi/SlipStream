package grabs

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Enqueue lands t on every wanted cell of work in its own Write, as a
// call for one Work does, and wakes the reconciler.
func (gr *Grabs) Enqueue(ctx context.Context, work works.WorkID, t searches.Trigger) error {
	interval, err := gr.interval(ctx)
	if err != nil {
		return err
	}
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		tree, err := gr.cfg.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return err
		}
		cells, err := gr.cfg.Files.Cells(ctx, tx, work)
		if err != nil {
			return err
		}
		wanted := slices.DeleteFunc(cells, func(c files.Cell) bool { return !c.Wanted() })
		_, err = gr.enqueue(ctx, tx, work, tree.ContentType, wanted, landing(t, false), interval)
		return err
	})
	gr.wake()
	return err
}

// Nudge asks for a search of items of work, nil for the whole Work, by t
// in its own Write, as a call for one Work does.
func (gr *Grabs) Nudge(ctx context.Context, work works.WorkID, t searches.Trigger, items []works.ItemID) error {
	return gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return gr.nudge(ctx, tx, work, t, landing(t, false).lane, items)
	})
}

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
		j, err = gr.judgingOf(ctx, r, tree, scopeOf(items), expect)
		return err
	})
	if err != nil {
		return SearchID{}, err
	}
	as := assessAll(j, dedupe(listedOf(rs)))
	interval, err := gr.interval(ctx)
	if err != nil {
		return SearchID{}, err
	}
	run := Search{ID: SearchID{UUID: storage.NewUUID()}, WorkID: work, Group: indexers.ItemGroup(items[0]), Trigger: searches.Search, Items: items}
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		started := storage.Now()
		run.Started = started.Time()
		p := db.InsertSearchParams{
			ID: run.ID, WorkID: work, GroupKey: &run.Group, Lane: new(int64(pace.Now)), Trigger: new(string(searches.Search)),
			IncludeGone: true, StartedAt: started,
		}
		if err := q.InsertSearch(ctx, p); err != nil {
			return err
		}
		for _, it := range items {
			if err := q.InsertSearchItem(ctx, db.InsertSearchItemParams{SearchID: run.ID, ItemID: it}); err != nil {
				return err
			}
		}
		_, err := gr.decided(ctx, tx, run, heardRun{releases: rs, names: map[int]string{}, answered: true}, as, interval)
		return err
	})
	return run.ID, err
}
