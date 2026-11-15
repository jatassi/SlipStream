package indexers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// markGUIDs is how many of a feed's newest GUIDs its position keeps.
const markGUIDs = 100

// RSSFetch is one RSS fetch of an Indexer: its new Releases since its
// stored position, newest first and clamped, and the Content Types they
// may match.
type RSSFetch struct {
	Indexer  ID
	Name     string
	Types    []works.TypeID
	Releases []Release
	// Pages is how many pages the fetch read, at least 1.
	Pages int
	// Gap is why the fetch didn't reach the stored position, GapNone when
	// it did.
	Gap     GapKind
	mark    feedMark
	fetched time.Time
}

// feedMark is a feed's position: its newest GUIDs, newest first, and its
// newest publish time, zero for none.
type feedMark struct {
	guids     []string
	published time.Time
}

// covers reports whether r is at or before m: one of m's GUIDs, or
// published no later than m's newest.
func (m feedMark) covers(r searching.Release) bool {
	return (r.GUID != "" && slices.Contains(m.guids, r.GUID)) ||
		(!m.published.IsZero() && !r.Published.IsZero() && !r.Published.After(m.published))
}

// after is m moved past rs, newest first.
func (m feedMark) after(rs []Release) feedMark {
	out := feedMark{published: m.published}
	for _, r := range rs {
		if r.GUID != "" && !slices.Contains(out.guids, r.GUID) {
			out.guids = append(out.guids, r.GUID)
		}
		if r.Published.After(out.published) {
			out.published = r.Published
		}
	}
	for _, g := range m.guids {
		if !slices.Contains(out.guids, g) {
			out.guids = append(out.guids, g)
		}
	}
	out.guids = out.guids[:min(len(out.guids), markGUIDs)]
	return out
}

// GapKind is why an RSS fetch didn't reach its feed's stored position.
type GapKind string

const (
	GapNone GapKind = ""
	// GapNoPaging is a feed that can't list older uploads, or stopped
	// listing them partway.
	GapNoPaging GapKind = "no_paging"
	// GapPageCap is a fetch that ran out of pages, or a page cut to
	// Timings.RSSPageItems.
	GapPageCap GapKind = "page_cap"
	// GapLate is a page past the fetch's search wait.
	GapLate GapKind = "late"
)

// FeedSkipped is a fetch Recent didn't read, worded for the RSS sync
// unit's line: the Indexer stopped, limited, failing inside its backoff,
// at its search limit, waiting on its home Node, late, deleted, or turned
// off since its unit was listed, whose Outcome is zero.
type FeedSkipped struct {
	Outcome  Outcome
	Sentence string
}

func (e *FeedSkipped) Error() string { return "the RSS fetch was skipped: " + e.Sentence }

// Recent fetches Indexer id's feed. Each page reserves its own recent
// query in its own Write and goes to the Node the Indexer routes to, with
// the categories of the Content Types it serves. One search wait, from the
// first ask, spans the whole page-back. Where Caps.RecentPaging, it pages
// back by offset until a page reaches the stored position or
// Timings.RSSPages run out. A first page that can't be read is a
// *FeedSkipped. A later one ends the page-back with a gap: late for a page
// past the wait, no_paging for any other, or a page that lists nothing
// new. The clamps: a publish time after receipt is the receipt, a page
// over Timings.RSSPageItems is cut and a page_cap gap, and a GUID over
// Timings.GUIDBytes is dropped with its Release. It writes no position.
func (x *Indexers) Recent(ctx context.Context, id ID) (RSSFetch, error) {
	var idx Indexer
	var mark *feedMark
	var wait time.Duration
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if idx, err = x.get(ctx, r, id); err != nil {
			return err
		}
		if mark, err = markOf(ctx, r, id); err != nil {
			return err
		}
		wait, err = searchWait(ctx, r)
		return err
	})
	if errors.Is(err, ErrNoIndexer) {
		return RSSFetch{}, &FeedSkipped{Outcome: Outcome{Kind: Removed}, Sentence: "Skipped because the Indexer was deleted."}
	}
	if err != nil {
		return RSSFetch{}, err
	}
	switch {
	case !idx.On:
		return RSSFetch{}, &FeedSkipped{Sentence: "Skipped because " + idx.Name + " is off."}
	case !idx.RSS:
		return RSSFetch{}, &FeedSkipped{Sentence: "Skipped because RSS is off on " + idx.Name + "."}
	}
	f := RSSFetch{Indexer: idx.ID, Name: idx.Name}
	var cats []newznab.ID
	f.Types, cats = x.feedTypes(idx)
	if len(f.Types) == 0 {
		return RSSFetch{}, &FeedSkipped{Outcome: Outcome{Kind: NoCategories}, Sentence: "Skipped because " + idx.Name + " has none of the categories SlipStream reads."}
	}
	if err := x.pageBack(ctx, &f, idx, mark, cats, time.Now().Add(wait), wait); err != nil {
		return RSSFetch{}, err
	}
	base := feedMark{}
	if mark != nil {
		base = *mark
	}
	f.mark, f.fetched = base.after(f.Releases), time.Now()
	return f, nil
}

// pageBack reads idx's pages into f from the newest, until one reaches
// mark or leaves a gap. It fails when the first page can't be read.
func (x *Indexers) pageBack(ctx context.Context, f *RSSFetch, idx Indexer, mark *feedMark, cats []newznab.ID, deadline time.Time, wait time.Duration) error {
	seen := map[string]bool{}
	for offset := 0; ; f.Pages++ {
		items, at, err := x.recentPage(ctx, idx, offset, cats, wait, deadline)
		if skip, ok := errors.AsType[*FeedSkipped](err); ok && f.Pages > 0 {
			f.Gap = GapNoPaging
			if skip.Outcome.Kind == Late {
				f.Gap = GapLate
			}
			return nil
		}
		if err != nil {
			return err
		}
		cut := len(items) > x.cfg.Timings.RSSPageItems
		items = items[:min(len(items), x.cfg.Timings.RSSPageItems)]
		offset += len(items)
		kept, reached := x.newOf(idx, items, at, mark, seen)
		f.Releases = append(f.Releases, kept...)
		f.Gap = pageGap(reached || len(items) == 0, cut, idx.Caps.RecentPaging, len(kept), f.Pages+1, x.cfg.Timings.RSSPages)
		if reached || len(items) == 0 || f.Gap != GapNone {
			f.Pages++
			return nil
		}
	}
}

// newOf is the Releases of items, received at at, that are past mark and
// not yet seen in this fetch, clamped, and whether a page reached mark.
// No mark is reached by any page.
func (x *Indexers) newOf(idx Indexer, items []searching.Release, at time.Time, mark *feedMark, seen map[string]bool) ([]Release, bool) {
	reached := mark == nil
	var out []Release
	for _, r := range items {
		if r.Published.After(at) {
			r.Published = at
		}
		switch {
		case len(r.GUID) > x.cfg.Timings.GUIDBytes:
			continue
		case mark != nil && mark.covers(r):
			reached = true
			continue
		case r.GUID != "" && seen[r.GUID]:
			continue
		}
		seen[r.GUID] = r.GUID != ""
		out = append(out, Release{Indexer: idx.ID, Priority: idx.Priority, Release: r})
	}
	return out, reached
}

// pageGap is the gap the page-th page leaves when it is the last one read:
// page_cap for a cut page, else GapNone once it reached the position or
// while the page-back may go on.
func pageGap(reached, cut, paging bool, fresh, page, pages int) GapKind {
	switch {
	case cut:
		return GapPageCap
	case reached:
		return GapNone
	case !paging || (page > 1 && fresh == 0):
		return GapNoPaging
	case page >= pages:
		return GapPageCap
	}
	return GapNone
}

// feedTypes is the Content Types idx's feed may match and the categories
// its fetch asks for: an RSS feed's Media with none, else each type whose
// categories it serves, with those categories.
func (x *Indexers) feedTypes(idx Indexer) ([]works.TypeID, []newznab.ID) {
	if idx.Kind == RSS {
		return slices.Clone(idx.Media), nil
	}
	var types []works.TypeID
	var cats []newznab.ID
	for _, typ := range slices.Sorted(maps.Keys(x.cfg.Types)) {
		served := served(idx, x.categories(typ))
		if len(served) == 0 {
			continue
		}
		types = append(types, typ)
		for _, c := range served {
			if !slices.Contains(cats, c) {
				cats = append(cats, c)
			}
		}
	}
	return types, cats
}

// recentPage reserves and reads one page of idx's feed from offset, and
// when it was received: *FeedSkipped when the gate, the search limit or
// the page's answer kept it from being read.
func (x *Indexers) recentPage(ctx context.Context, idx Indexer, offset int, cats []newznab.ID, wait time.Duration, deadline time.Time) ([]searching.Release, time.Time, error) {
	var q QueryID
	err := x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		st, err := status(ctx, tx, idx.ID)
		if err != nil {
			return err
		}
		now := storage.Now()
		if _, held := gate(idx, st, Usage{}, x.cfg.Roster, UseRecent, now.Time()); held != nil {
			return x.feedSkipped(idx.Name, *held)
		}
		q, err = reserve(ctx, tx, idx, UseRecent, 0, now)
		if limit, ok := errors.AsType[*LimitError](err); ok {
			return x.feedSkipped(idx.Name, Outcome{Kind: LimitReached, Until: limit.Until})
		}
		return err
	})
	if err != nil {
		return nil, time.Time{}, err
	}
	j, err := x.routed(ctx, q, idx, UseRecent, searching.RecentCall{Offset: offset})
	if held, ok := errors.AsType[*WaitError](err); ok {
		x.lose(ctx, q)
		return nil, time.Time{}, x.feedSkipped(idx.Name, held.Outcome)
	}
	if err != nil {
		x.lose(ctx, q)
		return nil, time.Time{}, err
	}
	j.wait, j.deadline = wait, deadline
	for _, c := range cats {
		j.wire.Categories = append(j.wire.Categories, int(c))
	}
	a, st, err := x.run(ctx, j)
	switch {
	case ctx.Err() != nil:
		return nil, time.Time{}, fmt.Errorf("read the feed of Indexer %s: %w", idx.Name, ctx.Err())
	case errors.Is(err, ErrLate):
		return nil, time.Time{}, x.feedSkipped(idx.Name, Outcome{Kind: Late})
	case errors.As(err, new(*UnavailableError)):
		return nil, time.Time{}, x.feedSkipped(idx.Name, Outcome{Kind: Waiting, Node: j.node})
	case err != nil:
		return nil, time.Time{}, err
	case a.Outcome != nil:
		return nil, time.Time{}, x.feedSkipped(idx.Name, outcomeOf(a, st, idx.Home))
	}
	return a.Releases, time.Now(), nil
}

// feedSkipped is the fetch of Indexer name skipped for o, in words.
func (x *Indexers) feedSkipped(name string, o Outcome) *FeedSkipped {
	var s string
	switch o.Kind {
	case LimitReached:
		s = fmt.Sprintf("Skipped because %s is at its search limit until %s.", name, x.Clock(o.Until))
	case IsLimited:
		s = fmt.Sprintf("Skipped because %s is limiting requests until %s.", name, x.Clock(o.Until))
	case IsStopped:
		s = fmt.Sprintf("Skipped because %s is stopped. Open Health to see how to fix it.", name)
	case Waiting:
		s = fmt.Sprintf("Skipped because %s is waiting for %s.", name, x.cfg.Roster.Name(o.Node))
	case Late:
		s = fmt.Sprintf("Skipped because %s didn't respond in time. The next RSS sync tries it again.", name)
	case IsFailing, Found, NoReleases, NoCategories, Removed, RSSOnly:
		s = fmt.Sprintf("Skipped because %s isn't responding. The next RSS sync tries it again.", name)
	}
	return &FeedSkipped{Outcome: o, Sentence: s}
}

// markOf is Indexer id's stored position in r, nil before its first
// fetch.
func markOf(ctx context.Context, r storage.Reader[file], id ID) (*feedMark, error) {
	row, err := db.New(r).GetRSS(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m := feedMark{published: row.LastPublishedAt.Time()}
	if err := decode(row.LastGuids, &m.guids); err != nil {
		return nil, fmt.Errorf("indexer %s's feed position: %w", id, err)
	}
	return &m, nil
}

// Seen is the feeds whose position or gap Saw wrote, for Seen to announce.
type Seen struct{ changed []ID }

// Saw writes f's position, its pages and its gap in tx, the Write that
// decided its Releases: gap_at kept from the first fetch of an unbroken
// gap, cleared by a fetch that reached the position. A feed whose Indexer
// was deleted since is left alone.
func (x *Indexers) Saw(ctx context.Context, tx *storage.Tx[file], f RSSFetch) (Seen, error) {
	_, err := Get(ctx, tx, f.Indexer)
	if errors.Is(err, ErrNoIndexer) {
		return Seen{}, nil
	}
	if err != nil {
		return Seen{}, err
	}
	q := db.New(tx)
	guids, err := json.Marshal(append([]string{}, f.mark.guids...))
	if err != nil {
		return Seen{}, fmt.Errorf("encode the feed position: %w", err)
	}
	p := db.PutRSSParams{
		IndexerID: f.Indexer, LastGuids: string(guids), LastPublishedAt: storage.FromTime(f.mark.published),
		LastFetchAt: storage.FromTime(f.fetched), LastPages: int64(max(1, f.Pages)),
	}
	if f.Gap != GapNone {
		p.GapAt, p.GapKind = storage.FromTime(f.fetched), new(string(f.Gap))
		switch row, err := q.GetRSS(ctx, f.Indexer); {
		case err == nil && !row.GapAt.IsZero():
			p.GapAt = row.GapAt
		case err != nil && !errors.Is(err, storage.ErrNoRows):
			return Seen{}, err
		}
	}
	if err := q.PutRSS(ctx, p); err != nil {
		return Seen{}, err
	}
	return Seen{changed: []ID{f.Indexer}}, nil
}

// Seen publishes indexers.rss-changed for s once its Write committed.
func (x *Indexers) Seen(ctx context.Context, s Seen) {
	for _, id := range s.changed {
		x.cfg.RSSChanged.Publish(ctx, RSSChanged{IndexerID: id.String()})
	}
}

// RSSUnits is one RSS sync unit per Indexer that is on with RSS on, placed
// as the health check's. Its Key is the Indexer id, which IDOf reads.
func (x *Indexers) RSSUnits(ctx context.Context, r storage.Reader[file]) ([]tasks.Unit, error) {
	all, err := x.all(ctx, r)
	if err != nil {
		return nil, err
	}
	statuses, err := statusesIn(ctx, r)
	if err != nil {
		return nil, err
	}
	var out []tasks.Unit
	for _, idx := range all {
		if idx.On && idx.RSS {
			out = append(out, tasks.Unit{Key: tasks.UnitKey(idx.ID.String()), Label: idx.Name, Place: placeOf(idx, statuses[idx.ID])})
		}
	}
	return out, nil
}

// IDOf is the Indexer of an RSSUnits unit's key, the zero ID, which no
// Indexer has, for any other key.
func IDOf(k tasks.UnitKey) ID {
	u, err := storage.ParseUUID(string(k))
	if err != nil {
		return ID{}
	}
	return ID{UUID: u}
}

// placeOf is where a task unit of idx at st runs: tied to its home while
// it waits for home, else portable.
func placeOf(idx Indexer, st Status) tasks.Place {
	if portable(idx, st) {
		return tasks.PortableOn(role.Searching, idx.Home)
	}
	return tasks.TiedTo(idx.Home)
}

// FeedFact is an Indexer's last RSS fetch, for the Indexer page.
type FeedFact struct {
	Fetched time.Time
	// Gap is why the last fetches didn't reach the feed's position, and
	// GapSince since when, GapNone while they did.
	Gap      GapKind
	GapSince time.Time
}

// feedFacts is every fetched feed's FeedFact in r.
func feedFacts(ctx context.Context, r storage.Reader[file]) (map[ID]FeedFact, error) {
	rows, err := db.New(r).ListRSS(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[ID]FeedFact, len(rows))
	for _, row := range rows {
		out[row.IndexerID] = FeedFact{Fetched: row.LastFetchAt.Time(), Gap: GapKind(deref(row.GapKind)), GapSince: row.GapAt.Time()}
	}
	return out, nil
}
