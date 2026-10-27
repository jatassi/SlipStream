package indexers

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Release is one normalized Release from one Indexer, with the Indexer's
// priority when it was asked.
type Release struct {
	Indexer  ID
	Priority int
	searching.Release
}

// Offered is the link a Grab of r fetches: its .torrent link, else its
// magnet link, since a .torrent's info dictionary needs no peer.
func (r Release) Offered() searching.Link {
	if r.Link.IsZero() {
		return r.Magnet
	}
	return r.Link
}

// Line is one Indexer in a search: asked with its reserved query, or
// skipped and why.
type Line struct {
	Indexer ID
	// Name is the Indexer's name when the search was planned.
	Name string
	// Query is zero when the Indexer is skipped.
	Query QueryID
	// Skipped is nil when the Indexer is asked.
	Skipped *Outcome
}

// Plan is the lines of one search of an Item of Content Type typ, read
// and reserved in tx: every Indexer that is on, has Automatic search on
// and isn't an RSS feed, by priority then name. One that doesn't serve
// typ, or that the gate or its search limit holds back, is skipped with
// why. Each asked line reserves its query, so concurrent searches never
// pass a limit.
func (x *Indexers) Plan(ctx context.Context, tx *storage.Tx[file], typ works.TypeID) ([]Line, error) {
	all, err := All(ctx, tx)
	if err != nil {
		return nil, err
	}
	statuses, err := statusesIn(ctx, tx)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(all, func(a, b Indexer) int { return cmp.Compare(a.Priority, b.Priority) })
	cats := x.categories(typ)
	now := storage.Now()
	var lines []Line
	for _, idx := range all {
		if !idx.On || !idx.AutoSearch || idx.Kind == RSS {
			continue
		}
		l := Line{Indexer: idx.ID, Name: idx.Name}
		if len(served(idx, cats)) == 0 {
			l.Skipped = &Outcome{Kind: NoCategories}
			lines = append(lines, l)
			continue
		}
		if _, wait := gate(idx, statuses[idx.ID], Usage{}, x.cfg.Roster, UseSearch, now.Time()); wait != nil {
			l.Skipped = wait
			lines = append(lines, l)
			continue
		}
		l.Query, err = reserve(ctx, tx, idx, UseSearch, now)
		if limit, ok := errors.AsType[*LimitError](err); ok {
			l.Skipped = &Outcome{Kind: LimitReached, Until: limit.Until}
		} else if err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, nil
}

func (x *Indexers) categories(typ works.TypeID) []newznab.ID {
	if f, ok := x.cfg.Types[typ]; ok {
		return f.Categories()
	}
	return nil
}

// served is the categories of idx that meet cats, a Content Type's: what
// its searches ask for, none when it doesn't serve the type. An Indexer
// with no categories set has every category it advertises.
func served(idx Indexer, cats []newznab.ID) []newznab.ID {
	have := idx.Categories
	if have == nil {
		have = idx.Caps.Categories
	}
	var out []newznab.ID
	for _, c := range have {
		if newznab.Meets([]newznab.ID{c}, cats) {
			out = append(out, c)
		}
	}
	return out
}

func statusesIn(ctx context.Context, r storage.Reader[file]) (map[ID]Status, error) {
	rows, err := db.New(r).ListStatuses(ctx)
	if err != nil {
		return nil, err
	}
	out := map[ID]Status{}
	for _, row := range rows {
		out[row.IndexerID] = statusOf(row)
	}
	return out, nil
}

// Search is one search: an Item's terms and the lines Plan asked.
type Search struct {
	Type  works.TypeID
	Terms Terms
	Asked []Line
}

// Answer is one asked Indexer's outcome and Releases.
type Answer struct {
	Indexer  ID
	Outcome  Outcome
	Releases []Release
}

// Search sends one job per asked line at once and returns their answers in
// line order once each answered or passed its deadline, Timings.Job from
// when its Node started it. A late job keeps being polled and is recorded
// when it answers. Every way an Indexer couldn't answer is its Answer's
// Outcome: removed, waiting, late or the site's own. It fails only when
// the library or ctx does.
func (x *Indexers) Search(ctx context.Context, s Search) ([]Answer, error) {
	answers := make([]Answer, len(s.Asked))
	errs := make([]error, len(s.Asked))
	var wg sync.WaitGroup
	for i, l := range s.Asked {
		wg.Go(func() { answers[i], errs[i] = x.searchOne(ctx, l, s) })
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	return answers, errors.Join(errs...)
}

func (x *Indexers) searchOne(ctx context.Context, l Line, s Search) (Answer, error) {
	out := Answer{Indexer: l.Indexer}
	idx, err := Get(ctx, x.cfg.Library, l.Indexer)
	if errors.Is(err, ErrNoIndexer) {
		out.Outcome = Outcome{Kind: Removed}
		return out, nil
	}
	if err != nil {
		return Answer{}, err
	}
	j, err := x.routed(ctx, l.Query, idx, UseSearch, searching.SearchCall{Query: query(idx.Caps, s.Terms)})
	if wait, ok := errors.AsType[*WaitError](err); ok {
		x.lose(ctx, l.Query)
		out.Outcome = wait.Outcome
		return out, nil
	}
	if err != nil {
		x.lose(ctx, l.Query)
		return Answer{}, err
	}
	for _, c := range served(idx, x.categories(s.Type)) {
		j.wire.Categories = append(j.wire.Categories, int(c))
	}
	a, st, err := x.run(ctx, j)
	switch {
	case ctx.Err() != nil:
		return Answer{}, fmt.Errorf("search Indexer %s: %w", idx.Name, ctx.Err())
	case errors.Is(err, ErrLate):
		out.Outcome = Outcome{Kind: Late}
	case errors.As(err, new(*UnavailableError)):
		out.Outcome = Outcome{Kind: Waiting, Node: j.node}
	case err != nil:
		return Answer{}, err
	default:
		out.Outcome = outcomeOf(a, st)
		for _, r := range a.Releases {
			out.Releases = append(out.Releases, Release{Indexer: idx.ID, Priority: idx.Priority, Release: r})
		}
	}
	return out, nil
}

// outcomeOf is a's Outcome, timed by the status it left.
func outcomeOf(a searching.Answer, st Status) Outcome {
	var stopped *searching.Stopped
	switch {
	case a.Outcome == nil && len(a.Releases) == 0:
		return Outcome{Kind: NoReleases}
	case a.Outcome == nil:
		return Outcome{Kind: Found, Releases: len(a.Releases)}
	case errors.As(a.Outcome, &stopped):
		return Outcome{Kind: IsStopped, Cause: stopped.Cause, Text: stopped.Text, Node: st.Node}
	case errors.As(a.Outcome, new(*searching.Limited)):
		return Outcome{Kind: IsLimited, Until: st.Until}
	}
	return Outcome{Kind: IsFailing, Until: st.Until}
}

// query is the query for terms under caps: terms' mode with every id
// parameter caps list for it, else a free-text search. It chooses by
// declared capability, so it never retries.
func query(c Caps, terms Terms) searching.Query {
	ids := map[string]string{}
	for _, p := range c.Modes[terms.Mode] {
		if id, ok := terms.IDs[p]; ok {
			ids[p] = id
		}
	}
	if len(ids) == 0 {
		return searching.Query{Mode: "search", Text: terms.Query}
	}
	return searching.Query{Mode: terms.Mode, IDs: ids}
}

// CategoriesOf is ids as Newznab category ids.
func CategoriesOf(ids []int) []newznab.ID {
	if ids == nil {
		return nil
	}
	out := make([]newznab.ID, len(ids))
	for i, id := range ids {
		out[i] = newznab.ID(id) //nolint:gosec // G115: Newznab category ids have at most six digits
	}
	return out
}
