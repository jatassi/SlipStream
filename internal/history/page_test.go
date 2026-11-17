package history_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

func notedAs(kind entry.Kind, group entry.Group, tally works.Noun) entry.Row[noted] {
	return entry.Row[noted]{
		Spec:   entry.Spec{Kind: kind, Label: string(kind), Group: group, BellByDefault: true, Tally: tally},
		Render: func(f noted, _ entry.Context) entry.Message { return entry.Message{Title: f.Name} },
	}
}

var (
	fetchedRow = notedAs("fetched", entry.Grabs, works.Noun{One: "Grab", Many: "Grabs"})
	refusedRow = notedAs("refused", entry.Grabs, works.Noun{})
	landedRow  = notedAs("landed", entry.Imports, works.Noun{One: "Import", Many: "Imports"})
	leftRow    = notedAs("left", entry.Library, works.Noun{})
)

// logged is a History whose owner "noter" records Fetched and Refused
// (Grabs), Landed (Imports) and Left (Library), each titled by its name.
type logged struct {
	*world
	rec entry.Recorder
	t   *testing.T
	// kept is every entry recorded, oldest first, as a test reads it.
	kept []kept
}

type kept struct {
	name  string
	group entry.Group
	kind  entry.Kind
	about entry.About
	at    time.Time
}

func logging(t *testing.T, f func(t *testing.T, l *logged)) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, notificationstest.Schemas()...)
		w := notificationstest.Over(t, lib, filepath.Join(t.TempDir(), secret.KeyFile), notifications.DefaultTimings(),
			notificationstest.Owner{Name: "noter", Rows: []entry.Entry{fetchedRow, refusedRow, landedRow, leftRow}})
		f(t, &logged{world: w, rec: w.Recorder("noter"), t: t})
	})
}

func (l *logged) record(row entry.Row[noted], name string, about entry.About) {
	l.t.Helper()
	err := l.Lib.Write(l.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return row.Record(ctx, tx, l.rec, noted{Name: name, On: about})
	})
	if err != nil {
		l.t.Fatal(err)
	}
	l.kept = append(l.kept, kept{name: name, group: row.Group, kind: row.Kind, about: about, at: time.Now()})
}

// raise records a New Health item entry named name.
func (l *logged) raise(name string) {
	l.t.Helper()
	l.Transitions(health.Transition{
		Category: health.LibraryID, Key: health.Key(name), Kind: health.Raised, To: health.Warn,
		Subject: name, Text: "It went wrong.", At: time.Now(), Since: time.Now(),
	})
	l.kept = append(l.kept, kept{name: "New Health item for " + name, group: entry.Health, kind: history.NewItemsRow.Kind, at: time.Now()})
}

func (l *logged) page(f history.Filter, b history.Before, limit int) history.Page {
	l.t.Helper()
	p, err := l.History.Page(l.t.Context(), identity.Admin{}, f, b, limit)
	if err != nil {
		l.t.Fatalf("listing %+v: %v", f, err)
	}
	return p
}

// all is every page of f, by before, as names.
func (l *logged) all(f history.Filter, limit int, between func()) []string {
	l.t.Helper()
	var out []string
	for b := history.Before(0); ; {
		p := l.page(f, b, limit)
		for _, e := range p.Entries {
			out = append(out, e.Message.Title)
		}
		if p.Next == 0 {
			return out
		}
		b = p.Next
		if between != nil {
			between()
		}
	}
}

// want is the names f lists of what was recorded, newest first.
func (l *logged) want(f history.Filter) []string {
	var out []string
	for _, k := range slices.Backward(l.kept) {
		if keeps(f, k) {
			out = append(out, k.name)
		}
	}
	return out
}

func keeps(f history.Filter, k kept) bool {
	switch {
	case f.Type != "" && k.about.Type != f.Type,
		!f.Work.IsZero() && !slices.Contains(k.about.Works, f.Work),
		!f.Since.IsZero() && k.at.Before(f.Since),
		f.Groups != nil && !slices.Contains(f.Groups, k.group):
		return false
	}
	named := false
	for _, kind := range f.Kinds {
		named = named || groupOf(kind) == k.group
	}
	return !named || slices.Contains(f.Kinds, k.kind)
}

func groupOf(k entry.Kind) entry.Group {
	for _, row := range []entry.Row[noted]{fetchedRow, refusedRow, landedRow, leftRow} {
		if row.Kind == k {
			return row.Group
		}
	}
	return entry.Health
}

// seeded records 250 entries over 10 days: every kind, two types, three
// Works, and a Health entry every 25.
func seeded(l *logged) []works.WorkID {
	ws := workIDs(3)
	rows := []entry.Row[noted]{fetchedRow, refusedRow, landedRow, leftRow}
	for i := range 250 {
		if i%25 == 24 {
			l.raise(fmt.Sprintf("health %d", i))
			time.Sleep(24 * time.Hour)
			continue
		}
		t := []works.TypeID{"films", "shows"}[i%2]
		l.record(rows[i%4], fmt.Sprintf("entry %d", i), entry.About{Type: t, Works: []works.WorkID{ws[i%3]}})
		time.Sleep(time.Minute)
	}
	return ws
}

func TestEveryPageByBeforeJoinsToTheWholeListWhileNewEntriesArrive(t *testing.T) {
	logging(t, func(t *testing.T, l *logged) {
		seeded(l)
		want := l.want(history.Filter{})
		n := 0
		got := l.all(history.Filter{}, 30, func() {
			n++
			l.record(leftRow, fmt.Sprintf("late %d", n), entry.About{})
		})
		if !slices.Equal(got, want) {
			t.Fatalf("the pages join to %d entries %q, want the %d recorded before the first page with no gap or repeat", len(got), got, len(want))
		}
	})
}

func TestEachFilterListsExactlyItsEntries(t *testing.T) {
	logging(t, func(t *testing.T, l *logged) {
		ws := seeded(l)
		since := l.kept[100].at
		for _, f := range []history.Filter{
			{},
			{Type: "films"},
			{Groups: []entry.Group{entry.Grabs}},
			{Groups: []entry.Group{entry.Health}},
			{Groups: []entry.Group{entry.Grabs, entry.Imports}, Kinds: []entry.Kind{refusedRow.Kind}},
			{Kinds: []entry.Kind{landedRow.Kind}},
			{Since: since},
			{Work: ws[1]},
			{Work: ws[1], Type: "shows", Kinds: []entry.Kind{fetchedRow.Kind, leftRow.Kind}},
			{Type: "shows", Groups: []entry.Group{entry.Imports, entry.Health}, Since: since},
			{Type: "films", Kinds: []entry.Kind{refusedRow.Kind}, Since: since},
		} {
			want := l.want(f)
			if got := l.all(f, 17, nil); !slices.Equal(got, want) {
				t.Errorf("%+v lists %d entries, want %d: %q", f, len(got), len(want), want)
			}
		}
	})
}

func TestAKindOutsideTheGroupsAskedForAndAnUnknownKindAreRefused(t *testing.T) {
	logging(t, func(t *testing.T, l *logged) {
		_, err := l.History.Page(t.Context(), identity.Admin{}, history.Filter{Groups: []entry.Group{entry.Imports}, Kinds: []entry.Kind{fetchedRow.Kind}}, 0, 10)
		if off, ok := errors.AsType[*history.KindOffGroupError](err); !ok || off.Kind.Kind != fetchedRow.Kind {
			t.Errorf("a Grabs kind under Imports alone gave %v, want its group refused", err)
		}
		_, err = l.History.Page(t.Context(), identity.Admin{}, history.Filter{Kinds: []entry.Kind{"nothing"}}, 0, 10)
		if unknown, ok := errors.AsType[*history.UnknownKindError](err); !ok || unknown.Kind != "nothing" {
			t.Errorf("an unknown kind gave %v, want it refused", err)
		}
	})
}
