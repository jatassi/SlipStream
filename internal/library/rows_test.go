package library_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

// budget is the spec's: GET /library answers 20,000 Works within 500 ms.
const budget = 500 * time.Millisecond

// BenchmarkLibraryRows20k builds the whole Library's rows over 20,000
// one-Item Works wanting two Targets, half of them with a file filling
// each, every one with a poster, two ratings and two listed fields.
func BenchmarkLibraryRows20k(b *testing.B) {
	const count = 20_000
	l := seeded(b, count)
	var slowest time.Duration
	for b.Loop() {
		start := time.Now()
		rows, err := l.Rows(b.Context(), identity.Admin{}, works.Scope{})
		slowest = max(slowest, time.Since(start))
		if err != nil {
			b.Fatal(err)
		}
		if len(rows) != count || rows[0].Work.Poster.IsZero() || rows[0].Work.Rating == nil || len(rows[0].Work.Values) != 2 ||
			rows[0].Rollup.Status != files.Met || rows[1].Rollup.Status != files.Missing || rows[0].Rollup.Files != 2 {
			b.Fatalf("Rows answered %d rows, the first %+v, want %d with the first met and the second missing", len(rows), rows[0], count)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N), "ms/rows")
	b.ReportMetric(float64(budget.Milliseconds()), "budget-ms")
	b.Logf("slowest of %d builds: %s, against a %s budget", b.N, slowest.Round(time.Millisecond), budget)
}

// seeded is a Library of count Works added through the fixture, each
// even one with a file at each Target.
func seeded(b *testing.B, count int) *library.Library {
	ctx := b.Context()
	lib := storagetest.Library(b, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		b.Fatal(err)
	}
	goals, err := targets.Open(ctx, targets.Config{Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{}})
	if err != nil {
		b.Fatal(err)
	}
	wanted := wantBoth(b, goals)
	root := rootstest.Root(b, lib, role.NodeID{UUID: storage.NewUUID()}, contenttypetest.ID, b.TempDir())
	start := time.Now()
	for i := range count {
		key := works.Key(fmt.Sprintf("w%05d", i))
		label := fmt.Sprintf("Work %05d", i)
		fx.Put(key, contenttypetest.Record{
			Tree: works.Fetched{
				Label: label, Children: []works.Entry{{Key: "b1", Label: "Box", Children: []works.Entry{{Key: "c1", Label: label}}}},
				Ratings: []works.Rating{{Source: "imdb", Value: 7.1}, {Source: "tmdb", Value: 6.8}},
				Artwork: []works.Art{{Kind: "photo", Ref: artwork.Ref{Source: "fixture", Shape: artwork.ShapePoster, Path: "/" + string(key) + ".jpg"}}},
			},
			WorkFields: []works.FieldValue{works.TextField("origin", "Here"), works.TextsField("tags", []string{"One", "Two"})},
		})
		tree, _, err := w.Add(ctx, identity.Admin{}, contenttypetest.ID, key, works.Adding{
			Monitor: true,
			Also: func(ctx context.Context, tx *storage.Tx[storage.Library], s works.Summary) error {
				if err := targets.Assign(ctx, tx, s, nil); err != nil {
					return err
				}
				return roots.Assign(ctx, tx, s, nil)
			},
		})
		if err != nil {
			b.Fatal(err)
		}
		if i%2 == 0 {
			fill(b, lib, tree, root.ID, wanted)
		}
	}
	b.Logf("seeded %d Works in %s", count, time.Since(start).Round(time.Millisecond))
	shelf := files.New(files.Config{Library: lib, Targets: goals, Grabs: grabs.GrabCells{InStep: func(role.NodeID) bool { return true }}, Today: time.Now})
	return library.New(library.Config{Library: lib, Works: w, Targets: goals, Files: shelf})
}

type wanted struct {
	quality release.Quality
	target  targets.TargetID
}

// wantBoth makes the default Profile want an HD and a 2160p Target, and
// answers the quality that meets each.
func wantBoth(b *testing.B, goals *targets.Targets) []wanted {
	ctx := b.Context()
	hd, err := goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{Name: "HD", Tiers: [][]release.Quality{{release.Bluray1080p}}, Cutoff: release.Bluray1080p})
	if err != nil {
		b.Fatal(err)
	}
	uhd, err := goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{Name: "2160p", Tiers: [][]release.Quality{{release.WEBDL2160p}}, Cutoff: release.WEBDL2160p})
	if err != nil {
		b.Fatal(err)
	}
	archive, err := goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{Name: "Archive", ContentType: contenttypetest.ID, Entries: []targets.Entry{
		{Target: hd.ID, Sourcing: targets.Download}, {Target: uhd.ID, Sourcing: targets.Download},
	}})
	if err != nil {
		b.Fatal(err)
	}
	if _, err := goals.MakeDefault(ctx, identity.Admin{}, archive.ID); err != nil {
		b.Fatal(err)
	}
	return []wanted{{release.Bluray1080p, hd.ID}, {release.WEBDL2160p, uhd.ID}}
}

// fill places one file in root for each of ws on tree's Item.
func fill(b *testing.B, lib *storage.DB[storage.Library], tree works.Tree, root roots.ID, ws []wanted) {
	err := lib.Write(b.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, w := range ws {
			_, err := files.Add(ctx, tx, files.File{
				WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: root,
				Path: tree.Label + "/" + tree.Label + " - " + w.quality.String() + ".mkv", Size: 1 << 30, Quality: w.quality,
			}, []targets.TargetID{w.target})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
}
