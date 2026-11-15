package grabs_test

import (
	"context"
	stdsql "database/sql"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func searchOf(t *testing.T, w *grabstest.World) grabs.WorkSearch {
	t.Helper()
	synctest.Wait()
	s, err := w.Grabs.Searching(t.Context(), w.Work)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func wantedPage(t *testing.T, w *grabstest.World, q grabs.WantedQuery) grabs.WantedPage {
	t.Helper()
	synctest.Wait()
	if q.Sort == "" {
		q.Sort = grabs.ByNext
	}
	if q.Limit == 0 {
		q.Limit = 100
	}
	p, err := w.Grabs.Wanted(t.Context(), identity.Admin{}, q)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func rowOf(t *testing.T, p grabs.WantedPage, work works.WorkID) grabs.WantedRow {
	t.Helper()
	i := slices.IndexFunc(p.Rows, func(r grabs.WantedRow) bool { return r.Work.ID == work })
	if i < 0 {
		t.Fatalf("Wanted lists %+v, want a row of %s", p.Rows, work)
	}
	return p.Rows[i]
}

func TestTheHerosLeadAndWantedsRowAgreeForTheSameWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		w.Answer("Quick Site", time.Minute, searching.Answer{})
		agree := func(step string, phase grabs.Phase, last grabs.Outcome) {
			t.Helper()
			lead, row := searchOf(t, w).Lead(), rowOf(t, wantedPage(t, w, grabs.WantedQuery{}), w.Work)
			var led grabs.GroupState
			if lead.Group != nil {
				led = *lead.Group
			}
			if led.Phase != phase || row.Phase != phase {
				t.Fatalf("%s: the hero leads with %s and Wanted's row reads %s, want both %s", step, led.Phase, row.Phase, phase)
			}
			if !reflect.DeepEqual(led.Last, row.Last) || lead.Group != nil && !led.Next.Equal(row.Next) {
				t.Fatalf("%s: the hero's group last ran %+v, next %v, and Wanted's row %+v, next %v, want them equal", step, led.Last, led.Next, row.Last, row.Next)
			}
			if (last == "") != (row.Last == nil) || row.Last != nil && row.Last.Outcome != last {
				t.Fatalf("%s: Wanted's row last ran %+v, want %q", step, row.Last, last)
			}
		}
		agree("before any search", grabs.Idle, "")
		press(t, w, grabs.Scope{})
		agree("pressed", grabs.InQueue, "")
		w.Run()
		agree("its run asks Quick Site", grabs.Searching, "")
		time.Sleep(2 * time.Minute)
		agree("its run found nothing", grabs.Idle, grabs.NothingFound)
		if lead := searchOf(t, w).Lead(); lead.Group.Next.IsZero() || lead.Group.Label != "Winter" {
			t.Fatalf("after the run the hero leads with %+v, want Winter with its next Backlog search", lead.Group)
		}
	})
}

func TestANeverSearchedRowIsNextSearchedByTheNextBacklogSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		page := wantedPage(t, w, grabs.WantedQuery{})
		if row := rowOf(t, page, w.Work); row.Phase != grabs.Idle || page.Status.NextBacklog.IsZero() || !row.Next.Equal(page.Status.NextBacklog) {
			t.Fatalf("a never-searched row reads %s, next %v, with Backlog search next at %v, want idle and next at the Backlog search", row.Phase, row.Next, page.Status.NextBacklog)
		}
		settings(t, w, map[string]string{grabs.BacklogOn.Key: "false"})
		if row := rowOf(t, wantedPage(t, w, grabs.WantedQuery{}), w.Work); !row.Next.IsZero() {
			t.Fatalf("with Backlog search off a never-searched row is next searched at %v, want only on a press", row.Next)
		}
	})
}

func TestWantedsItemsReadAsTheWorkPagesCodes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 3, Coded: true})
		got := rowOf(t, wantedPage(t, w, grabs.WantedQuery{}), w.Work).Ranges
		if want := []grabs.ItemRange{{First: "B1E01", Last: "B1E03"}}; !slices.Equal(got, want) {
			t.Fatalf("Wanted's row lists %+v, want %+v", got, want)
		}
	})
}

func TestWantedsCursorStaysStableAcrossAQueueChange(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		for range 3 {
			w.AddWork()
		}
		all := wantedPage(t, w, grabs.WantedQuery{})
		first := wantedPage(t, w, grabs.WantedQuery{Limit: 2})
		if all.Total != 4 || first.Total != 4 || len(first.Rows) != 2 || first.Next == "" {
			t.Fatalf("Wanted lists %d Works, and a first page of 2 holds %d rows of %d with cursor %q, want 4 Works and a cursor", all.Total, len(first.Rows), first.Total, first.Next)
		}
		if _, err := w.Grabs.SearchWork(t.Context(), identity.Admin{}, first.Rows[0].Work.ID, grabs.Scope{}); err != nil {
			t.Fatal(err)
		}
		second := wantedPage(t, w, grabs.WantedQuery{Limit: 2, Cursor: first.Next})
		got := []works.WorkID{second.Rows[0].Work.ID, second.Rows[1].Work.ID}
		want := []works.WorkID{all.Rows[2].Work.ID, all.Rows[3].Work.ID}
		if !slices.Equal(got, want) || second.Next != "" {
			t.Fatalf("after a press the second page lists %v with cursor %q, want %v and no cursor", got, second.Next, want)
		}
		_, err := w.Grabs.Wanted(t.Context(), identity.Admin{}, grabs.WantedQuery{Sort: grabs.ByTitle, Limit: 2, Cursor: first.Next})
		if !errors.Is(err, grabs.ErrCursorStale) {
			t.Fatalf("a cursor of another sort answers %v, want ErrCursorStale", err)
		}
	})
}

func TestANotFoundRowLeavesTheLibrarysCountsAlone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		w.Answer("Quick Site", time.Minute, searching.Answer{})
		rollup := func() files.Rollup {
			t.Helper()
			synctest.Wait()
			_, _, r, err := w.Config.Files.WorkCells(t.Context(), identity.Admin{}, w.Work)
			if err != nil {
				t.Fatal(err)
			}
			return r
		}
		flagged := func(r files.Rollup) int {
			n := 0
			for _, need := range r.Needs {
				if need.Flags() {
					n++
				}
			}
			return n
		}
		before := rollup()
		press(t, w, grabs.Scope{})
		w.Run()
		time.Sleep(2 * time.Minute)
		after := rollup()
		if after.Status != before.Status || after.Counted != before.Counted || flagged(after) != flagged(before) || after.Reason != before.Reason {
			t.Fatalf("after a search found nothing the Work reads %v %+v with %d flagged needs and reason %v, want %v %+v, %d and %v as before",
				after.Status, after.Counted, flagged(after), after.Reason, before.Status, before.Counted, flagged(before), before.Reason)
		}
		i := slices.IndexFunc(after.Needs, func(n files.Need) bool { return n.Kind == files.NeedNotFound })
		if i < 0 || after.Needs[i].SearchedAt.IsZero() || after.Needs[i].NextAt.IsZero() {
			t.Fatalf("the Work's needs are %+v, want a not found row with its last and next search", after.Needs)
		}
		if row := rowOf(t, wantedPage(t, w, grabs.WantedQuery{}), w.Work); row.Reason != files.ReasonMissing || row.Last.Outcome != grabs.NothingFound {
			t.Fatalf("Wanted's row is %+v, want it missing with its search finding nothing", row)
		}
	})
}

func TestAStarvedIndexerAppearsInTheStatusSentence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"TorrentLeech"}, RSSEvery: 15 * time.Minute})
		w.Limit("TorrentLeech", 1, indexers.Hour)
		w.Run()
		enqueue(t, w, searches.Backlog)
		want := grabs.Limit{Indexer: w.Sites["TorrentLeech"], Name: "TorrentLeech", Starved: true}
		if got := wantedPage(t, w, grabs.WantedQuery{}).Status.Limits; !slices.Equal(got, []grabs.Limit{want}) {
			t.Fatalf("the status sentence's limits are %+v, want %+v", got, want)
		}
		lead := searchOf(t, w).Lead()
		if b := lead.Group.Behind; lead.Group.Phase != grabs.Searching || b == nil || !b.Starved || b.Indexer != want.Indexer || b.Name != "TorrentLeech" {
			t.Fatalf("the hero leads with %+v behind %+v, want its search waiting on TorrentLeech, starved", lead.Group, b)
		}
	})
}

func TestTheQueueReadsDontGrowWithTheLibrary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		w.Answer("Quick Site", time.Second, searching.Answer{Releases: []searching.Release{{
			Title: "The.Attic.2020.1080p.BluRay.x264", Size: 8 << 30, Seeders: 10, Link: searching.LinkOf("https://site.test/attic.torrent"),
		}}})
		w.Run()
		var unbound []string
		counts := func() []int {
			t.Helper()
			only := works.Only([]works.WorkID{w.Work})
			cells := grabs.GrabCells{InStep: func(role.NodeID) bool { return true }}
			return []int{
				queriesOf(t, w, &unbound, func(ctx context.Context, r storage.Reader[storage.Library]) error {
					_, err := cells.GrabCells(ctx, r, only)
					return err
				}),
				queriesOf(t, w, &unbound, func(ctx context.Context, r storage.Reader[storage.Library]) error {
					_, err := cells.Searching(ctx, r, only)
					return err
				}),
			}
		}
		search := func(work works.WorkID) {
			t.Helper()
			if _, err := w.Grabs.SearchWork(t.Context(), identity.Admin{}, work, grabs.Scope{}); err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Minute)
		}
		search(w.Work)
		one := counts()
		for range 3 {
			search(w.AddWork())
		}
		if grabbed, _ := grabbed(t, w); len(grabbed) == 0 {
			t.Fatal("no search grabbed, want each Work's search to grab its Release")
		}
		unbound = nil
		if many := counts(); !slices.Equal(many, one) {
			t.Fatalf("with 4 searched Works the cells' reads of one Work ask %v queries, want %v as with one", many, one)
		}
		for _, q := range unbound {
			if strings.Contains(q, "FROM grabs_") && !strings.Contains(q, "FROM grabs_settings") {
				t.Errorf("a read of one Work's cells asked %q, which binds no Work and reads a whole grabs table", q)
			}
		}
	})
}

// queriesOf is how many queries read asks of w's library, adding those that
// bind no argument to unbound.
func queriesOf(t *testing.T, w *grabstest.World, unbound *[]string, read func(context.Context, storage.Reader[storage.Library]) error) int {
	t.Helper()
	synctest.Wait()
	n := 0
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return read(ctx, counting{Reader: r, n: &n, unbound: unbound})
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// counting is a reader that counts the queries asked through it and
// keeps those that bind no argument.
type counting struct {
	storage.Reader[storage.Library]
	n       *int
	unbound *[]string
}

func (c counting) asked(query string, args []any) {
	*c.n++
	if len(args) == 0 {
		*c.unbound = append(*c.unbound, query)
	}
}

func (c counting) QueryContext(ctx context.Context, query string, args ...any) (*stdsql.Rows, error) {
	c.asked(query, args)
	return c.Reader.QueryContext(ctx, query, args...)
}

func (c counting) QueryRowContext(ctx context.Context, query string, args ...any) *stdsql.Row {
	c.asked(query, args)
	return c.Reader.QueryRowContext(ctx, query, args...)
}
