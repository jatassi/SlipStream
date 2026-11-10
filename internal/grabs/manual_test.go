package grabs_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// counted is a scripted Searching Node that counts each Indexer's calls.
type counted struct {
	*indexerstest.Node
	mu    sync.Mutex
	calls map[string]int
}

func (c *counted) Run(ctx context.Context, j searching.Job) (searching.Poll, error) {
	c.mu.Lock()
	c.calls[j.Indexer]++
	c.mu.Unlock()
	return c.Node.Run(ctx, j)
}

func (c *counted) callsOf(id indexers.ID) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[id.String()]
}

func (c *counted) forget() {
	c.mu.Lock()
	c.calls = map[string]int{}
	c.mu.Unlock()
}

// errorLog keeps every record at level Error.
type errorLog struct {
	mu   sync.Mutex
	msgs []string
}

func (*errorLog) Enabled(_ context.Context, l slog.Level) bool { return l >= slog.LevelError }
func (e *errorLog) WithAttrs([]slog.Attr) slog.Handler         { return e }
func (e *errorLog) WithGroup(string) slog.Handler              { return e }

func (e *errorLog) Handle(_ context.Context, r slog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.msgs = append(e.msgs, r.Message)
	return nil
}

func (e *errorLog) errors() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.msgs)
}

// streaming is The Attic with Quick Site, answering 5 seconds into a job,
// and Slow Site, answering 60 seconds in and asked by Manual search
// alone.
type streaming struct {
	atticWorld
	node *counted
	log  *errorLog
}

var (
	quick = twinSite{name: "Quick Site", priority: 10, releases: []searching.Release{
		listing("The.Attic.2020.1080p.BluRay.x264-GRP", 50, "a"),
		listing("The.Attic.2020.1080p.WEB-DL-GRP", 90, "b"),
	}}
	slow = twinSite{name: "Slow Site", priority: 25, manualOnly: true, releases: []searching.Release{
		listing("The.Attic.2020.1080p.BluRay.x264-OTHER", 9, "c"),
	}}
)

func newStreaming(t *testing.T, sites ...twinSite) streaming {
	t.Helper()
	s := streaming{node: &counted{Node: &indexerstest.Node{}, calls: map[string]int{}}, log: &errorLog{}}
	s.atticWorld = newAttic(t, s.node, sites, nil, slog.New(s.log))
	for _, site := range sites {
		after := 5 * time.Second
		if site.manualOnly {
			after = time.Minute
		}
		s.node.Answer(s.ids[site.name], after, searching.Answer{Releases: site.releases})
	}
	return s
}

// running runs g's reconciler until the returned stop, which waits for it
// to end.
func running(t *testing.T, g *grabs.Grabs) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error)
	go func() { done <- g.Run(ctx) }()
	return func() {
		cancel()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

func (s streaming) start(t *testing.T, g *grabs.Grabs, scope grabs.ManualScope) grabs.ManualID {
	t.Helper()
	id, started, err := g.StartManualSearch(t.Context(), identity.Admin{}, s.work, scope)
	if err != nil || !started {
		t.Fatalf("start a Manual search = %v, %v, want started", started, err)
	}
	return id
}

// listed is each Release Manual runs stored, as "<Indexer> <title>",
// sorted.
func (s streaming) listed(t *testing.T) []string {
	t.Helper()
	return s.strings(t, `SELECT r.indexer_name || ' ' || r.title FROM grabs_release r
		JOIN grabs_search s ON s.id = r.search_id WHERE s.manual_id IS NOT NULL AND r.outcome = 'listed' ORDER BY 1`)
}

// heardFrom is the name of each Indexer whose line of a Manual run has an
// outcome, sorted.
func (s streaming) heardFrom(t *testing.T) []string {
	t.Helper()
	return s.strings(t, `SELECT l.indexer_name FROM grabs_search_indexer l JOIN grabs_search s ON s.id = l.search_id
		WHERE s.manual_id IS NOT NULL AND l.outcome IS NOT NULL ORDER BY 1`)
}

// manualRuns is the outcome of each Manual run, "open" while it runs.
func (s streaming) manualRuns(t *testing.T) []string {
	t.Helper()
	return s.strings(t, `SELECT coalesce(outcome, 'open') FROM grabs_search WHERE manual_id IS NOT NULL ORDER BY 1`)
}

func (s streaming) strings(t *testing.T, query string) []string {
	t.Helper()
	rows, err := s.lib.QueryContext(t.Context(), query)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func (s streaming) count(t *testing.T, query string) int {
	t.Helper()
	var n int
	if err := s.lib.QueryRowContext(t.Context(), query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

var (
	quickListed = []string{"Quick Site The.Attic.2020.1080p.BluRay.x264-GRP", "Quick Site The.Attic.2020.1080p.WEB-DL-GRP"}
	allListed   = []string{
		"Quick Site The.Attic.2020.1080p.BluRay.x264-GRP", "Quick Site The.Attic.2020.1080p.WEB-DL-GRP",
		"Slow Site The.Attic.2020.1080p.BluRay.x264-OTHER",
	}
)

func TestAManualSearchStoresEachIndexersReleasesAsItAnswersAndNeverGrabs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newStreaming(t, quick, slow)
		g := s.open()
		s.start(t, g, grabs.ManualScope{})
		stop := running(t, g)
		time.Sleep(10 * time.Second)
		synctest.Wait()
		if got := s.listed(t); !slices.Equal(got, quickListed) {
			t.Fatalf("before Slow Site answers the Manual search lists %q, want %q", got, quickListed)
		}
		if got := s.manualRuns(t); !slices.Equal(got, []string{"open"}) {
			t.Fatalf("before Slow Site answers the runs are %q, want one open", got)
		}
		time.Sleep(time.Minute)
		synctest.Wait()
		stop()
		if got := s.listed(t); !slices.Equal(got, allListed) {
			t.Fatalf("the Manual search lists %q, want %q", got, allListed)
		}
		if got := s.manualRuns(t); !slices.Equal(got, []string{"listed"}) {
			t.Fatalf("the runs are %q, want one listed", got)
		}
		if n := s.count(t, `SELECT count(*) FROM grabs_grab`); n != 0 {
			t.Fatalf("the Manual search grabbed %d Releases, want none", n)
		}
		if got := s.log.errors(); len(got) != 0 {
			t.Fatalf("the search logged errors %q", got)
		}
	})
}

func TestAnAnswerTwoControllersHearIsStoredOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newStreaming(t, quick, slow)
		first, second := s.open(), s.open()
		s.start(t, first, grabs.ManualScope{})
		stopFirst, stopSecond := running(t, first), running(t, second)
		time.Sleep(70 * time.Second)
		synctest.Wait()
		stopFirst()
		stopSecond()
		if got := s.listed(t); !slices.Equal(got, allListed) {
			t.Fatalf("the Manual search lists %q, want each Release once: %q", got, allListed)
		}
	})
}

func TestAManualSearchAfterARestartKeepsItsReleasesAndAsksOnlyTheSilentIndexer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newStreaming(t, quick, slow)
		before := s.open()
		s.start(t, before, grabs.ManualScope{})
		stop := running(t, before)
		time.Sleep(10 * time.Second)
		synctest.Wait()
		stop()
		if got := s.listed(t); !slices.Equal(got, quickListed) {
			t.Fatalf("at the restart the Manual search lists %q, want %q", got, quickListed)
		}
		s.node.forget()
		stop = running(t, s.open())
		time.Sleep(time.Minute)
		synctest.Wait()
		stop()
		if n := s.node.callsOf(s.ids[quick.name]); n != 0 {
			t.Fatalf("after the restart Quick Site was asked %d times, want none: it had answered", n)
		}
		if n := s.node.callsOf(s.ids[slow.name]); n == 0 {
			t.Fatal("after the restart Slow Site was never asked")
		}
		if got := s.listed(t); !slices.Equal(got, allListed) {
			t.Fatalf("the Manual search lists %q, want %q", got, allListed)
		}
		if got := s.manualRuns(t); !slices.Equal(got, []string{"listed"}) {
			t.Fatalf("the runs are %q, want one listed", got)
		}
	})
}

func TestSearchAgainEndsTheReplacedRunQuietly(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newStreaming(t, quick, slow)
		g := s.open()
		old := s.start(t, g, grabs.ManualScope{})
		stop := running(t, g)
		time.Sleep(10 * time.Second)
		synctest.Wait()
		if again, started, err := g.StartManualSearch(t.Context(), identity.Admin{}, s.work, grabs.ManualScope{}); err != nil || started || again != old {
			t.Fatalf("opening the same scope again = %v, %v, %v, want the search it has", again, started, err)
		}
		if again := s.start(t, g, grabs.ManualScope{Again: true}); again == old {
			t.Fatal("Search again kept the old Manual search")
		}
		synctest.Wait()
		if got := s.node.Canceled(); !slices.Contains(got, s.ids[slow.name].String()) {
			t.Fatalf("Search again canceled the jobs of %q, want Slow Site's", got)
		}
		time.Sleep(70 * time.Second)
		synctest.Wait()
		stop()
		if got := s.listed(t); !slices.Equal(got, allListed) {
			t.Fatalf("the new Manual search lists %q, want %q", got, allListed)
		}
		if n := len(s.node.Searches()); n != 4 {
			t.Fatalf("the Indexers ran %d searches, want 4: two for each Manual search, none again", n)
		}
		if got := s.log.errors(); len(got) != 0 {
			t.Fatalf("the replaced run logged errors %q", got)
		}
	})
}

func TestAWorkDeletedMidSearchEndsItsRunQuietly(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newStreaming(t, quick, slow)
		g := s.open()
		s.start(t, g, grabs.ManualScope{})
		stop := running(t, g)
		time.Sleep(10 * time.Second)
		synctest.Wait()
		present := func(_ context.Context, _ *storage.Tx[storage.Library], ws []works.WorkID) ([]works.WorkID, error) {
			return ws, nil
		}
		if _, err := s.cfg.Works.Delete(t.Context(), identity.Admin{}, []works.WorkID{s.work}, present); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		stop()
		if n := len(s.node.Searches()); n != 2 {
			t.Fatalf("the Indexers ran %d searches, want 2: none again once the Work was gone", n)
		}
		if n := s.count(t, `SELECT count(*) FROM grabs_search`); n != 0 {
			t.Fatalf("%d runs outlived their Work", n)
		}
		if got := s.log.errors(); len(got) != 0 {
			t.Fatalf("the run of a deleted Work logged errors %q", got)
		}
	})
}

func TestManualAndAutomaticRunsKeepEachOthersReleases(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		unwanted := twinSite{name: "Quick Site", priority: 10, releases: []searching.Release{listing("The.Attic.2020.720p.WEB-DL-GRP", 50, "a")}}
		s := newStreaming(t, unwanted, slow)
		g := s.open()
		stop := running(t, g)
		automatic := func() {
			if _, err := g.SearchWork(t.Context(), identity.Admin{}, s.work, grabs.Scope{}); err != nil {
				t.Fatal(err)
			}
			time.Sleep(10 * time.Second)
			synctest.Wait()
		}
		automaticRows := `SELECT count(*) FROM grabs_release r JOIN grabs_search s ON s.id = r.search_id WHERE s.manual_id IS NULL`
		automatic()
		s.start(t, g, grabs.ManualScope{})
		time.Sleep(70 * time.Second)
		synctest.Wait()
		if n := s.count(t, automaticRows); n != 1 {
			t.Fatalf("after a Manual run finished the automatic run keeps %d Releases, want 1", n)
		}
		automatic()
		stop()
		if n := s.count(t, automaticRows); n != 1 {
			t.Fatalf("after a second automatic run the automatic runs keep %d Releases, want the latest's 1", n)
		}
		if got := s.listed(t); len(got) != 2 {
			t.Fatalf("after an automatic run finished the Manual search lists %q, want both its Releases", got)
		}
		searches, err := g.Searches(t.Context(), identity.Admin{}, s.work)
		if err != nil {
			t.Fatal(err)
		}
		if len(searches) != 2 || slices.ContainsFunc(searches, func(r grabs.Search) bool { return !r.Manual.IsZero() }) {
			t.Fatalf("the Work's searches are %+v, want its 2 automatic runs alone", searches)
		}
		if _, err := g.Releases(t.Context(), identity.Admin{}, manualRun(t, s)); !errors.Is(err, grabs.ErrNoSearch) {
			t.Fatalf("the Releases of a Manual run = %v, want ErrNoSearch", err)
		}
	})
}

func manualRun(t *testing.T, s streaming) grabs.SearchID {
	t.Helper()
	var id storage.UUID
	if err := s.lib.QueryRowContext(t.Context(), `SELECT id FROM grabs_search WHERE manual_id IS NOT NULL`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return grabs.SearchID{UUID: id}
}

func TestAManualRunWhoseIndexersAreAllSkippedFinishes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newStreaming(t, quick)
		if _, err := s.cfg.Indexers.Update(t.Context(), identity.Admin{}, s.ids[quick.name], indexers.Edit{Categories: &[]newznab.ID{5000}}, indexers.Typed{}); err != nil {
			t.Fatal(err)
		}
		g := s.open()
		s.start(t, g, grabs.ManualScope{})
		stop := running(t, g)
		synctest.Wait()
		stop()
		if got := s.manualRuns(t); !slices.Equal(got, []string{"listed"}) {
			t.Fatalf("the runs are %q, want one listed", got)
		}
		if got := s.heardFrom(t); !slices.Equal(got, []string{"Quick Site"}) {
			t.Fatalf("the run has outcomes for %q, want Quick Site skipped", got)
		}
		if n := len(s.node.Jobs()); n != 0 {
			t.Fatalf("the Indexers ran %d jobs, want none", n)
		}
	})
}

func TestATypedQueryGoesToEachIndexerAsText(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newStreaming(t, quick, slow)
		g := s.open()
		s.start(t, g, grabs.ManualScope{Query: "attic 2020 remux"})
		stop := running(t, g)
		time.Sleep(70 * time.Second)
		synctest.Wait()
		stop()
		queries := s.node.Searches()
		if len(queries) != 2 || slices.ContainsFunc(queries, func(q searching.Query) bool { return q.Mode != "search" || q.Text != "attic 2020 remux" }) {
			t.Fatalf("the Indexers were asked %+v, want the typed text twice", queries)
		}
	})
}

func TestAManualSearchStartRefusesWhileReconcilingAndForACellTheItemHasNot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newStreaming(t, quick)
		var id storage.UUID
		if err := s.lib.QueryRowContext(t.Context(), `SELECT id FROM works_item`).Scan(&id); err != nil {
			t.Fatal(err)
		}
		cell := &files.CellKey{Item: works.ItemID{UUID: id}, Edition: "any", Target: targets.TargetID{UUID: storage.NewUUID()}}
		if _, _, err := s.open().StartManualSearch(t.Context(), identity.Admin{}, s.work, grabs.ManualScope{
			Part: grabs.Within(works.Within{Items: []works.ItemID{cell.Item}}), Cell: cell,
		}); !errors.Is(err, grabs.ErrNoCell) {
			t.Fatalf("a Manual search of a cell the Item hasn't = %v, want ErrNoCell", err)
		}
		s.cfg.Reconciling = func(context.Context, storage.Reader[storage.Library]) (bool, error) { return true, nil }
		if _, _, err := s.open().StartManualSearch(t.Context(), identity.Admin{}, s.work, grabs.ManualScope{}); !errors.Is(err, grabs.ErrReconciling) {
			t.Fatalf("a Manual search while Reconciling = %v, want ErrReconciling", err)
		}
		if n := s.count(t, `SELECT count(*) FROM grabs_manual`); n != 0 {
			t.Fatalf("the refused starts left %d Manual searches", n)
		}
	})
}
