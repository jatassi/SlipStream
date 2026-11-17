package grabs

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Several is a search of several Works at once: Works by id, as the
// Library's bulk Search sends them, or every Work Wanted lists for Wanted,
// as Search all does.
type Several struct {
	Works  []works.WorkID
	Wanted *WantedFilter
}

// WantedFilter is the Works Wanted lists: of Content Type Type, every one
// when zero, wanted for any of Reasons, every Reason but gone when nil.
type WantedFilter struct {
	Type    works.TypeID
	Reasons []files.Reason
}

// goneAlone is whether f lists only gone cells, the one filter whose
// search opens them.
func (f WantedFilter) goneAlone() bool {
	return slices.Equal(f.Reasons, []files.Reason{files.ReasonGone})
}

// Skip is why a search of several Works left one out.
type Skip string

const (
	SkipNothingWanted Skip = "nothing_wanted"
	// SkipGoneOnly is a Work whose only wanted cells are gone.
	SkipGoneOnly      Skip = "gone_only"
	SkipNoRoot        Skip = "no_root"
	SkipNotSearchable Skip = "not_searchable"
)

// WorkOutcome is what a search of several Works did with one: queued,
// merged when every group of it was already queued or searching, or
// skipped with why.
type WorkOutcome struct {
	Work    works.WorkID
	Merged  bool
	Skipped Skip
}

// SeveralQueued is a search of several Works' answer: how many it queued
// and merged, how many it left out because only gone cells want, and each
// Work's outcome.
type SeveralQueued struct {
	Queued, Merged, LeftGone int
	Works                    []WorkOutcome
}

func (s *SeveralQueued) add(o WorkOutcome) {
	switch {
	case o.Skipped == SkipGoneOnly:
		s.LeftGone++
	case o.Skipped != "":
	case o.Merged:
		s.Merged++
	default:
		s.Queued++
	}
	s.Works = append(s.Works, o)
}

// SearchSeveral queues each Work of s in the bulk lane, in one Write: the
// groups of its Searchable cells, and of its gone cells when s's filter
// is gone alone. It doesn't answer the checks a press on one Work does: a
// Work that can't search is skipped with why, and dispatch refuses a run
// no Indexer serves.
func (gr *Grabs) SearchSeveral(ctx context.Context, _ identity.Admin, s Several) (SeveralQueued, error) {
	interval, err := gr.interval(ctx)
	if err != nil {
		return SeveralQueued{}, err
	}
	var out SeveralQueued
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := gr.fence(ctx, tx); err != nil {
			return err
		}
		listed, gone, err := gr.severalOf(ctx, tx, s)
		if err != nil {
			return err
		}
		e := landing(searches.SearchAll, true)
		e.gone = gone
		covers := map[works.TypeID]bool{}
		out = SeveralQueued{Works: make([]WorkOutcome, 0, len(listed))}
		for _, w := range listed {
			o, err := gr.searchOne(ctx, tx, w, e, covers, interval)
			if err != nil {
				return err
			}
			out.add(o)
		}
		return nil
	})
	if err != nil {
		return SeveralQueued{}, err
	}
	changed := map[works.WorkID]bool{}
	for _, o := range out.Works {
		if o.Skipped == "" && !o.Merged {
			changed[o.Work] = true
		}
	}
	gr.queueChanged(ctx, changed)
	gr.wake()
	return out, nil
}

// severalOf is what each Work s names wants, in s's order, and whether
// its search opens gone cells. A filter lists each Work wanted for one of
// its Reasons, and each Work only gone cells want, which it leaves out.
func (gr *Grabs) severalOf(ctx context.Context, r storage.Reader[file], s Several) ([]files.WorkWanted, bool, error) {
	if s.Wanted == nil {
		ws, err := gr.cfg.Files.Wanted(ctx, r, works.Only(s.Works))
		if err != nil {
			return nil, false, err
		}
		of := make(map[works.WorkID]files.WorkWanted, len(ws))
		for _, w := range ws {
			of[w.Work] = w
		}
		out := make([]files.WorkWanted, len(s.Works))
		for i, id := range s.Works {
			out[i] = files.WorkWanted{Work: id}
			if w, ok := of[id]; ok {
				out[i] = w
			}
		}
		return out, false, nil
	}
	f := *s.Wanted
	if f.Reasons == nil {
		f.Reasons = []files.Reason{files.ReasonFailed, files.ReasonMissing, files.ReasonUpgrade}
	}
	ws, err := gr.cfg.Files.Wanted(ctx, r, works.Every[works.WorkID](nil))
	if err != nil {
		return nil, false, err
	}
	return slices.DeleteFunc(ws, func(w files.WorkWanted) bool {
		if f.Type != "" && w.Type != f.Type {
			return true
		}
		reasons := reasonsOf(w.Cells)
		return !slices.ContainsFunc(f.Reasons, func(r files.Reason) bool { return slices.Contains(reasons, r) }) &&
			!slices.Equal(reasons, []files.Reason{files.ReasonGone})
	}), f.goneAlone(), nil
}

// reasonsOf is every Reason Wanted lists cells for.
func reasonsOf(cells []files.Cell) []files.Reason {
	lead, others, ok := files.ReasonOf(cells)
	if !ok {
		return nil
	}
	out := []files.Reason{lead}
	for r := range others {
		out = append(out, r)
	}
	slices.Sort(out)
	return out
}

// searchOne lands e on the groups of w's cells e opens, and answers what
// it did. covers is whether each Content Type's search terms cover
// Items, filled as it reads them.
func (gr *Grabs) searchOne(ctx context.Context, tx *storage.Tx[file], w files.WorkWanted, e landed, covers map[works.TypeID]bool, interval time.Duration) (WorkOutcome, error) {
	o := WorkOutcome{Work: w.Work}
	var cells []files.Cell
	for _, c := range w.Cells {
		if opens(c, e.trigger, e.gone) {
			cells = append(cells, c)
		}
	}
	if len(cells) == 0 {
		o.Skipped = SkipNothingWanted
		if slices.ContainsFunc(w.Cells, func(c files.Cell) bool { return c.Gone }) {
			o.Skipped = SkipGoneOnly
		}
		return o, nil
	}
	ok, seen := covers[w.Type]
	if !seen {
		terms, err := gr.cfg.Terms[w.Type].SearchTerms(ctx, tx, cells[0].Item.ID)
		if err != nil {
			return WorkOutcome{}, err
		}
		ok = terms.Covers != nil
		covers[w.Type] = ok
	}
	if !ok {
		o.Skipped = SkipNotSearchable
		return o, nil
	}
	switch _, err := roots.OfWork(ctx, tx, w.Work); {
	case errors.Is(err, roots.ErrNoRoot):
		o.Skipped = SkipNoRoot
		return o, nil
	case err != nil:
		return WorkOutcome{}, err
	}
	queued, err := gr.enqueue(ctx, tx, w.Work, w.Type, cells, e, interval)
	if err != nil {
		return WorkOutcome{}, err
	}
	o.Merged = !slices.ContainsFunc(queued, func(q Queued) bool { return !q.Merged })
	return o, nil
}
