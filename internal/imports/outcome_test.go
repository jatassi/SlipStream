package imports

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
)

// fileIn is a root file in outcome o with the columns the CHECKs ask of o.
func fileIn(o Outcome) File {
	f := File{Path: "Odd/a.mkv", Cut: "standard", ReleaseName: "a.mkv", SourceName: "a.mkv", Outcome: o}
	switch o {
	case LeftOut, NotBetter, Waiting, FailedOut, Declined:
		f.Reason = &Reason{Code: ReasonCode(scanning.NoMatch), Scan: &scanning.Reason{Code: scanning.NoMatch}}
	case Placed:
		f.PlacedBy = InPlace
	case Pending, Sending, Placing:
	}
	return f
}

func TestSetFileTakesExactlyTheMovesTheTableListsAndTheCheckHoldsEveryOutcome(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	write := func(fn func(ctx context.Context, tx *storage.Tx[storage.Library]) error) error {
		return w.lib.Write(t.Context(), fn)
	}
	insert := func(o Outcome) (ImportID, error) {
		im := Import{ID: ImportID{UUID: storage.NewUUID()}, Source: FromRoot, Root: w.root.ID, Folder: "Odd", ContentType: contenttypetest.ID, Created: storage.Now(), Finished: storage.Now(), Files: []File{fileIn(o)}}
		return im.ID, write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return insertImport(ctx, tx, im) })
	}
	if _, err := insert("bogus"); err == nil {
		t.Errorf("the outcome CHECK took an outcome no Outcome names")
	}
	for _, from := range Outcomes {
		for _, to := range Outcomes {
			id, err := insert(from)
			if err != nil {
				t.Fatalf("the outcome CHECK refused a %s file: %v", from, err)
			}
			err = write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				return setFile(ctx, db.New(tx), id, from, fileIn(to))
			})
			if want := from == to || slices.Contains(moves[from], to); (err == nil) != want {
				t.Errorf("setFile %s to %s = %v, want it taken %v as moves lists", from, to, err, want)
			}
		}
	}
	id, err := insert(Waiting)
	if err != nil {
		t.Fatal(err)
	}
	err = write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return setFile(ctx, db.New(tx), id, Pending, fileIn(Waiting))
	})
	if !errors.Is(err, errMoved) {
		t.Errorf("setFile on a file that no longer reads the outcome it was read at = %v, want errMoved", err)
	}
}

func TestEveryMoveIsBetweenOutcomesAndNoFinalOutcomeMovesButARetryOrASettingsWait(t *testing.T) {
	t.Parallel()
	for from, tos := range moves {
		if from.Final() && from != FailedOut && (from != LeftOut || !slices.Equal(tos, []Outcome{Waiting})) {
			t.Errorf("the final outcome %s moves to %v", from, tos)
		}
		for _, to := range append([]Outcome{from}, tos...) {
			if !slices.Contains(Outcomes, to) {
				t.Errorf("moves names %s, which no outcome CHECK value is", to)
			}
		}
	}
}
