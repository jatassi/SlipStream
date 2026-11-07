package files_test

import (
	"context"
	"maps"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// boxed adds a Work of a Scarf in the Winter box and a Button in a spare
// box, wanting from profile, with a folder per level when folders.
func (l *library) boxed(t *testing.T, key works.Key, folders bool, profile targets.ProfileID) works.Tree {
	t.Helper()
	l.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: "Shelf " + string(key), Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
		{Key: "b0", Label: "Odds", Aside: true, Children: []works.Entry{{Key: "c0", Label: "Button"}}},
	}}})
	tree, _, err := l.works.Add(t.Context(), admin, contenttypetest.ID, key, works.Adding{
		Monitor: true, LevelFolders: folders,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, &profile); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestNewFileSitsInItsLevelFolder(t *testing.T) {
	l := open(t)
	web := l.target(t, "Web", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Web", web)
	svc := l.files(files.NoGrabs{})
	foldered := l.boxed(t, "w1", true, p)
	flat := l.boxed(t, "w2", false, p)
	for _, tree := range []works.Tree{foldered, flat} {
		for i := range tree.Items {
			l.download(t, svc, tree, i, release.WEBDL1080p, string(tree.Key)+" "+tree.Items[i].Label)
		}
	}
	l.settle(t, svc)

	want := map[string]string{
		"Shelf w1/Winter/Shelf w1 - Scarf - Web.mkv":       "w1 Scarf",
		"Shelf w1/Spare boxes/Shelf w1 - Button - Web.mkv": "w1 Button",
		"Shelf w2/Shelf w2 - Scarf - Web.mkv":              "w2 Scarf",
		"Shelf w2/Shelf w2 - Button - Web.mkv":             "w2 Button",
	}
	if got := l.paths(t, foldered.ID, flat.ID); !maps.Equal(got, want) {
		t.Fatalf("the files sit at %v, want each in its level's folder when the Work keeps one: %v", got, want)
	}
}
