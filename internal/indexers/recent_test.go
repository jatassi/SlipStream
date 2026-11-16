package indexers_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

// heard records what a Publisher was told.
type heard[T any] struct {
	mu  sync.Mutex
	got []T
}

func (h *heard[T]) Publish(_ context.Context, v T) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.got = append(h.got, v)
}

func (h *heard[T]) all() []T {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.got)
}

// pagingCaps are indexerstest.Caps with RSS sync paging back by offset.
const pagingCaps = `{"modes":{"search":["q"]},"categories":[2000],"recentPaging":true}`

// uploads are Releases with GUIDs prefix-n down to prefix-1, newest
// first, a second apart before at.
func uploads(prefix string, n int, at time.Time) []searching.Release {
	out := make([]searching.Release, n)
	for i := range out {
		g := fmt.Sprintf("%s-%d", prefix, n-i)
		out[i] = searching.Release{Title: g, GUID: g, Published: at.Add(-time.Duration(i+1) * time.Second)}
	}
	return out
}

// leech is the Indexer the RSS tests fetch.
const leech = "TorrentLeech"

// listing has leech answer its recent call at each offset with the pages
// in order, offsets counted from their lengths.
func (p *planned) listing(pages ...[]searching.Release) {
	p.t.Helper()
	offset := 0
	for _, page := range pages {
		p.node.AnswerCall(p.ids[leech], indexerstest.Recent(offset), 0, searching.Answer{Releases: page})
		offset += len(page)
	}
}

func (p *planned) caps(name, caps string) {
	p.t.Helper()
	err := p.lib.Write(p.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return indexerstest.SetCaps(ctx, tx, p.ids[name], caps)
	})
	if err != nil {
		p.t.Fatal(err)
	}
}

// fetch is Recent on leech, written by Saw in its own Write.
func (p *planned) fetch() indexers.RSSFetch {
	p.t.Helper()
	f, err := p.x.Recent(p.t.Context(), p.ids[leech])
	if err != nil {
		p.t.Fatalf("Recent: %v", err)
	}
	var seen indexers.Seen
	err = p.lib.Write(p.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		seen, err = p.x.Saw(ctx, tx, f)
		return err
	})
	if err != nil {
		p.t.Fatal(err)
	}
	p.x.Seen(p.t.Context(), seen)
	return f
}

func guids(f indexers.RSSFetch) []string {
	out := make([]string, len(f.Releases))
	for i, r := range f.Releases {
		out[i] = r.GUID
	}
	return out
}

func (p *planned) feed() *indexers.FeedFact {
	p.t.Helper()
	v, err := p.x.Get(p.t.Context(), identity.Admin{}, p.ids[leech])
	if err != nil {
		p.t.Fatal(err)
	}
	return v.Feed
}

// gapItem is leech's rss-gap Health item, nil for none.
func (p *planned) gapItem() *health.Item {
	p.t.Helper()
	a, err := p.x.HealthSource().Answer(p.t.Context())
	if err != nil {
		p.t.Fatal(err)
	}
	for _, it := range a.Items {
		if it.Key == health.KeyOf("rss-gap", p.ids[leech].String()) {
			return &it
		}
	}
	return nil
}

func TestRecentPagesBackToTheStoredPositionAndTheReserveKeepsItsPages(t *testing.T) {
	p := plannedWith(t, "TorrentLeech")
	p.caps("TorrentLeech", pagingCaps)
	now := time.Now()
	old := uploads("old", 3, now.Add(-time.Hour))
	p.listing(old)
	if f := p.fetch(); !slices.Equal(guids(f), []string{"old-3", "old-2", "old-1"}) || f.Pages != 1 || f.Gap != indexers.GapNone {
		t.Fatalf("the first fetch = %v in %d pages, gap %q; want the newest page whole, no gap", guids(f), f.Pages, f.Gap)
	}

	newer := uploads("new", 6, now)
	p.listing(newer[:3], newer[3:], old)
	f := p.fetch()
	if want := []string{"new-6", "new-5", "new-4", "new-3", "new-2", "new-1"}; !slices.Equal(guids(f), want) || f.Pages != 3 || f.Gap != indexers.GapNone {
		t.Fatalf("the second fetch = %v in %d pages, gap %q; want %v in 3 pages back to the stored position", guids(f), f.Pages, f.Gap, want)
	}
	var pages int
	if err := p.lib.QueryRowContext(t.Context(), "SELECT last_pages FROM indexers_rss").Scan(&pages); err != nil || pages != 3 {
		t.Fatalf("last_pages = %d, %v; want 3", pages, err)
	}
	if got := p.rssChanged.all(); len(got) != 2 || got[1].IndexerID != p.ids["TorrentLeech"].String() {
		t.Fatalf("indexers.rss-changed = %v, want one per fetch", got)
	}

	p.rssEvery = 2 * time.Hour
	p.edit("TorrentLeech", indexers.Edit{Searches: &indexers.Limit{Count: 8, Per: indexers.Hour}})
	r := p.leech()
	if r.Background.Free != 0 || r.Press.Free != 3 {
		t.Fatalf("Room with 4 of 8 used = %+v, want no background line free beside the next fetch and its 3 pages", r)
	}
}

func TestAFetchThatRunsOutOfPagesRaisesAPageCapGap(t *testing.T) {
	p := plannedWith(t, "TorrentLeech")
	p.caps("TorrentLeech", pagingCaps)
	now := time.Now()
	p.listing(uploads("old", 1, now.Add(-time.Hour)))
	p.fetch()

	pages := make([][]searching.Release, 12)
	for i := range pages {
		pages[i] = []searching.Release{{Title: "busy", GUID: fmt.Sprintf("busy-%d", i), Published: now.Add(-time.Duration(i) * time.Minute)}}
	}
	p.listing(pages...)
	f := p.fetch()
	if f.Pages != indexers.DefaultTimings().RSSPages || f.Gap != indexers.GapPageCap || len(f.Releases) != 10 {
		t.Fatalf("a busy feed = %d Releases in %d pages, gap %q; want 10 in the 10-page cap and page_cap", len(f.Releases), f.Pages, f.Gap)
	}
	it := p.gapItem()
	want := "RSS sync may have missed releases on TorrentLeech. More releases were added to TorrentLeech between two RSS syncs than 10 pages list. Search from Wanted to find what was missed, or run RSS sync more often."
	if it == nil || it.Level != health.Warn || it.Text != want || it.Action.Label() != "Open Wanted" ||
		it.Action.Href() != "/library/wanted/searched?reason=missing%2Cfailed" {
		t.Fatalf("the gap's Health item = %+v, want a Warn saying %q linked to Wanted", it, want)
	}
}

func TestAFeedThatCantPageRaisesAGapOnlyWhenItMissesThePosition(t *testing.T) {
	p := plannedWith(t, "TorrentLeech")
	now := time.Now()
	old := uploads("old", 2, now.Add(-time.Hour))
	p.listing(old)
	p.fetch()

	p.listing(append(uploads("new", 2, now), old...))
	if f := p.fetch(); f.Gap != indexers.GapNone || len(f.Releases) != 2 {
		t.Fatalf("a page that reaches the position = %v, gap %q; want the 2 new ones, no gap", guids(f), f.Gap)
	}
	if p.gapItem() != nil {
		t.Fatal("a feed that reached its position raised a gap")
	}

	p.listing(uploads("busy", 3, now.Add(time.Minute)))
	f := p.fetch()
	if f.Gap != indexers.GapNoPaging || f.Pages != 1 || len(f.Releases) != 3 {
		t.Fatalf("a page past the position on a feed that can't page = %v in %d pages, gap %q; want 3 in 1 page, no_paging", guids(f), f.Pages, f.Gap)
	}
	for _, j := range p.node.Jobs() {
		if c, ok := j.Call.(searching.RecentCall); ok && c.Offset != 0 {
			t.Fatalf("a feed without RecentPaging was asked at offset %d", c.Offset)
		}
	}
	it := p.gapItem()
	if it == nil || !strings.Contains(it.Text, "TorrentLeech can't list older ones.") {
		t.Fatalf("the gap's Health item = %+v, want no_paging's words", it)
	}

	p.listing(uploads("busy", 3, now.Add(time.Minute)))
	p.fetch()
	if p.gapItem() != nil || p.feed().Gap != indexers.GapNone {
		t.Fatal("a fetch that reached the position left the gap")
	}
}

func TestALatePageStopsThePageBackAndALateFirstPageSkips(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := plannedWith(t, "TorrentLeech")
		p.caps("TorrentLeech", pagingCaps)
		now := time.Now()
		p.listing(uploads("old", 1, now.Add(-time.Hour)))
		p.fetch()

		first := uploads("new", 2, now)
		p.node.AnswerCall(p.ids["TorrentLeech"], indexerstest.Recent(0), 30*time.Second, searching.Answer{Releases: first})
		p.node.AnswerCall(p.ids["TorrentLeech"], indexerstest.Recent(2), 5*time.Minute, searching.Answer{Releases: uploads("older", 2, now.Add(-10*time.Minute))})
		began := time.Now()
		f := p.fetch()
		if f.Gap != indexers.GapLate || !slices.Equal(guids(f), []string{"new-2", "new-1"}) {
			t.Fatalf("a late second page = %v, gap %q; want the first page and late", guids(f), f.Gap)
		}
		if took := time.Since(began); took != indexers.SearchWait.Default {
			t.Fatalf("the page-back took %s, want one search wait, %s", took, indexers.SearchWait.Default)
		}

		p.node.AnswerCall(p.ids["TorrentLeech"], indexerstest.Recent(0), 5*time.Minute, searching.Answer{Releases: first})
		var skip *indexers.FeedSkipped
		if _, err := p.x.Recent(t.Context(), p.ids["TorrentLeech"]); !errors.As(err, &skip) || skip.Outcome.Kind != indexers.Late {
			t.Fatalf("a late first page = %v, want skipped late", err)
		}
		if want := "Skipped because TorrentLeech didn't respond in time. The next RSS sync tries it again."; skip.Sentence != want {
			t.Fatalf("the skip says %q, want %q", skip.Sentence, want)
		}
		synctest.Wait()
	})
}

func TestRecentClampsWhatTheSearchingWorkerSent(t *testing.T) {
	p := plannedWith(t, "TorrentLeech")
	now := time.Now()
	page := []searching.Release{
		{Title: "future", GUID: "future", Published: now.Add(48 * time.Hour)},
		{Title: "long", GUID: strings.Repeat("g", indexers.DefaultTimings().GUIDBytes+1), Published: now},
		{Title: "fine", GUID: "fine", Published: now.Add(-time.Minute)},
	}
	p.listing(page)
	f := p.fetch()
	if !slices.Equal(guids(f), []string{"future", "fine"}) {
		t.Fatalf("Releases = %v, want the long GUID dropped", guids(f))
	}
	if got := f.Releases[0].Published; got.After(time.Now()) {
		t.Fatalf("a future publish time reads %s, want at most its receipt", got)
	}

	oversized := uploads("flood", indexers.DefaultTimings().RSSPageItems+5, now.Add(time.Hour))
	p.listing(oversized)
	f = p.fetch()
	if len(f.Releases) != indexers.DefaultTimings().RSSPageItems || f.Gap != indexers.GapPageCap {
		t.Fatalf("an oversized page = %d Releases, gap %q; want it cut to %d and page_cap", len(f.Releases), f.Gap, indexers.DefaultTimings().RSSPageItems)
	}
}

func TestAFetchAtTheSearchLimitIsSkippedWithWhenItFrees(t *testing.T) {
	p := plannedWith(t, "TorrentLeech")
	p.edit("TorrentLeech", indexers.Edit{Searches: &indexers.Limit{Count: 1, Per: indexers.Hour}})
	p.fetch()
	var skip *indexers.FeedSkipped
	if _, err := p.x.Recent(t.Context(), p.ids["TorrentLeech"]); !errors.As(err, &skip) || skip.Outcome.Kind != indexers.LimitReached {
		t.Fatalf("a fetch at the search limit = %v, want skipped limit_reached", err)
	}
	if !strings.HasPrefix(skip.Sentence, "Skipped because TorrentLeech is at its search limit until ") {
		t.Fatalf("the skip says %q", skip.Sentence)
	}
}

func TestSawRollsBackWithItsWrite(t *testing.T) {
	p := plannedWith(t, "TorrentLeech")
	p.listing(uploads("new", 2, time.Now()))
	f, err := p.x.Recent(t.Context(), p.ids["TorrentLeech"])
	if err != nil {
		t.Fatal(err)
	}
	refused := errors.New("the deciding Write failed")
	err = p.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := p.x.Saw(ctx, tx, f); err != nil {
			return err
		}
		return refused
	})
	if !errors.Is(err, refused) || p.feed() != nil {
		t.Fatalf("Saw in a Write that failed = %v, feed %+v; want no position stored", err, p.feed())
	}
}

func TestRSSUnitsArePlacedAsTheHealthCheckAndOnlyWithRSSOn(t *testing.T) {
	p := plannedWith(t, "Alder", "Birch", "Cedar")
	p.edit("Birch", indexers.Edit{RSS: new(false)})
	p.edit("Cedar", indexers.Edit{On: new(false)})
	var checked, rss []string
	err := p.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		units, err := p.x.Tasks()[1].Units(ctx, r)
		if err != nil {
			return err
		}
		for _, u := range units {
			checked = append(checked, fmt.Sprintf("%s %v", u.Label, u.Place))
		}
		units, err = p.x.RSSUnits(ctx, r)
		for _, u := range units {
			rss = append(rss, fmt.Sprintf("%s %v", u.Label, u.Place))
			if indexers.IDOf(u.Key) != p.ids[u.Label] {
				t.Errorf("IDOf(%s) isn't %s's id", u.Key, u.Label)
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rss) != 1 || !slices.Contains(checked, rss[0]) {
		t.Fatalf("RSS units = %v, want Alder alone placed as its health check unit, one of %v", rss, checked)
	}
}
