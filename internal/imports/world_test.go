package imports

import (
	"context"
	"path"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// world is a library of contenttypetest Works on one Node, "Media
// server", whose default Profile wants one Bluray-1080p Target, with one
// root to hold their files.
type world struct {
	lib     *storage.DB[storage.Library]
	fx      *contenttypetest.Fixture
	works   *works.Works
	roots   *roots.Roots
	files   *files.Files
	library *library.Library
	im      *Imports
	root    roots.Root
	node    role.NodeID
	hd      targets.TargetID
	changed *heard[Changed]
	moved   *heard[files.Changed]
	steps   map[tasks.UnitKey]tasks.Step
}

type heard[T any] struct {
	mu   sync.Mutex
	sent []T
}

func (h *heard[T]) Publish(_ context.Context, v T) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sent = append(h.sent, v)
}

func (h *heard[T]) take() []T {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := h.sent
	h.sent = nil
	return out
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type noGrabs struct{}

func (noGrabs) CancelTransfersIn(context.Context, *storage.Tx[storage.Library], []works.WorkID) error {
	return nil
}

func (noGrabs) TouchTransfers() {}

// shelfFacet hints a folder's name as release names write it and places
// a file at the Item its name, without extension, labels.
type shelfFacet struct {
	scanning.None
	release.TitleOnly
}

func (shelfFacet) Hint(name string) scanning.Hint { return scanning.ReadName(name) }

func (shelfFacet) Place(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (scanning.Placer, error) {
	items, err := works.ItemsOf(ctx, r, works.Only([]works.WorkID{work}))
	if err != nil {
		return nil, err
	}
	return func(rel string) scanning.Placement {
		label := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
		if i := slices.IndexFunc(items[work], func(it works.Item) bool { return it.Label == label }); i >= 0 {
			return scanning.Placed(items[work][i].ID)
		}
		return scanning.Unplaced(scanning.Reason{Code: scanning.NoSuchItem, Names: label})
	}, nil
}

func newWorld(t *testing.T) *world {
	t.Helper()
	ctx := t.Context()
	w := &world{
		lib:     storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema(), Schema()),
		fx:      contenttypetest.New(),
		node:    role.NodeID{UUID: storage.NewUUID()},
		changed: &heard[Changed]{},
		moved:   &heard[files.Changed]{},
	}
	var err error
	w.works, err = works.Open(ctx, works.Config{
		Library: w.lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: w.fx}, Source: w.fx.Source(), Artwork: w.fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Removed: discard[works.Removed]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := rootstest.Config(t, w.lib, contenttypetest.ID)
	cfg.Names = func(context.Context) (map[role.NodeID]nodes.Named, error) {
		return map[role.NodeID]nodes.Named{w.node: {Name: "Media server"}}, nil
	}
	cfg.Describe = w.works.Describe
	if w.roots, err = roots.Open(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	w.root = w.newRoot(t)
	goals, err := targets.Open(ctx, targets.Config{Library: w.lib, Works: w.works, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: w.fx}, Changed: discard[targets.Changed]{}})
	if err != nil {
		t.Fatal(err)
	}
	hd, err := goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{Scores: targets.Scores{Increment: 1}, Name: "HD", Tiers: [][]release.Quality{{release.Bluray1080p}}, Cutoff: release.Bluray1080p})
	if err != nil {
		t.Fatal(err)
	}
	w.hd = hd.ID
	profile, err := goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{Name: "HD", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := goals.MakeDefault(ctx, identity.Admin{}, profile.ID); err != nil {
		t.Fatal(err)
	}
	if w.files, err = files.Open(ctx, files.Config{
		Library: w.lib, Targets: goals, Grabs: grabs.GrabCells{InStep: func(role.NodeID) bool { return true }},
		Naming: map[works.TypeID]naming.Facet{contenttypetest.ID: w.fx}, Names: map[works.TypeID]release.Names{contenttypetest.ID: w.fx}, Today: time.Now,
	}); err != nil {
		t.Fatal(err)
	}
	w.im = New(Config{
		Library: w.lib, Roots: w.roots, Files: w.files, Works: w.works,
		Facets: map[works.TypeID]scanning.Facet{contenttypetest.ID: shelfFacet{}},
		Add: func(ctx context.Context, by identity.Admin, a library.Adding) (library.Added, error) {
			return w.library.Add(ctx, by, a)
		},
		Steps: func(tasks.TaskID) (tasks.RunID, map[tasks.UnitKey]tasks.Step, bool) {
			return 7, w.steps, w.steps != nil
		},
		Changed: w.changed, FilesChanged: w.moved,
	})
	w.library = library.New(library.Config{Library: w.lib, Works: w.works, Roots: w.roots, Files: w.files, Grabs: noGrabs{}, Leaver: w.im})
	return w
}

func (w *world) newRoot(t *testing.T) roots.Root {
	t.Helper()
	v, err := w.roots.Create(t.Context(), identity.Admin{}, roots.NewRoot{ContentType: contenttypetest.ID, Node: w.node, Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	return v.Root
}

// stock puts Work key in the fixture with one box of three clips.
func (w *world) stock(key works.Key) {
	w.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: "Shelf " + string(key), Children: []works.Entry{{Key: "b1", Label: "Box", Children: []works.Entry{
		{Key: "c1", Label: "Clip 1"}, {Key: "c2", Label: "Clip 2"}, {Key: "c3", Label: "Clip 3"},
	}}}}})
}

// add puts Work key in the library, homed in root.
func (w *world) add(t *testing.T, key works.Key, root roots.ID) works.Tree {
	t.Helper()
	w.stock(key)
	added, err := w.library.Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: key, Root: &root})
	if err != nil {
		t.Fatal(err)
	}
	return added.Tree
}

// found is a file a scan read whole.
func found(p string) PendingFile {
	return PendingFile{
		Path: p, Size: 1 << 30, Seen: &storing.Seen{Modified: time.UnixMilli(1_700_000_000_000)},
		Media:   &media.Record{Version: media.Version, Container: "matroska,webm", Video: media.Video{Codec: "hevc", Width: 1920, Height: 1080}},
		Quality: release.Bluray1080p, Cut: release.Standard, Reason: scanning.Reason{Code: scanning.NoMatch},
	}
}

// pend writes what a scan of the world's root found in folder.
func (w *world) pend(t *testing.T, p pend) {
	t.Helper()
	if p.Root.IsZero() {
		p.Root = w.root.ID
	}
	p.ContentType = contenttypetest.ID
	if p.Hint.Title == "" {
		p.Hint = scanning.ReadName(p.Folder)
	}
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return upsertPending(ctx, tx, p, storage.Now())
	})
	if err != nil {
		t.Fatal(err)
	}
}

// entries is what Pending imports lists, the ones that need the Admin or
// the ignored ones.
func (w *world) entries(t *testing.T, ignored bool) []Entry {
	t.Helper()
	l, err := w.im.Pending(t.Context(), identity.Admin{}, Filter{Ignored: ignored})
	if err != nil {
		t.Fatal(err)
	}
	return l.Entries
}

// at is the entry of folder Pending imports lists, needing the Admin or
// ignored, nil when it lists none.
func (w *world) at(t *testing.T, folder string) *Entry {
	t.Helper()
	for _, ignored := range []bool{false, true} {
		for _, e := range w.entries(t, ignored) {
			if e.Folder == folder {
				return &e
			}
		}
	}
	return nil
}

func paths(e *Entry) []string {
	if e == nil {
		return nil
	}
	out := make([]string, len(e.Files))
	for i, f := range e.Files {
		out[i] = f.Path
	}
	return out
}

// filesOf is the paths work's file rows record.
func (w *world) filesOf(t *testing.T, work works.WorkID) []string {
	t.Helper()
	var out []string
	err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		on, err := files.OnRoot(ctx, r, w.root.ID)
		if err != nil {
			return err
		}
		for _, f := range on.Files {
			if f.Work == work {
				out = append(out, f.Path.String())
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// place records a Bluray-1080p file of work at path, on the Item path's
// name labels.
func (w *world) place(t *testing.T, work works.Tree, p string) {
	t.Helper()
	label := strings.TrimSuffix(path.Base(p), path.Ext(p))
	i := slices.IndexFunc(work.Items, func(it works.Item) bool { return it.Label == label })
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{
			WorkID: work.ID, ItemID: work.Items[i].ID, Cut: release.Standard, RootID: w.root.ID, Path: p, Size: 1 << 30, Quality: release.Bluray1080p,
			Media: media.Record{Version: media.Version, Video: media.Video{Codec: "hevc", Width: 1920, Height: 1080}},
			Seen:  &storing.Seen{Modified: time.UnixMilli(1_700_000_000_000)},
		}, []targets.TargetID{w.hd})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
