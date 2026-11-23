package library_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/history/historytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestOnlyAnAddThatAnnouncesRecordsWorksAdded(t *testing.T) {
	s := newShelf(t)
	l := s.library(nil)
	s.stock("w1")
	s.stock("w2")
	if _, err := l.Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: "w1", Root: &s.root.ID}); err != nil {
		t.Fatal(err)
	}
	if got := entrytest.Of(s.notices, library.AddedRow); len(got) != 0 {
		t.Fatalf("an add that doesn't announce recorded Works added %+v", got)
	}
	added, err := l.Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: "w2", Root: &s.root.ID, Announce: true})
	if err != nil {
		t.Fatal(err)
	}
	got := entrytest.Of(s.notices, library.AddedRow)
	if len(got) != 1 || got[0].Work.ID != added.Tree.ID || got[0].Work.Label != "Shelf w2" || !got[0].By.APIKey || got[0].Root == "" {
		t.Fatalf("the add flow's add recorded %+v, want one Works added for Shelf w2 by the API key into its root", got)
	}
	if about := entrytest.Abouts(s.notices, library.AddedRow)[0]; about.Type != contenttypetest.ID || !slices.Equal(about.Works, []works.WorkID{added.Tree.ID}) {
		t.Errorf("Works added is about %+v, want Shelf w2", about)
	}
}

func TestDeletingOneWorkRecordsItWhole(t *testing.T) {
	s := newShelf(t)
	l := s.library(nil)
	s.stock("w1")
	added, err := l.Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: "w1", Root: &s.root.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := l.Delete(t.Context(), identity.Admin{}, []works.WorkID{added.Tree.ID}, library.Deleting{Files: library.LeaveFiles}); err != nil {
		t.Fatal(err)
	}
	got := entrytest.Of(s.notices, library.DeletedRow)
	if len(got) != 1 || len(got[0].Types) != 1 || got[0].Types[0].Count != 1 || got[0].Types[0].Work == nil || got[0].Types[0].Work.ID != added.Tree.ID {
		t.Fatalf("a one-Work delete recorded %+v, want one Works deleted carrying the Work", got)
	}
}

// deleteBudget is the design's: a 20,000-Work delete's Write within 500 ms.
const deleteBudget = 500 * time.Millisecond

// quietOutbox owes nothing and was told nothing.
type quietOutbox struct{}

func (quietOutbox) Queue(context.Context, *storage.Tx[storage.Library], history.Captured) error {
	return nil
}

func (quietOutbox) Owed(context.Context, storage.Reader[storage.Library], []history.EventID) (map[history.EventID]bool, error) {
	return nil, nil
}

func (quietOutbox) Told(context.Context, storage.Reader[storage.Library], string) (history.Told, bool, error) {
	return history.Told{}, false, nil
}

func TestA20000WorkDeleteAcrossTwoContentTypesRecordsOneEntryPerContentType(t *testing.T) {
	s := newShelf(t)
	h := history.New(history.Config{Library: s.lib, Catalog: entry.NewCatalog(), Outbox: quietOutbox{}, Log: slog.New(slog.DiscardHandler)})
	s.rec = h.Recorder("library", library.Entries())
	l := s.library(nil)
	insert := func(contentType string, n int) {
		err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `
				WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?)
				INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at)
				SELECT randomblob(16), ?, 'k' || i, ? || ' ' || printf('%05d', i), printf('%05d', i), 1, 1 FROM n`, n, contentType, contentType)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	insert(string(contenttypetest.ID), 12000)
	insert("other", 8000)
	var ids []works.WorkID
	rows, err := s.lib.QueryContext(t.Context(), `SELECT id FROM works_work`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id works.WorkID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, _, err := l.Delete(t.Context(), identity.Admin{}, ids, library.Deleting{Files: library.LeaveFiles}); err != nil {
		t.Fatal(err)
	}
	t.Logf("the delete took %s, against a %s budget", time.Since(start).Round(time.Millisecond), deleteBudget)
	var tally []string
	for _, e := range historytest.Entries(t, s.lib) {
		var f library.DeletedFacts
		if err := json.Unmarshal([]byte(e.Subject), &f); err != nil {
			t.Fatal(err)
		}
		ty := f.Types[0]
		tally = append(tally, fmt.Sprintf("%s %d %s %d %d %v", e.Kind, len(f.Types), e.Type, len(e.Works), ty.Count, ty.First))
	}
	want := []string{
		"works_deleted 1 fixture 12000 12000 [fixture 00001 fixture 00002 fixture 00003]",
		"works_deleted 1 other 8000 8000 [other 00001 other 00002 other 00003]",
	}
	if !reflect.DeepEqual(tally, want) {
		t.Fatalf("History holds %q, want %q", tally, want)
	}
}

func TestLibraryMessagesReadAsWritten(t *testing.T) {
	books := entry.Nouns{Work: works.Noun{One: "Book", Many: "Books"}, Item: works.Noun{One: "Book", Many: "Books"}, Single: true}
	zines := entry.Nouns{Work: works.Noun{One: "Zine", Many: "Zines"}, Levels: []works.Noun{{One: "Volume", Many: "Volumes"}}, Item: works.Noun{One: "Issue", Many: "Issues"}}
	dune := entry.Work{
		ID: works.WorkID{UUID: must(storage.ParseUUID("019a8f5e-0000-7000-8000-000000000002"))}, Label: "Dune (1965)", Type: "book", Nouns: books,
		Poster: artwork.Ref{Source: "tmdb", Shape: artwork.ShapePoster, Path: "/dune.jpg"},
	}
	page := "/works/" + dune.ID.String()
	stored := func(e entry.Entry, facts any, name string, names ...string) {
		entrytest.Stored(t, e, facts, "testdata/notices", name, names...)
	}
	stored(library.AddedRow, library.AddedFacts{Work: dune, Root: "Books library root on Den", By: entry.Actor{Name: "Sam"}}, "works-added",
		"Dune (1965)", "Books library root on Den", "Sam", page)
	stored(library.AddedRow, library.AddedFacts{Work: dune, By: entry.Actor{APIKey: true}}, "works-added-no-root", "Dune (1965)", "API key", page)
	stored(library.DeletedRow, library.DeletedFacts{By: entry.Actor{Name: "Sam"}, Types: []library.DeletedType{{Nouns: books, Count: 1, First: []string{dune.Label}, Work: &dune}}}, "works-deleted-one",
		"Dune (1965)", "Sam", "/library")
	stored(library.DeletedRow, library.DeletedFacts{By: entry.Actor{Name: "Sam"}, Types: []library.DeletedType{{Nouns: books, Count: 12, First: []string{"Dune (1965)", "Emma (1815)", "Ulysses (1922)"}}}}, "works-deleted-many",
		"12 books", "Dune (1965)", "Emma (1815)", "Ulysses (1922)", "9 more", "/library")
	stored(library.DeletedRow, library.DeletedFacts{By: entry.Actor{APIKey: true}, Types: []library.DeletedType{
		{Nouns: books, Count: 12000, First: []string{"Dune (1965)", "Emma (1815)", "Ulysses (1922)"}},
		{Nouns: zines, Count: 8000, First: []string{"Ark", "Bolt", "Crux"}},
	}}, "works-deleted-two-types", "12,000 books", "8,000 zines", "Dune (1965)", "Ark", "11,997 more", "7,997 more", "API key", "/library")
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
