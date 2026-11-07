package series_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/series"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type noPrefetch struct{}

func (noPrefetch) Prefetch([]artwork.Want) {}

func TestTheLibraryAuditPassesWithSeries(t *testing.T) {
	storagetest.AuditLibrary(t, node.LibrarySchemas(contenttype.NewSet(series.Def)))
}

func TestWorksOpensWithTheSeriesDescription(t *testing.T) {
	lib := series.Def.OpenLibrary(contenttype.LibraryDeps{Region: metadata.Region, Log: slog.New(slog.DiscardHandler)}, series.Def.Endpoints())
	w, err := works.Open(t.Context(), works.Config{
		Library:  storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(series.Def))...),
		Catalogs: map[works.TypeID]works.Catalog{series.Def.ID(): lib}, Timings: works.DefaultTimings(),
		Artwork: noPrefetch{}, Changed: discard[works.Changed]{}, Removed: discard[works.Removed]{},
		TypesChanged: discard[works.ContentTypesChanged]{}, TypeChanged: discard[works.TypeChanged]{},
		Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	types, err := w.Types(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].ID != series.Def.ID() || types[0].Aside != (works.Noun{One: "Special", Many: "Specials"}) {
		t.Fatalf("Types = %+v, want series with Specials aside", types)
	}
}
