package grabs

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

const rssName = "RSS sync"

// rssTask is RSS sync: one unit per Indexer that is on with RSS on,
// placed as the Indexer health check's.
func (gr *Grabs) rssTask() tasks.Task {
	return tasks.Task{
		ID:               RSSTaskID,
		Name:             "RSS sync",
		About:            "Reads the new uploads of each Indexer with RSS on, and grabs any that are missing from the library or upgrade a file there.",
		Group:            tasks.GroupOf(role.Searching.Group()),
		Cadence:          tasks.Every{Interval: RSSEvery},
		WhileReconciling: tasks.Pauses,
		Noun:             setting.Noun{One: "Indexer", Many: "Indexers"},
		Units:            gr.cfg.Indexers.RSSUnits,
		Shown:            anyRSS,
		Off: func(ctx context.Context, r storage.Reader[file]) (bool, error) {
			s, err := db.New(r).GetSettings(ctx)
			return !RSSOn.Value(s.RssOn), err
		},
		Run: func(ctx context.Context, u tasks.Unit, p *tasks.Progress) error {
			note, err := gr.syncFeed(ctx, indexers.IDOf(u.Key))
			if err != nil {
				return err
			}
			p.Note(note)
			return nil
		},
		Follows: []live.Event{indexers.ChangedEvent},
		Health:  string(health.IndexersID),
	}
}

// anyRSS shows RSS sync while any Indexer has RSS on.
func anyRSS(ctx context.Context, r storage.Reader[file]) (bool, error) {
	all, err := indexers.All(ctx, r)
	return slices.ContainsFunc(all, func(idx indexers.Indexer) bool { return idx.RSS }), err
}

// RSSIn is RSS sync's cadence and whether it is on, read in r: what each
// Indexer's search limit keeps slots for.
func (gr *Grabs) RSSIn(ctx context.Context, r storage.Reader[file]) (time.Duration, bool, error) {
	s, err := db.New(r).GetSettings(ctx)
	if err != nil {
		return 0, false, err
	}
	every, err := gr.cfg.Tasks.EveryIn(ctx, r, RSSTaskID)
	if err != nil {
		return 0, false, err
	}
	return RSSEvery.Value(every), RSSOn.Value(s.RssOn), nil
}

// syncFeed is one RSS sync of Indexer id: it reads the feed's new
// Releases, matches them to Works, and in one fenced Write decides each
// matched Work and moves the feed's position past them. A crash before
// that Write reads the same Releases again next time. It answers the
// unit's note.
func (gr *Grabs) syncFeed(ctx context.Context, id indexers.ID) (string, error) {
	feed, err := gr.readFeed(ctx, id)
	if err != nil {
		return "", err
	}
	var ms []matched
	err = gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		ms, err = gr.match(ctx, r, feed)
		return err
	})
	if err != nil {
		return "", err
	}
	t := rssTally{read: len(feed.Releases)}
	for _, m := range ms {
		t.matched += len(m.releases)
	}
	var took []Grab
	var runs []Search
	var seen indexers.Seen
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := gr.fence(ctx, tx); err != nil {
			return err
		}
		for _, m := range ms {
			run, grabs, kept, err := gr.decideFeed(ctx, tx, feed.Name, m)
			if err != nil {
				return err
			}
			if kept {
				runs = append(runs, run)
			}
			took = append(took, grabs...)
		}
		var err error
		seen, err = gr.cfg.Indexers.Saw(ctx, tx, feed)
		return err
	})
	if errors.Is(err, ErrReconciling) {
		return "", tasks.PausedWhileReconciling(rssName)
	}
	if err != nil {
		return "", err
	}
	gr.announceFeed(ctx, seen, runs, took)
	t.grabbed = len(took)
	return t.words(), nil
}

// readFeed is Indexer id's new Releases, each with no published time
// given the time it was heard. RSS sync off and an Indexer's skip are the
// unit's owned skips.
func (gr *Grabs) readFeed(ctx context.Context, id indexers.ID) (indexers.RSSFetch, error) {
	var on bool
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		s, err := db.New(r).GetSettings(ctx)
		on = RSSOn.Value(s.RssOn)
		return err
	})
	if err != nil {
		return indexers.RSSFetch{}, err
	}
	if !on {
		return indexers.RSSFetch{}, tasks.SkipOwned("RSS sync is off in Settings.")
	}
	feed, err := gr.cfg.Indexers.Recent(ctx, id)
	if skip, ok := errors.AsType[*indexers.FeedSkipped](err); ok {
		return indexers.RSSFetch{}, tasks.SkipOwned(skip.Sentence)
	}
	if err != nil {
		return indexers.RSSFetch{}, err
	}
	heard := gr.cfg.Clock()
	for i := range feed.Releases {
		if feed.Releases[i].Published.IsZero() {
			feed.Releases[i].Published = heard
		}
	}
	return feed, nil
}

// announceFeed tells of a committed RSS sync: the feed's new position,
// the runs it kept and the Grabs it took.
func (gr *Grabs) announceFeed(ctx context.Context, seen indexers.Seen, runs []Search, took []Grab) {
	gr.cfg.Indexers.Seen(ctx, seen)
	for _, run := range runs {
		gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: run.ID.String(), WorkID: run.WorkID.String()})
	}
	for _, g := range took {
		gr.cfg.Indexers.Taken(ctx, g.Ticket)
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: g.WorkID.String()})
	}
	if len(took) > 0 {
		gr.wake()
	}
}

// matched is one Work's share of a feed: its tree, how its Releases are
// judged, and the feed's Releases that name it, in feed order.
type matched struct {
	tree     works.Tree
	judging  judging
	releases []indexers.Release
}

// match finds the Work each of f's Releases names, read in r: for each
// Content Type the feed may match, the library title search's Works whose
// title and year the Release's name matches under the type's clean-title
// rule, and whose ids don't name another Work. A Release goes to the
// first Work it matches. Expect is read once per Work.
func (gr *Grabs) match(ctx context.Context, r storage.Reader[file], f indexers.RSSFetch) ([]matched, error) {
	known, err := gr.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	expects := map[works.WorkID]indexers.Expect{}
	byWork := map[works.WorkID]int{}
	var out []matched
	for _, rel := range f.Releases {
		work, ok, err := gr.matchOne(ctx, r, rel, f.Types, known, expects)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		i, seen := byWork[work]
		if !seen {
			tree, err := gr.cfg.Works.TreeIn(ctx, r, work)
			if err != nil {
				return nil, err
			}
			j, err := gr.judgingOf(ctx, r, tree, nil, expects[work])
			if err != nil {
				return nil, err
			}
			i, byWork[work] = len(out), len(out)
			out = append(out, matched{tree: tree, judging: j})
		}
		out[i].releases = append(out[i].releases, rel)
	}
	return out, nil
}

// matchOne is the first Work of types that rel's name matches.
func (gr *Grabs) matchOne(ctx context.Context, r storage.Reader[file], rel indexers.Release, types []works.TypeID, known targets.Cuts, expects map[works.WorkID]indexers.Expect) (work works.WorkID, hit bool, err error) {
	for _, typ := range types {
		names, ok := gr.cfg.Names[typ]
		if !ok {
			continue
		}
		rd := release.Title(rel.Title, names, known.For(typ))
		ids := maps.Clone(rd.Facts.IDs)
		if ids == nil {
			ids = map[string]string{}
		}
		maps.Copy(ids, rel.IDs)
		for _, title := range rd.Name.Titles {
			found, err := gr.cfg.Works.MatchIn(ctx, r, title, &typ)
			if err != nil {
				return works.WorkID{}, false, err
			}
			if work, hit, err = gr.firstMatch(ctx, r, typ, found, rd.Name, ids, expects); err != nil || hit {
				return work, hit, err
			}
		}
	}
	return works.WorkID{}, false, nil
}

// firstMatch is the first of found, Works of typ, whose Expect's claim n
// with ids matches. It caches each Work's Expect in expects.
func (gr *Grabs) firstMatch(ctx context.Context, r storage.Reader[file], typ works.TypeID, found []works.WorkID, n release.Name, ids map[string]string, expects map[works.WorkID]indexers.Expect) (work works.WorkID, hit bool, err error) {
	for _, work = range found {
		expect, ok := expects[work]
		if !ok {
			if expect, err = gr.cfg.Terms[typ].Expect(ctx, r, work); err != nil {
				return works.WorkID{}, false, err
			}
			expects[work] = expect
		}
		if m, _ := release.Match(n, ids, expect.Claim, gr.cfg.Names[typ]); m == release.Matched {
			return work, true, nil
		}
	}
	return works.WorkID{}, false, nil
}

// decideFeed decides m, the Releases feed listed for one Work, in tx,
// and answers its run, the Grabs it made or joined and whether it kept
// the run. A decision that grabs or waits keeps its RSS run, finished,
// with its Releases and the Items its claims hold, and the Work keeps its
// latest RSS run. One that takes nothing keeps none.
func (gr *Grabs) decideFeed(ctx context.Context, tx *storage.Tx[file], feed string, m matched) (Search, []Grab, bool, error) {
	q := db.New(tx)
	now := storage.Now()
	run := Search{ID: SearchID{UUID: storage.NewUUID()}, WorkID: m.tree.ID, Trigger: searches.RSS, Started: now.Time()}
	if err := q.InsertRSSSearch(ctx, db.InsertRSSSearchParams{ID: run.ID, WorkID: run.WorkID, StartedAt: now}); err != nil {
		return run, nil, false, err
	}
	d, err := gr.decideIn(ctx, tx, rssInput(run, assessAll(m.judging, dedupe(listedOf(m.releases)))))
	if err != nil {
		return run, nil, false, err
	}
	if len(d.grabs) == 0 && len(d.waits) == 0 {
		return run, nil, false, q.DeleteRSSSearch(ctx, run.ID)
	}
	for _, it := range claimedItems(m.tree, d) {
		if err := q.InsertSearchItem(ctx, db.InsertSearchItemParams{SearchID: run.ID, ItemID: it}); err != nil {
			return run, nil, false, err
		}
	}
	names := make(map[int]string, len(m.releases))
	for i := range m.releases {
		names[i] = feed
	}
	if err := gr.storeReleases(ctx, q, run.ID, d.rows, names, len(d.grabs) > 0, nil); err != nil {
		return run, nil, false, err
	}
	if err := commitWaits(ctx, q, run, d.waits, d.at); err != nil {
		return run, nil, false, err
	}
	s := db.SettleRSSSearchParams{ID: run.ID, Outcome: new(Waiting), Candidates: int64(d.candidates)}
	if len(d.grabs) > 0 {
		notes, err := encodeReasons(d.notes)
		if err != nil {
			return run, nil, false, err
		}
		s.Outcome, s.Grabs, s.Notes = new(Grabbed), int64(len(d.grabs)), &notes
	}
	if err := q.SettleRSSSearch(ctx, s); err != nil {
		return run, nil, false, err
	}
	return run, d.grabs, true, q.PruneRSS(ctx, db.PruneRSSParams{WorkID: run.WorkID, Kept: run.ID})
}

// claimedItems is the Items of tree that d's Grabs and waits hold a cell
// of, in tree order: what an RSS run searched, which a rerun reads.
func claimedItems(tree works.Tree, d decisionOut) []works.ItemID {
	held := map[works.ItemID]bool{}
	for _, g := range d.grabs {
		for _, c := range g.Targets {
			held[c.Key.Item] = true
		}
	}
	for _, w := range d.waits {
		for _, k := range w.cells {
			held[k.Item] = true
		}
	}
	var out []works.ItemID
	for _, it := range tree.Items {
		if held[it.ID] {
			out = append(out, it.ID)
		}
	}
	return out
}

// rssTally is what one RSS sync unit read, matched and grabbed.
type rssTally struct{ read, matched, grabbed int }

// words is t as the unit's note: "Read 87 new releases. 3 matched a
// Work, and 1 was grabbed."
func (t rssTally) words() string {
	switch {
	case t.read == 0:
		return "Read no new releases."
	case t.matched == 0:
		return fmt.Sprintf("Read %s. None matched a Work.", releasesWords(t.read))
	}
	grabbed := fmt.Sprintf("%d was grabbed", t.grabbed)
	switch t.grabbed {
	case 0:
		grabbed = "none was grabbed"
	case 1:
	default:
		grabbed = fmt.Sprintf("%d were grabbed", t.grabbed)
	}
	return fmt.Sprintf("Read %s. %d matched a Work, and %s.", releasesWords(t.read), t.matched, grabbed)
}

func releasesWords(n int) string {
	if n == 1 {
		return "1 new release"
	}
	return fmt.Sprintf("%d new releases", n)
}
