package files_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

const (
	longLevels = 40
	longItems  = 30
)

// longWork is a Work of 40 levels of 30 Items wanting three Targets, each
// Item holding a WEB-DL that fills all three.
func longWork(b *testing.B) (*library, works.Tree) {
	b.Helper()
	l := open(b)
	tiers := [][]release.Quality{{release.Bluray1080p}, {release.WEBDL1080p}}
	var ts []targets.Target
	for _, name := range []string{"Best", "HD", "Any"} {
		ts = append(ts, l.target(b, name, tiers...))
	}
	profile := l.profile(b, "Long", ts...)
	fetched := works.Fetched{Label: "Evening Tides"}
	for s := 1; s <= longLevels; s++ {
		level := works.Entry{Key: works.Key(fmt.Sprintf("b%d", s)), Label: fmt.Sprintf("Season %d", s)}
		for e := 1; e <= longItems; e++ {
			level.Children = append(level.Children, works.Entry{Key: works.Key(fmt.Sprintf("c%d-%d", s, e)), Label: fmt.Sprintf("Episode %d", e)})
		}
		fetched.Children = append(fetched.Children, level)
	}
	l.fx.Put("long", contenttypetest.Record{Tree: fetched})
	tree, _, err := l.works.Add(b.Context(), admin, contenttypetest.ID, "long", works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, &profile); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		b.Fatal(err)
	}
	fills := []targets.TargetID{ts[0].ID, ts[1].ID, ts[2].ID}
	for i, it := range tree.Items {
		l.place(b, it.ID, tree.ID, fmt.Sprintf("Evening Tides/%d.mkv", i), release.WEBDL1080p, fills...)
	}
	return l, tree
}

var errRolledBack = errors.New("rolled back")

// BenchmarkPlacingOneFileInALongWork measures one file placed in a Work of
// 1,200 Items, each holding a file filling three Targets: Judge is the
// Read that loads the Work, refills all 3,600 cells and names all 1,201
// files; Import is that and its writes, in a Write rolled back each time.
func BenchmarkPlacingOneFileInALongWork(b *testing.B) {
	l, tree := longWork(b)
	svc := l.files(files.NoGrabs{})
	in := holdingOf(tree, 0, files.HoldingOf(storage.NewUUID(), ".mkv"), 1024, media.Record{Video: media.Video{Width: 1920, Height: 1080}}, release.Bluray1080p)
	b.Run("Judge", func(b *testing.B) {
		for b.Loop() {
			err := l.lib.Read(b.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
				v, err := svc.Judge(ctx, r, files.Batch{Work: tree.ID, Root: l.root.ID, Candidates: []files.Candidate{in}})
				if err == nil && len(v[0].Fills) != 3 {
					b.Fatalf("the Bluray fills %d Targets, want 3", len(v[0].Fills))
				}
				return err
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Import", func(b *testing.B) {
		for b.Loop() {
			err := l.lib.Write(b.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				if _, err := l.land(ctx, tx, svc, tree.ID, in); err != nil {
					return err
				}
				return errRolledBack
			})
			if !errors.Is(err, errRolledBack) {
				b.Fatal(err)
			}
		}
	})
}

const manyWorks = 500

// BenchmarkPreviewingAChangeToAProfileOfManyWorks measures the preview of
// reordering a Profile 500 Works want from, each of two Items holding a
// file filling both its Targets: the preview loads, refills and renames
// all 1,000 files and rolls back.
func BenchmarkPreviewingAChangeToAProfileOfManyWorks(b *testing.B) {
	l := open(b)
	tiers := [][]release.Quality{{release.Bluray1080p}, {release.WEBDL1080p}}
	hd, wide := l.target(b, "HD", tiers...), l.target(b, "Any", tiers...)
	profile := l.profile(b, "Many", hd, wide)
	for w := range manyWorks {
		key := works.Key(fmt.Sprintf("w%d", w))
		l.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: fmt.Sprintf("Shelf %d", w), Children: []works.Entry{
			{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}, {Key: "c2", Label: "Mittens"}}},
		}}})
		tree, _, err := l.works.Add(b.Context(), admin, contenttypetest.ID, key, works.Adding{
			Monitor: true,
			Also: func(ctx context.Context, tx *storage.Tx[storage.Library], s works.Summary) error {
				return targets.Assign(ctx, tx, s, &profile)
			},
		})
		if err != nil {
			b.Fatal(err)
		}
		for i, it := range tree.Items {
			l.place(b, it.ID, tree.ID, fmt.Sprintf("Shelf %d/%d.mkv", w, i), release.WEBDL1080p, hd.ID, wide.ID)
		}
	}
	svc := l.files(files.NoGrabs{})
	reordered := targets.UpdateProfile{ID: profile, Spec: targets.ProfileSpec{Name: "Many", ContentType: contenttypetest.ID, Entries: []targets.Entry{
		{Target: wide.ID, Sourcing: targets.Download}, {Target: hd.ID, Sourcing: targets.Download},
	}}}
	for b.Loop() {
		i, err := svc.Change(b.Context(), admin, reordered, files.Mode{Unwanted: files.Delete, Preview: true})
		if err != nil {
			b.Fatal(err)
		}
		if i.Works != manyWorks || i.Renamed != 2*manyWorks {
			b.Fatalf("the preview = %+v, want %d Works and every file renamed", i, manyWorks)
		}
	}
}
