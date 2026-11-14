package imports

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// dottedFacet places a file at the Item its name labels up to the first
// dot, so the rest of the name can carry release facts.
type dottedFacet struct{ shelfFacet }

func (dottedFacet) Place(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (scanning.Placer, error) {
	items, err := works.ItemsOf(ctx, r, works.Only([]works.WorkID{work}))
	if err != nil {
		return nil, err
	}
	return func(rel string) scanning.Placement {
		label, _, _ := strings.Cut(path.Base(rel), ".")
		if i := slices.IndexFunc(items[work], func(it works.Item) bool { return it.Label == label }); i >= 0 {
			return scanning.Placed(items[work][i].ID)
		}
		return scanning.Unplaced(scanning.Reason{Code: scanning.NoSuchItem, Names: label})
	}, nil
}

func dotted(w *world) *Imports {
	cfg := w.im.cfg
	cfg.Facets = map[works.TypeID]scanning.Facet{contenttypetest.ID: dottedFacet{}}
	return w.newImports(cfg)
}

func TestATestedNameGetsThePathItsImportWouldUnderADraft(t *testing.T) {
	w := newWorld(t)
	tree := w.add(t, "w1", w.root.ID)
	im := dotted(w)
	draft := &files.Draft{Type: contenttypetest.ID, Patch: setting.Patch{"fixture.naming.file.standard": json.RawMessage(`"{Shelf} - {Thing} - {Edition Tag} [{Target}]"`)}}

	stored, err := im.Imagine(t.Context(), identity.Admin{}, tree.ID, `Shelf w1\Clip 2.1080p.BluRay.x264.mkv`, nil)
	if err != nil {
		t.Fatal(err)
	}
	drafted, err := im.Imagine(t.Context(), identity.Admin{}, tree.ID, "Clip 2.1080p.BluRay.x264.mkv", draft)
	if err != nil {
		t.Fatal(err)
	}
	if stored != "Shelf w1/Box/Shelf w1 - Clip 2 - HD.mkv" || drafted != "Shelf w1/Box/Shelf w1 - Clip 2 - [HD].mkv" {
		t.Fatalf("tested paths = %q and %q, want the stored template's and the draft's", stored, drafted)
	}
}

func TestATestedNameWithNoItemOrNoQualityGetsNoPath(t *testing.T) {
	w := newWorld(t)
	tree := w.add(t, "w1", w.root.ID)
	im := dotted(w)

	_, err := im.Imagine(t.Context(), identity.Admin{}, tree.ID, "Clip 9.1080p.BluRay.mkv", nil)
	if unplaced, ok := errors.AsType[*UnplacedError](err); !ok || unplaced.Reason != (scanning.Reason{Code: scanning.NoSuchItem, Names: "Clip 9"}) {
		t.Fatalf("a name of no Item = %v, want the facet's reason", err)
	}
	if _, err := im.Imagine(t.Context(), identity.Admin{}, tree.ID, "Clip 2.webm", nil); !errors.Is(err, ErrNoQuality) {
		t.Fatalf("a name with no quality = %v, want ErrNoQuality", err)
	}
	if _, err := im.Imagine(t.Context(), identity.Admin{}, works.WorkID{UUID: storage.NewUUID()}, "Clip 2.1080p.BluRay.mkv", nil); !errors.Is(err, works.ErrNoWork) {
		t.Fatalf("a name in a Work that isn't there = %v, want ErrNoWork", err)
	}
}
