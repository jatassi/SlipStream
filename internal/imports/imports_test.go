package imports

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestAddAttachesIntoALibraryWorkOfItsRootAndAddsAnyOtherSuggestion(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	ctx := t.Context()
	shelved := w.add(t, "w1", w.root.ID)
	w.stock("w2")
	w.pend(t, pend{Folder: "Shelf w1", Suggestion: &Suggestion{Key: "w1", Label: "Shelf w1"}, Files: []PendingFile{found("Shelf w1/Clip 2.mkv")}})
	w.pend(t, pend{Folder: "Shelf w2", Suggestion: &Suggestion{Key: "w2", Label: "Shelf w2"}, Files: []PendingFile{found("Shelf w2/Clip 1.mkv"), found("Shelf w2/Clip 3.mkv")}})
	w.pend(t, pend{Folder: "Odd", Files: []PendingFile{found("Odd/Clip 1.mkv")}})
	ids := make([]EntryID, 3)
	for i, folder := range []string{"Shelf w1", "Shelf w2", "Odd"} {
		ids[i] = w.at(t, folder).ID
	}

	outs, err := w.im.Add(ctx, identity.Admin{}, ids)
	if err != nil {
		t.Fatal(err)
	}
	if outs[0].Err != nil || outs[0].Work != shelved.ID {
		t.Fatalf("Add of an entry suggesting a Work in its root = %+v, want its files in that Work", outs[0])
	}
	if got := w.filesOf(t, shelved.ID); !slices.Equal(got, []string{"Shelf w1/Clip 2.mkv"}) {
		t.Errorf("the library Work's files = %v, want the entry's file", got)
	}
	if outs[1].Err != nil || outs[1].Work.IsZero() {
		t.Fatalf("Add of an entry suggesting a Work not in the library = %+v, want it added", outs[1])
	}
	added, err := w.works.Get(ctx, identity.Admin{}, outs[1].Work)
	if err != nil {
		t.Fatal(err)
	}
	if added.Key != "w2" || added.AddedBy != works.AddedByAdmin {
		t.Errorf("the Work Add made = %s added by %q, want w2 added by an Admin", added.Key, added.AddedBy)
	}
	if got := w.filesOf(t, added.ID); !slices.Equal(got, []string{"Shelf w2/Clip 1.mkv", "Shelf w2/Clip 3.mkv"}) {
		t.Errorf("the added Work's files = %v, want both of the entry's files", got)
	}
	if !errors.Is(outs[2].Err, ErrNoSuggestion) {
		t.Errorf("Add of an entry with no suggestion = %v, want ErrNoSuggestion on its own", outs[2].Err)
	}
	if got := w.entries(t, false); len(got) != 1 || got[0].Folder != "Odd" {
		t.Errorf("Pending imports after Add lists %+v, want only the entry Add refused", got)
	}
	if got := len(w.moved.take()); got != 2 {
		t.Errorf("Add published files.changed %d times, want once per Work its files joined", got)
	}
}

func TestAddRefusesAWorkThatLivesInAnotherRoot(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	other := w.newRoot(t)
	elsewhere := w.add(t, "w1", other.ID)
	w.pend(t, pend{Folder: "Shelf w1", Suggestion: &Suggestion{Key: "w1", Label: "Shelf w1"}, Files: []PendingFile{found("Shelf w1/Clip 1.mkv")}})

	outs, err := w.im.Add(t.Context(), identity.Admin{}, []EntryID{w.at(t, "Shelf w1").ID})
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(outs[0].Err, ErrElsewhere) {
		t.Fatalf("Add into a Work of another root = %+v, want ErrElsewhere", outs[0])
	}
	if got := paths(w.at(t, "Shelf w1")); !slices.Equal(got, []string{"Shelf w1/Clip 1.mkv"}) {
		t.Errorf("the refused entry lists %v, want its file still", got)
	}
	if got := w.filesOf(t, elsewhere.ID); len(got) != 0 {
		t.Errorf("the other root's Work holds %v, want nothing", got)
	}
}

func TestAddLeavesTheFilesPlaceCantPlaceUnderTheWork(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.stock("w1")
	w.pend(t, pend{Folder: "Shelf w1", Suggestion: &Suggestion{Key: "w1", Label: "Shelf w1"}, Files: []PendingFile{found("Shelf w1/Clip 1.mkv"), found("Shelf w1/Clip 9.mkv")}})

	outs, err := w.im.Add(t.Context(), identity.Admin{}, []EntryID{w.at(t, "Shelf w1").ID})
	if err != nil || outs[0].Err != nil {
		t.Fatalf("Add = %+v, %v", outs, err)
	}
	if got := w.filesOf(t, outs[0].Work); !slices.Equal(got, []string{"Shelf w1/Clip 1.mkv"}) {
		t.Errorf("the added Work's files = %v, want the one file Place placed", got)
	}
	e := w.at(t, "Shelf w1")
	if got := paths(e); !slices.Equal(got, []string{"Shelf w1/Clip 9.mkv"}) {
		t.Fatalf("the entry lists %v after Add, want the file Place couldn't place", got)
	}
	if r := e.Files[0].Reason; r.Code != scanning.NoSuchItem || r.Names != "Clip 9" {
		t.Errorf("the file left reads %+v, want Place's reason", r)
	}
	if e.Suggestion == nil || e.Suggestion.Key != "w1" || e.Suggestion.WorkID != outs[0].Work {
		t.Errorf("the entry left suggests %+v, want the Work its other file joined", e.Suggestion)
	}
}

func TestIntoAttachesTheChosenFilesAndRefusesAWrongRootItemOrPath(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	ctx := t.Context()
	target := w.add(t, "w1", w.root.ID)
	stranger := w.add(t, "w2", w.root.ID)
	elsewhere := w.add(t, "w3", w.newRoot(t).ID)
	w.pend(t, pend{Folder: "Odd", Files: []PendingFile{found("Odd/a.mkv"), found("Odd/b.mkv")}})
	id := w.at(t, "Odd").ID

	for _, c := range []struct {
		name   string
		work   works.WorkID
		chosen []Chosen
		want   error
	}{
		{"a Work in another root", elsewhere.ID, []Chosen{{Path: "Odd/a.mkv", Item: elsewhere.Items[0].ID}}, ErrElsewhere},
		{"an Item of another Work", target.ID, []Chosen{{Path: "Odd/a.mkv", Item: stranger.Items[0].ID}}, ErrNotOfWork},
		{"a path the entry doesn't list", target.ID, []Chosen{{Path: "Odd/c.mkv", Item: target.Items[0].ID}}, ErrChanged},
	} {
		if _, err := w.im.Into(ctx, identity.Admin{}, id, c.work, c.chosen); !errors.Is(err, c.want) {
			t.Errorf("Into %s = %v, want %v", c.name, err, c.want)
		}
	}
	if got := paths(w.at(t, "Odd")); len(got) != 2 {
		t.Fatalf("after refused Intos the entry lists %v, want both files", got)
	}

	left, err := w.im.Into(ctx, identity.Admin{}, id, target.ID, []Chosen{{Path: "Odd/a.mkv", Item: target.Items[1].ID}})
	if err != nil {
		t.Fatal(err)
	}
	if left == nil || !slices.Equal(paths(left), []string{"Odd/b.mkv"}) {
		t.Fatalf("Into of one file answers %+v, want the entry with the other file", left)
	}
	if got := w.filesOf(t, target.ID); !slices.Equal(got, []string{"Odd/a.mkv"}) {
		t.Errorf("the Work's files = %v, want the chosen file", got)
	}
	left, err = w.im.Into(ctx, identity.Admin{}, id, target.ID, []Chosen{{Path: "Odd/b.mkv", Item: target.Items[2].ID}})
	if err != nil || left != nil {
		t.Fatalf("Into of the last file answers %+v, %v; want no entry left", left, err)
	}
	if w.at(t, "Odd") != nil {
		t.Errorf("Pending imports still lists the entry whose files all joined a Work")
	}
	if _, err := w.im.Into(ctx, identity.Admin{}, id, target.ID, nil); !errors.Is(err, ErrNoEntry) {
		t.Errorf("Into a deleted entry = %v, want ErrNoEntry", err)
	}
}

func TestMatchKeepsItsSuggestionUntilTheHintChanges(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.stock("w9")
	w.pend(t, pend{Folder: "Shelf (2001)", Suggestion: &Suggestion{Key: "w1", Label: "Shelf w1"}, Files: []PendingFile{found("Shelf (2001)/a.mkv")}})
	id := w.at(t, "Shelf (2001)").ID

	e, err := w.im.Match(t.Context(), identity.Admin{}, id, "w9")
	if err != nil {
		t.Fatal(err)
	}
	if e.Suggestion == nil || e.Suggestion.Key != "w9" || e.Suggestion.Label != "Shelf w9" || e.Suggestion.Confident {
		t.Fatalf("Match answers the suggestion %+v, want w9, not confident", e.Suggestion)
	}
	w.pend(t, pend{Folder: "Shelf (2001)", Suggestion: &Suggestion{Key: "w2", Confident: true}, Listed: []string{"Shelf (2001)/a.mkv"}, Files: []PendingFile{found("Shelf (2001)/b.mkv")}})
	if s := w.at(t, "Shelf (2001)").Suggestion; s == nil || s.Key != "w9" {
		t.Errorf("a scan with the same hint left the suggestion %+v, want the Admin's w9", s)
	}
	w.pend(t, pend{Folder: "Shelf (2001)", Hint: scanning.Hint{Title: "Shelf", Year: 2002}, Suggestion: &Suggestion{Key: "w2"}, Listed: []string{"Shelf (2001)/a.mkv", "Shelf (2001)/b.mkv"}})
	if s := w.at(t, "Shelf (2001)").Suggestion; s == nil || s.Key != "w2" {
		t.Errorf("a scan whose hint changed left the suggestion %+v, want its own w2", s)
	}
}

func TestAnIgnoredEntryStaysIgnoredUntilItsFilesChange(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	ctx := t.Context()
	w.pend(t, pend{Folder: "Odd", Files: []PendingFile{found("Odd/a.mkv"), found("Odd/b.mkv")}})
	id := w.at(t, "Odd").ID
	if err := w.im.Ignore(ctx, identity.Admin{}, []EntryID{id}); err != nil {
		t.Fatal(err)
	}
	if n, err := w.im.Count(ctx); err != nil || n != 0 {
		t.Fatalf("Count after Ignore = %d, %v; want 0", n, err)
	}
	if got := w.changed.take(); len(got) != 1 || !slices.Equal(got[0].RootIDs, []string{w.root.ID.String()}) {
		t.Errorf("Ignore published %+v, want imports.changed for its root", got)
	}
	w.pend(t, pend{Folder: "Odd", Listed: []string{"Odd/a.mkv", "Odd/b.mkv"}})
	if e := w.at(t, "Odd"); e == nil || !e.Ignored() {
		t.Fatalf("a scan that found the same files left %+v, want it ignored", e)
	}
	w.pend(t, pend{Folder: "Odd", Listed: []string{"Odd/a.mkv"}})
	if e := w.at(t, "Odd"); e == nil || e.Ignored() || !slices.Equal(paths(e), []string{"Odd/a.mkv"}) {
		t.Fatalf("a scan that found a file gone left %+v, want it listed again with the file left", e)
	}
	if err := w.im.Ignore(ctx, identity.Admin{}, []EntryID{id}); err != nil {
		t.Fatal(err)
	}
	w.pend(t, pend{Folder: "Odd", Listed: []string{"Odd/a.mkv"}, Files: []PendingFile{found("Odd/c.mkv")}})
	if e := w.at(t, "Odd"); e == nil || e.Ignored() {
		t.Fatalf("a scan that found a new file left %+v, want it listed again", e)
	}
	if err := w.im.Ignore(ctx, identity.Admin{}, []EntryID{id}); err != nil {
		t.Fatal(err)
	}
	if err := w.im.Unignore(ctx, identity.Admin{}, []EntryID{id}); err != nil {
		t.Fatal(err)
	}
	if n, err := w.im.Count(ctx); err != nil || n != 1 {
		t.Errorf("Count after Unignore = %d, %v; want 1", n, err)
	}
}

func TestAnEntryWhoseFilesAreAllGoneIsDeleted(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.pend(t, pend{Folder: "Odd", Files: []PendingFile{found("Odd/a.mkv")}})
	w.pend(t, pend{Folder: "Odd"})
	if e := w.at(t, "Odd"); e != nil {
		t.Fatalf("a scan that found none of an entry's files left %+v, want it deleted", e)
	}
}

func TestLeaveListsTheLeftFilesUnderIgnoredWithTheirRecords(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	ctx := t.Context()
	left := w.add(t, "w1", w.root.ID)
	w.place(t, left, "Shelf w1/Clip 1.mkv")
	w.place(t, left, "Shelf w1/Box/Clip 2.mkv")
	w.place(t, left, "Clip 3.mkv")
	w.pend(t, pend{Folder: "Shelf w1", Files: []PendingFile{found("Shelf w1/extra.mkv")}})

	if _, _, err := w.library.Delete(ctx, identity.Admin{}, []works.WorkID{left.ID}, library.Deleting{Files: library.LeaveFiles}); err != nil {
		t.Fatal(err)
	}
	if got := w.entries(t, false); len(got) != 0 {
		t.Errorf("after a Leave files delete Pending imports needs the Admin for %+v, want nothing", got)
	}
	folder := w.at(t, "Shelf w1")
	if folder == nil || !folder.Ignored() || !slices.Equal(paths(folder), []string{"Shelf w1/Box/Clip 2.mkv", "Shelf w1/Clip 1.mkv", "Shelf w1/extra.mkv"}) {
		t.Fatalf("the Work folder's entry = %+v, want it ignored with the left files beside the one it listed", folder)
	}
	if folder.Hint.Title != "Shelf w1" {
		t.Errorf("the entry's hint = %+v, want the folder's name read", folder.Hint)
	}
	for _, f := range folder.Files[:2] {
		if f.Reason.Code != scanning.Left || f.Media == nil || f.Media.Width != 1920 || f.Seen == nil || f.Size != 1<<30 {
			t.Errorf("left file %+v, want its stored record with reason left", f)
		}
	}
	loose := w.at(t, "Clip 3.mkv")
	if loose == nil || !loose.Ignored() || !slices.Equal(paths(loose), []string{"Clip 3.mkv"}) {
		t.Errorf("the loose file's entry = %+v, want it ignored on its own", loose)
	}
}

func TestPendingLeadsWithEachComparedRoot(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	ctx := t.Context()
	other := w.newRoot(t)
	w.pend(t, pend{Folder: "Odd", Files: []PendingFile{found("Odd/a.mkv")}})
	w.pend(t, pend{Folder: "Even", Files: []PendingFile{found("Even/a.mkv")}})
	elsewhere := found("Heat/a.mkv")
	elsewhere.Reason = scanning.Reason{Code: scanning.Elsewhere, Title: "Heat", Year: 1995, Root: other.ID.String()}
	w.pend(t, pend{Folder: "Heat", Files: []PendingFile{elsewhere}})
	err := w.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return record(ctx, tx, w.root.ID, Comparison{Place: w.root.Place, Found: 400, Added: 398}, storage.Now())
	})
	if err != nil {
		t.Fatal(err)
	}

	l, err := w.im.Pending(ctx, identity.Admin{}, Filter{Root: &w.root.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Leads) != 1 || l.Leads[0] != (Lead{Root: w.root.ID, ContentType: w.root.ContentType, Added: 398, NeedsYou: 3}) {
		t.Errorf("Pending leads with %+v, want the root's 398 added and 3 entries that need the Admin", l.Leads)
	}
	if len(l.Entries) != 3 || l.Entries[0].Folder != "Even" {
		t.Errorf("Pending lists %+v, want the root's 3 entries in folder order", l.Entries)
	}
	if l.Roots[w.root.ID] != "Shelves library root at "+w.root.Path+" on Media server" || l.Roots[other.ID] != "Shelves library root at "+other.Path+" on Media server" {
		t.Errorf("Pending words its roots %v, want its own and the one a reason names", l.Roots)
	}
	l, err = w.im.Pending(ctx, identity.Admin{}, Filter{Root: &other.ID})
	if err != nil || len(l.Leads) != 0 || len(l.Entries) != 0 {
		t.Errorf("Pending of a root never compared = %+v, %v; want nothing", l, err)
	}
}
