package library_test

import (
	"context"
	"fmt"
	"maps"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/library"
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

func TestDeleteCountsEveryItemFile(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	lib := storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := targets.Open(ctx, targets.Config{Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{}}); err != nil {
		t.Fatal(err)
	}
	root := rootstest.Root(t, lib, role.NodeID{UUID: storage.NewUUID()}, contenttypetest.ID, t.TempDir())
	box := func(key string, aside bool, items ...string) works.Entry {
		e := works.Entry{Key: works.Key(key), Label: "Box " + key, Aside: aside}
		for _, it := range items {
			e.Children = append(e.Children, works.Entry{Key: works.Key(it), Label: "Clip " + it})
		}
		return e
	}
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "Shelf w1", Children: []works.Entry{
		box("b1", false, "c1", "c2", "c3"), box("b2", false, "c4", "c5"), box("b0", true, "c6"),
	}}})
	tree, _, err := w.Add(ctx, identity.Admin{}, contenttypetest.ID, "w1", works.Adding{
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], s works.Summary) error {
			if err := targets.Assign(ctx, tx, s, nil); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, s, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var bytes int64
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i, it := range tree.Items {
			size := int64(i+1) << 20
			bytes += size
			if _, err := files.Add(ctx, tx, files.File{
				WorkID: tree.ID, Items: []works.ItemID{it.ID}, Cut: release.Standard, RootID: root.ID, Path: fmt.Sprintf("Shelf w1/%s.mkv", it.Label), Size: size, Quality: release.WEBDL1080p,
			}, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	l := library.New(library.Config{Library: lib, Works: w})
	plan, outcomes, err := l.Delete(ctx, identity.Admin{}, []works.WorkID{tree.ID}, library.Deleting{Files: library.LeaveFiles, Preview: true})
	if want := (library.DeletePlan{Works: 1, Files: 6, Bytes: bytes}); err != nil || plan != want || len(outcomes) != 1 {
		t.Fatalf("a delete preview of a Work with a file on each of its 6 Items across 3 levels = %+v, %v, %v; want %+v", plan, outcomes, err, want)
	}
}

// heard is a Leaver that remembers each Work it hears of and how many
// file rows it still had then.
type heard map[works.WorkID]int

func (h heard) Leave(ctx context.Context, tx *storage.Tx[storage.Library], ws []works.WorkID) error {
	held, err := files.Holding(ctx, tx, ws)
	if err != nil {
		return err
	}
	for _, id := range ws {
		h[id] = held[id].Files
	}
	return nil
}

func TestLeaveFilesTellsTheLeaverWhileFileRowsStand(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	s := newShelf(t)
	h := heard{}
	l := s.library(h)
	add := func(key works.Key, placed int) works.Tree {
		t.Helper()
		s.stock(key)
		added, err := l.Add(ctx, identity.Admin{}, library.Adding{
			Type: contenttypetest.ID, Key: key,
			Attach: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
				trees, err := works.OutlinesIn(ctx, tx, works.Only([]works.WorkID{w.ID}))
				if err != nil {
					return err
				}
				for _, it := range trees[0].Items[:placed] {
					if err := s.place(ctx, tx, w.ID, it); err != nil {
						return err
					}
				}
				return nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return added.Tree
	}
	left, alsoLeft, removed := add("w1", 2), add("w2", 1), add("w3", 0)

	if _, _, err := l.Delete(ctx, identity.Admin{}, []works.WorkID{left.ID, alsoLeft.ID}, library.Deleting{Files: library.LeaveFiles, Preview: true}); err != nil {
		t.Fatal(err)
	}
	if len(h) != 0 {
		t.Fatalf("a Leave files preview told the Leaver of %v, want nothing", h)
	}
	if _, _, err := l.Delete(ctx, identity.Admin{}, []works.WorkID{left.ID, alsoLeft.ID}, library.Deleting{Files: library.LeaveFiles}); err != nil {
		t.Fatal(err)
	}
	if want := (heard{left.ID: 2, alsoLeft.ID: 1}); !maps.Equal(h, want) {
		t.Fatalf("a Leave files delete told the Leaver %v, want each Work with the file rows it held %v", h, want)
	}
	clear(h)
	if _, _, err := l.Delete(ctx, identity.Admin{}, []works.WorkID{removed.ID}, library.Deleting{Files: library.DeleteFiles}); err != nil {
		t.Fatal(err)
	}
	if len(h) != 0 {
		t.Fatalf("a Delete files delete told the Leaver of %v, want nothing", h)
	}
}
