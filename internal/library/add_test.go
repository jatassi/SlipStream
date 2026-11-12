package library_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

// shelf is a library of contenttypetest Works whose default Profile
// wants one Bluray-1080p Target, with one root to hold their files.
type shelf struct {
	lib   *storage.DB[storage.Library]
	fx    *contenttypetest.Fixture
	works *works.Works
	files *files.Files
	roots *roots.Roots
	root  roots.Root
	hd    targets.TargetID
}

func newShelf(t *testing.T) shelf {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Removed: discard[works.Removed]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	homes, err := roots.Open(ctx, rootstest.Config(t, lib, contenttypetest.ID))
	if err != nil {
		t.Fatal(err)
	}
	root, err := homes.Create(ctx, identity.Admin{}, roots.NewRoot{ContentType: contenttypetest.ID, Node: role.NodeID{UUID: storage.NewUUID()}, Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	goals, err := targets.Open(ctx, targets.Config{Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{}})
	if err != nil {
		t.Fatal(err)
	}
	hd, err := goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{Scores: targets.Scores{Increment: 1}, Name: "HD", Tiers: [][]release.Quality{{release.Bluray1080p}}, Cutoff: release.Bluray1080p})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{Name: "HD", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := goals.MakeDefault(ctx, identity.Admin{}, profile.ID); err != nil {
		t.Fatal(err)
	}
	fs, err := files.Open(ctx, files.Config{Library: lib, Targets: goals, Grabs: grabs.GrabCells{InStep: func(role.NodeID) bool { return true }}, Today: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	return shelf{lib: lib, fx: fx, works: w, roots: homes, root: root.Root, hd: hd.ID, files: fs}
}

func (s shelf) library(leaver library.Leaver) *library.Library {
	return library.New(library.Config{Library: s.lib, Works: s.works, Roots: s.roots, Files: s.files, Grabs: noGrabs{}, Leaver: leaver})
}

type noGrabs struct{}

func (noGrabs) CancelTransfersIn(context.Context, *storage.Tx[storage.Library], []works.WorkID) error {
	return nil
}

func (noGrabs) TouchTransfers() {}

// stock puts Work key in the fixture with one box of three clips.
func (s shelf) stock(key works.Key) {
	s.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: "Shelf " + string(key), Children: []works.Entry{{Key: "b1", Label: "Box", Children: []works.Entry{
		{Key: "c1", Label: "Clip 1"}, {Key: "c2", Label: "Clip 2"}, {Key: "c3", Label: "Clip 3"},
	}}}}})
}

// place records a Bluray-1080p file that fills the HD Target on Item it
// of Work w.
func (s shelf) place(ctx context.Context, tx *storage.Tx[storage.Library], w works.WorkID, it works.Item) error {
	_, err := files.Add(ctx, tx, files.File{
		WorkID: w, Items: []works.ItemID{it.ID}, Cut: release.Standard, RootID: s.root.ID, Path: w.String() + "/" + it.Label + ".mkv", Size: 1 << 30, Quality: release.Bluray1080p,
	}, []targets.TargetID{s.hd})
	return err
}

func TestAddAttachesFilesBeforeThePresetReadsThem(t *testing.T) {
	t.Parallel()
	s := newShelf(t)
	s.stock("w1")
	existing := works.PresetExisting
	added, err := s.library(nil).Add(t.Context(), identity.Admin{}, library.Adding{
		Type: contenttypetest.ID, Key: "w1", Preset: &existing, By: works.AddedByScan,
		Attach: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			trees, err := works.OutlinesIn(ctx, tx, works.Only([]works.WorkID{w.ID}))
			if err != nil {
				return err
			}
			return s.place(ctx, tx, w.ID, trees[0].Items[1])
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var monitored []string
	for _, it := range added.Tree.Items {
		if it.Monitored {
			monitored = append(monitored, it.Label)
		}
	}
	if len(monitored) != 1 || monitored[0] != "Clip 2" {
		t.Fatalf("an add monitoring existing Items that attaches a file to Clip 2 monitors %v, want [Clip 2]", monitored)
	}
	if added.Tree.AddedBy != works.AddedByScan {
		t.Fatalf("a Work a scan adds reads added by %q, want scan", added.Tree.AddedBy)
	}
}

func TestAddRollsBackWhenAttachFails(t *testing.T) {
	t.Parallel()
	s := newShelf(t)
	s.stock("w1")
	refused := errors.New("the folder moved")
	_, err := s.library(nil).Add(t.Context(), identity.Admin{}, library.Adding{
		Type: contenttypetest.ID, Key: "w1",
		Attach: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			trees, err := works.OutlinesIn(ctx, tx, works.Only([]works.WorkID{w.ID}))
			if err != nil {
				return err
			}
			if err := s.place(ctx, tx, w.ID, trees[0].Items[0]); err != nil {
				return err
			}
			return refused
		},
	})
	if !errors.Is(err, refused) {
		t.Fatalf("an add whose Attach fails answers %v, want its error", err)
	}
	var rows []works.Row
	err = s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err = s.works.RowsIn(ctx, r, works.Scope{})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("after an add whose Attach failed the library lists %d Works, want none", len(rows))
	}
}

func TestAddByAnAdminReadsAddedByAdmin(t *testing.T) {
	t.Parallel()
	s := newShelf(t)
	s.stock("w1")
	added, err := s.library(nil).Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: "w1"})
	if err != nil {
		t.Fatal(err)
	}
	if added.Tree.AddedBy != works.AddedByAdmin {
		t.Fatalf("a Work an Admin adds reads added by %q, want admin", added.Tree.AddedBy)
	}
}
