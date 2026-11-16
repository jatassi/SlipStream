package library_test

import (
	"context"
	"flag"
	"fmt"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/notice/noticetest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var update = flag.Bool("update", false, "rewrite the rendered messages under testdata/notices")

func TestOnlyAnAddThatAnnouncesRecordsWorksAdded(t *testing.T) {
	s := newShelf(t)
	l := s.library(nil)
	s.stock("w1")
	s.stock("w2")
	if _, err := l.Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: "w1", Root: &s.root.ID}); err != nil {
		t.Fatal(err)
	}
	if got := noticetest.Of(s.notices, library.AddedRow); len(got) != 0 {
		t.Fatalf("an add that doesn't announce recorded Works added %+v", got)
	}
	added, err := l.Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: "w2", Root: &s.root.ID, Announce: true})
	if err != nil {
		t.Fatal(err)
	}
	got := noticetest.Of(s.notices, library.AddedRow)
	if len(got) != 1 || got[0].Work.ID != added.Tree.ID || got[0].Work.Label != "Shelf w2" || !got[0].By.APIKey || got[0].Root == "" {
		t.Fatalf("the add flow's add recorded %+v, want one Works added for Shelf w2 by the API key into its root", got)
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
	got := noticetest.Of(s.notices, library.DeletedRow)
	if len(got) != 1 || len(got[0].Types) != 1 || got[0].Types[0].Count != 1 || got[0].Types[0].Work == nil || got[0].Types[0].Work.ID != added.Tree.ID {
		t.Fatalf("a one-Work delete recorded %+v, want one Works deleted carrying the Work", got)
	}
}

func TestA20000WorkDeleteAcrossTwoContentTypesRecordsOneEvent(t *testing.T) {
	s := newShelf(t)
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
	if _, _, err := l.Delete(t.Context(), identity.Admin{}, ids, library.Deleting{Files: library.LeaveFiles}); err != nil {
		t.Fatal(err)
	}
	got := noticetest.Of(s.notices, library.DeletedRow)
	if len(got) != 1 {
		t.Fatalf("the delete recorded %d Works deleted, want 1", len(got))
	}
	var tally []string
	for _, ty := range got[0].Types {
		tally = append(tally, fmt.Sprint(ty.Count, ty.First, ty.Work == nil))
	}
	want := []string{"12000 [fixture 00001 fixture 00002 fixture 00003] true", "8000 [other 00001 other 00002 other 00003] true"}
	if !reflect.DeepEqual(tally, want) {
		t.Fatalf("Works deleted tells %q, want %q", tally, want)
	}
}

func TestLibraryMessagesReadAsWritten(t *testing.T) {
	books := notice.Nouns{Work: works.Noun{One: "Book", Many: "Books"}, Item: works.Noun{One: "Book", Many: "Books"}, Single: true}
	zines := notice.Nouns{Work: works.Noun{One: "Zine", Many: "Zines"}, Levels: []works.Noun{{One: "Volume", Many: "Volumes"}}, Item: works.Noun{One: "Issue", Many: "Issues"}}
	dune := notice.Work{
		ID: works.WorkID{UUID: must(storage.ParseUUID("019a8f5e-0000-7000-8000-000000000002"))}, Label: "Dune (1965)", Type: "book", Nouns: books,
		Poster: artwork.Ref{Source: "tmdb", Shape: artwork.ShapePoster, Path: "/dune.jpg"},
	}
	golden := func(e notice.Entry, facts any, name string) {
		noticetest.Golden(t, e, facts, "testdata/notices", name, *update)
	}
	golden(library.AddedRow, library.AddedFacts{Work: dune, Root: "Books library root on Den", By: notice.Actor{Name: "Sam"}}, "works-added")
	golden(library.AddedRow, library.AddedFacts{Work: dune, By: notice.Actor{APIKey: true}}, "works-added-no-root")
	golden(library.DeletedRow, library.DeletedFacts{By: notice.Actor{Name: "Sam"}, Types: []library.DeletedType{{Nouns: books, Count: 1, First: []string{dune.Label}, Work: &dune}}}, "works-deleted-one")
	golden(library.DeletedRow, library.DeletedFacts{By: notice.Actor{Name: "Sam"}, Types: []library.DeletedType{{Nouns: books, Count: 12, First: []string{"Dune (1965)", "Emma (1815)", "Ulysses (1922)"}}}}, "works-deleted-many")
	golden(library.DeletedRow, library.DeletedFacts{By: notice.Actor{APIKey: true}, Types: []library.DeletedType{
		{Nouns: books, Count: 12000, First: []string{"Dune (1965)", "Emma (1815)", "Ulysses (1922)"}},
		{Nouns: zines, Count: 8000, First: []string{"Ark", "Bolt", "Crux"}},
	}}, "works-deleted-two-types")
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
