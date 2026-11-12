package imports

import (
	"context"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// stackedFacet places a file as shelfFacet does, as part n of its Item
// when its name ends in " - cd" and n.
type stackedFacet struct{ shelfFacet }

func (stackedFacet) Place(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (scanning.Placer, error) {
	place, err := shelfFacet{}.Place(ctx, r, work)
	if err != nil {
		return nil, err
	}
	return func(rel string) scanning.Placement {
		label, n, ok := strings.Cut(strings.TrimSuffix(path.Base(rel), path.Ext(rel)), " - cd")
		if !ok {
			return place(rel)
		}
		items, _ := place(label).Items()
		part, _ := strconv.Atoi(n)
		return scanning.PlacedPart(part, items...)
	}, nil
}

// TestAWorkSplitIntoPartsWaitsPartsTogether opens a folder holding an
// Item in two parts, an Item whole and an Item's lone part. The two parts
// wait parts together, and the rest go on to the decision.
func TestAWorkSplitIntoPartsWaitsPartsTogether(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.cfg.Facets = map[works.TypeID]scanning.Facet{contenttypetest.ID: stackedFacet{}}
	one := w.add(t, "w1", w.root.ID)
	dir := w.downloads(t, nil)
	write(t, dir, "Shelf w1/Clip 1 - cd1.mkv", "Shelf w1/Clip 1 - cd2.mkv", "Shelf w1/Clip 2.mkv", "Shelf w1/Clip 3 - cd1.mkv")
	w.reconcile(t)

	_, byWork := w.opened(t, dir)
	imp := w.until(t, "the whole file and the lone part are decided", byWork[one.ID].ID, func(imp *Import) bool {
		return imp.Files[positionOf(imp, "Clip 2.mkv")].Outcome == Waiting && imp.Files[positionOf(imp, "Clip 3 - cd1.mkv")].Outcome == Waiting
	})
	said, err := w.im.Sentences(t.Context(), identity.Admin{}, *imp)
	if err != nil {
		t.Fatal(err)
	}
	for _, base := range []string{"Clip 1 - cd1.mkv", "Clip 1 - cd2.mkv"} {
		p := positionOf(imp, base)
		if f := imp.Files[p]; f.Reason.Code != Parts || f.Reason.Parts != 2 {
			t.Errorf("%s waits %+v, want parts of 2", base, f.Reason)
		}
		if want := "This shelf is split into 2 files. SlipStream imports a shelf as one file."; said[p] != want {
			t.Errorf("%s reads %q, want %q", base, said[p], want)
		}
	}
	for _, base := range []string{"Clip 2.mkv", "Clip 3 - cd1.mkv"} {
		if f := imp.Files[positionOf(imp, base)]; f.Reason.Code == Parts {
			t.Errorf("%s waits parts, want the decision's verdict: a whole file and a lone part each hold their Item", base)
		}
	}
}
