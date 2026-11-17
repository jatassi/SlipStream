package history_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/historytest"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// noted is an owner's facts that are about whatever On says, as each
// owner's About answers from its own facts.
type noted struct {
	Name string      `json:"name"`
	On   entry.About `json:"-"`
}

func (f noted) About() entry.About { return f.On }

var notedRow = entry.Row[noted]{
	Spec:   entry.Spec{Kind: "noted", Label: "Noted", Group: entry.Library, BellByDefault: true},
	Render: func(f noted, _ entry.Context) entry.Message { return entry.Message{Title: "Noted " + f.Name} },
}

type capturing struct {
	*world
	rec entry.Recorder
}

func capture(t *testing.T, f func(t *testing.T, c capturing)) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, notificationstest.Schemas()...)
		w := notificationstest.Over(t, lib, filepath.Join(t.TempDir(), secret.KeyFile), notifications.DefaultTimings(),
			notificationstest.Owner{Name: "noter", Rows: []entry.Entry{notedRow}})
		f(t, capturing{world: w, rec: w.Recorder("noter")})
	})
}

// record records fs in one Write, as one owner's Write that makes them
// true.
func (c capturing) record(t *testing.T, fs ...noted) {
	t.Helper()
	err := c.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, f := range fs {
			if err := notedRow.Record(ctx, tx, c.rec, f); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
}

func workIDs(n int) []works.WorkID {
	out := make([]works.WorkID, n)
	for i := range out {
		out[i] = works.WorkID{UUID: storage.NewUUID()}
	}
	return out
}

func sameWorks(got, want []works.WorkID) bool {
	return len(got) == len(want) && !slices.ContainsFunc(want, func(w works.WorkID) bool { return !slices.Contains(got, w) })
}

func titles(f notifications.Feed) []string {
	var out []string
	for _, r := range f.Rows {
		out = append(out, r.Title)
	}
	return out
}

func TestAnEntryKeepsTheGrabImportAndWorkItIsAboutAndReachesTheBell(t *testing.T) {
	capture(t, func(t *testing.T, c capturing) {
		admin := c.Admin("sam@example.test")
		one := workIDs(1)
		grab, imp := storage.NewUUID(), storage.NewUUID()
		c.record(t, noted{Name: "grab", On: entry.About{Type: contenttypetest.ID, Works: one, Grab: grab, Import: imp}})
		got := historytest.Entries(t, c.Lib)
		if len(got) != 1 {
			t.Fatalf("History holds %+v, want one entry", got)
		}
		e := got[0]
		if e.Kind != "noted" || e.Type != contenttypetest.ID || e.Work != one[0] || !sameWorks(e.Works, one) || e.Grab != grab || e.Import != imp {
			t.Errorf("the entry is %+v, want it about Grab %s, Import %s and Work %s of the fixture type", e, grab, imp, one[0])
		}
		if f := titles(c.Feed(admin)); !slices.Equal(f, []string{"Noted grab"}) {
			t.Errorf("the bell shows %q, want the entry", f)
		}
	})
}

func TestAQuietEntryIsInHistoryAndNotOnTheBell(t *testing.T) {
	capture(t, func(t *testing.T, c capturing) {
		admin := c.Admin("sam@example.test")
		c.record(t, noted{Name: "quiet", On: entry.About{Type: contenttypetest.ID, Works: workIDs(1), Grab: storage.NewUUID(), Quiet: true}})
		if got := historytest.Entries(t, c.Lib); len(got) != 1 || got[0].Grab.IsZero() {
			t.Fatalf("History holds %+v, want the quiet entry with its Grab", got)
		}
		if f := titles(c.Feed(admin)); len(f) != 0 {
			t.Errorf("the bell shows %q for a quiet entry", f)
		}
	})
}

func TestTwoEntriesInOneWriteEachKeepEveryWorkAndEachReachTheBell(t *testing.T) {
	capture(t, func(t *testing.T, c capturing) {
		admin := c.Admin("sam@example.test")
		first, second := workIDs(3), workIDs(2)
		c.record(t,
			noted{Name: "first", On: entry.About{Type: contenttypetest.ID, Works: first}},
			noted{Name: "second", On: entry.About{Type: "other", Works: second}},
		)
		got := historytest.Entries(t, c.Lib)
		if len(got) != 2 || got[0].Type != contenttypetest.ID || got[1].Type != "other" {
			t.Fatalf("History holds %+v, want one entry per type", got)
		}
		for i, want := range [][]works.WorkID{first, second} {
			if !got[i].Work.IsZero() || !sameWorks(got[i].Works, want) {
				t.Errorf("entry %d is about Work %s and Works %v, want no one Work and every one of %v", i, got[i].Work, got[i].Works, want)
			}
		}
		if f := titles(c.Feed(admin)); !slices.Equal(f, []string{"Noted second", "Noted first"}) {
			t.Errorf("the bell shows %q, want both entries, newest first", f)
		}
	})
}
