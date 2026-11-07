package series

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func placedOf(tree works.Tree) works.Placed {
	at := works.Placed{Work: tree.ID, Levels: map[works.Key]works.LevelID{}, Items: map[works.Key]works.ItemID{}}
	for _, l := range tree.Levels {
		at.Levels[l.Key] = l.ID
	}
	for _, it := range tree.Items {
		at.Items[it.Key] = it.ID
	}
	return at
}

// stored runs Store over r in a Write and answers what it reported and how
// many rows SQLite says the Write changed.
func (sh shelf) stored(t *testing.T, at works.Placed, r record) (changed bool, rows int64) {
	t.Helper()
	detail, err := encode(r)
	if err != nil {
		t.Fatal(err)
	}
	err = sh.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var before, after int64
		if err := tx.QueryRowContext(ctx, "SELECT total_changes()").Scan(&before); err != nil {
			return fmt.Errorf("count changes: %w", err)
		}
		if changed, err = sh.s.Store(ctx, tx, at, detail); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, "SELECT total_changes()").Scan(&after); err != nil {
			return fmt.Errorf("count changes: %w", err)
		}
		rows = after - before
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return changed, rows
}

func TestStoreWritesOnlyChangedRows(t *testing.T) {
	sh := open(t)
	r := newShow(81189, "The Long Road", 0, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000)
	tree := sh.add(t, r)
	if len(tree.Items) != 10000 {
		t.Fatalf("added %d episodes, want 10000", len(tree.Items))
	}
	at := placedOf(tree)
	if changed, rows := sh.stored(t, at, r); changed || rows != 0 {
		t.Fatalf("storing the same record reported changed %t and wrote %d rows, want neither", changed, rows)
	}
	retitled := r
	retitled.Show.EnglishTitle = "The Longer Road"
	if changed, rows := sh.stored(t, at, retitled); !changed || rows != 1 {
		t.Fatalf("a title change reported changed %t and wrote %d rows, want one row", changed, rows)
	}
	edited := retitled
	edited.Episodes = slices.Clone(retitled.Episodes)
	edited.Episodes[4321].Overview = "A new overview."
	if changed, rows := sh.stored(t, at, edited); !changed || rows != 1 {
		t.Fatalf("one episode's change reported changed %t and wrote %d rows, want one row", changed, rows)
	}
	tree, err := sh.works.Get(t.Context(), identity.Admin{}, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	ep := itemOf(t, tree, keyOf(int64(edited.Episodes[4321].ID)))
	if f, _ := fieldOf(ep.Fields, "overview"); f.Text != "A new overview." {
		t.Fatalf("the edited episode's overview reads %q, want the new one", f.Text)
	}
}

func TestStoreWritesTheOrderFromTheRecord(t *testing.T) {
	sh := open(t)
	aired := newShow(7101, "Signal Hill", 0, 3)
	aired.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderAbsolute}
	tree := sh.add(t, aired)
	absolute := absoluteOf(aired, db.SeasonID(7101*100+50))
	tree = sh.refresh(t, tree.ID, absolute)
	if f, _ := fieldOf(tree.Fields, "episode_order"); f.Text != "Absolute" {
		t.Fatalf("episode order reads %q after a fetch in Absolute order, want Absolute", f.Text)
	}
	for _, l := range tree.Levels {
		if live := l.WithdrawnAt.IsZero(); live != (l.Key == keyOf(int64(absolute.Seasons[0].ID))) {
			t.Fatalf("level %s live %t: only the Absolute season should be live", l.Key, live)
		}
	}
}

func TestSpecialMovingToSeason3KeepsItsRow(t *testing.T) {
	sh := open(t)
	r := newShow(7001, "Severed", 1, 2, 2)
	tree := sh.add(t, r)
	special := r.Episodes[len(r.Episodes)-1]
	gone := r.Episodes[1]
	before := itemOf(t, tree, keyOf(int64(special.ID)))
	if !before.Aside {
		t.Fatalf("the special isn't aside before the move")
	}

	moved := r
	moved.Seasons = append(slices.Clone(r.Seasons[:2]), season{ID: db.SeasonID(7001*100 + 3), Number: 3})
	moved.Episodes = nil
	for _, e := range r.Episodes {
		switch e.ID {
		case gone.ID:
			continue
		case special.ID:
			e.Season, e.Number, e.AiredSeason, e.AiredNumber = 3, 1, ptr(int64(3)), ptr(int64(1))
		}
		moved.Episodes = append(moved.Episodes, e)
	}
	tree = sh.refresh(t, tree.ID, moved)

	after := itemOf(t, tree, keyOf(int64(special.ID)))
	if after.ID != before.ID || after.Aside || !after.Live() {
		t.Fatalf("the special after the move is %+v, want the same live Item outside the specials", after)
	}
	code, _ := fieldOf(after.Fields, "code")
	number, _ := fieldOf(after.Fields, "season")
	overview, _ := fieldOf(after.Fields, "overview")
	if code.Text != "S03E01" || number.Number != 3 || overview.Text != special.Overview {
		t.Fatalf("the moved special reads %s, season %v, %q, want S03E01, season 3 and its overview", code.Text, number.Number, overview.Text)
	}
	withdrawn := itemOf(t, tree, keyOf(int64(gone.ID)))
	if withdrawn.Live() {
		t.Fatalf("the episode TVDB stopped listing is still live")
	}
	if f, _ := fieldOf(withdrawn.Fields, "code"); f.Text != "S01E02" {
		t.Fatalf("the withdrawn episode's code reads %q, want its last stored S01E02", f.Text)
	}
	for _, l := range tree.Levels {
		if l.Key == keyOf(int64(moved.Seasons[2].ID)) {
			if f, _ := fieldOf(l.Fields, "season"); f.Number != 3 {
				t.Fatalf("season 3's number field reads %v", f.Number)
			}
			return
		}
	}
	t.Fatalf("no season 3 level after the move")
}

func TestAwardsSurviveARefreshWithoutOMDb(t *testing.T) {
	sh := open(t)
	r := newShow(7201, "Quiet Bay", 0, 1)
	r.Show.Awards = ptr("Won 2 Emmys")
	tree := sh.add(t, r)
	unread := r
	unread.Show.Awards = nil
	tree = sh.refresh(t, tree.ID, unread)
	if f, _ := fieldOf(tree.Fields, "awards"); f.Text != "Won 2 Emmys" {
		t.Fatalf("awards read %q after a refresh OMDb wasn't read for, want the stored line", f.Text)
	}
	none := r
	none.Show.Awards = ptr("")
	tree = sh.refresh(t, tree.ID, none)
	if f, ok := fieldOf(tree.Fields, "awards"); ok {
		t.Fatalf("awards read %q after OMDb listed none, want no field", f.Text)
	}
}

func TestARefreshPublishesOnlyARealChange(t *testing.T) {
	sh := open(t)
	r := newShow(7301, "Harbor", 1, 3)
	tree := sh.add(t, r)
	was := sh.sent.n.Load()
	sh.refresh(t, tree.ID, r)
	if n := sh.sent.n.Load(); n != was {
		t.Fatalf("an unchanged refresh published %d changes", n-was)
	}
	edited := r
	edited.Seasons = slices.Clone(r.Seasons)
	edited.Seasons[0].Overview = "A new season overview."
	sh.refresh(t, tree.ID, edited)
	if n := sh.sent.n.Load(); n != was+1 {
		t.Fatalf("a season's new overview published %d changes, want 1", n-was)
	}
}
