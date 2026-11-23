package housekeeping

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/setting"
)

// rows is a table a fake batch prunes: each row's id, and whether it is
// kept.
type rows struct {
	kept    map[int]bool
	ids     []int
	batches int
}

func (r *rows) batch(_ context.Context, after, limit int) (Swept[int], error) {
	r.batches++
	var s Swept[int]
	var left []int
	for _, id := range r.ids {
		switch {
		case id <= after || s.Seen == limit:
			left = append(left, id)
		case r.kept[id]:
			s.Seen, s.Last = s.Seen+1, id
			left = append(left, id)
		default:
			s.Seen, s.Deleted, s.Last = s.Seen+1, s.Deleted+1, id
		}
	}
	r.ids = left
	return s, nil
}

func TestDrainPassesAFullBatchOfKeptRowsAndStopsOnAShortOne(t *testing.T) {
	r := &rows{kept: map[int]bool{}}
	for id := 1; id <= 25; id++ {
		r.ids = append(r.ids, id)
		r.kept[id] = id <= 20
	}
	tally, err := Drain(t.Context(), 10, r.batch)
	if err != nil {
		t.Fatal(err)
	}
	if tally != (Tally{Deleted: 5, Kept: 20}) {
		t.Fatalf("tally = %+v, want 5 deleted and 20 kept", tally)
	}
	if len(r.ids) != 20 {
		t.Fatalf("%d rows are left, want the 20 kept", len(r.ids))
	}
	if r.batches != 3 {
		t.Fatalf("Drain ran %d batches, want 3: two full, then a short one", r.batches)
	}

	exact := &rows{kept: map[int]bool{}, ids: []int{1, 2, 3, 4}}
	tally, err = Drain(t.Context(), 2, exact.batch)
	if err != nil {
		t.Fatal(err)
	}
	if tally != (Tally{Deleted: 4}) || exact.batches != 3 {
		t.Fatalf("tally = %+v after %d batches, want 4 deleted after 3: a full last batch needs one more to see the end", tally, exact.batches)
	}
}

func TestAPruneNoteSaysWhatWentAndWhatStayed(t *testing.T) {
	grabs := setting.Noun{One: "Grab", Many: "Grabs"}
	for _, c := range []struct {
		t    Tally
		want string
	}{
		{Tally{}, "Nothing was old enough to delete."},
		{Tally{Deleted: 1204}, "Deleted 1,204 Grabs."},
		{Tally{Deleted: 1}, "Deleted 1 Grab."},
		{Tally{Deleted: 1204, Kept: 12}, "Deleted 1,204 Grabs. Kept 12 that are still in use."},
		{Tally{Kept: 12}, "Kept 12 Grabs that are still in use."},
		{Tally{Kept: 1}, "Kept 1 Grab that is still in use."},
		{Tally{Deleted: 2, Kept: 1}, "Deleted 2 Grabs. Kept 1 that is still in use."},
	} {
		if got := c.t.words(grabs, "that are still in use"); got != c.want {
			t.Errorf("%+v reads %q, want %q", c.t, got, c.want)
		}
	}
	entries := setting.Noun{One: "entry", Many: "entries"}
	if got, want := (Tally{Deleted: 3, Kept: 2}).words(entries, "that are still being sent"), "Deleted 3 entries. Kept 2 that are still being sent."; got != want {
		t.Errorf("History's note reads %q, want %q", got, want)
	}
}
