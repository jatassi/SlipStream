package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"path"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
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
	mediav1 "github.com/jatassi/slipstream/internal/pb/slipstream/media/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/media/v1/mediav1connect"
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
	cfg     Config
	root    roots.Root
	node    role.NodeID
	hd      targets.TargetID
	changed *heard[Changed]
	moved   *heard[files.Changed]
	steps   map[tasks.UnitKey]tasks.Step
	// stop, when set, sees each Import file's IO before it runs.
	stop  atomic.Pointer[func(context.Context, Step)]
	probe media.Record
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
		lib:     storagetest.Library(t, identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema(), naming.Schema(), Schema()),
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
		Naming: naming.New(naming.Config{
			Facets:   map[works.TypeID]naming.Facet{contenttypetest.ID: w.fx},
			Describe: func(works.TypeID) (works.Description, bool) { return contenttypetest.Description, true },
			Budget:   func(context.Context, storage.Reader[storage.Library], works.TypeID) (int, error) { return 1000, nil },
		}),
		Names: map[works.TypeID]release.Names{contenttypetest.ID: w.fx}, Today: time.Now,
	}); err != nil {
		t.Fatal(err)
	}
	w.probe = media.Record{Version: media.Version, Container: "matroska,webm", Video: media.Video{Codec: "hevc", Width: 1920, Height: 1080}}
	w.cfg = Config{
		Library: w.lib, Roots: w.roots, Files: w.files, Works: w.works, Targets: goals, Probes: w.probes,
		Facets: map[works.TypeID]scanning.Facet{contenttypetest.ID: shelfFacet{}},
		Add: func(ctx context.Context, by identity.Admin, a library.Adding) (library.Added, error) {
			return w.library.Add(ctx, by, a)
		},
		Steps: func(tasks.TaskID) (tasks.RunID, map[tasks.UnitKey]tasks.Step, bool) {
			return 7, w.steps, w.steps != nil
		},
		Changed: w.changed, FilesChanged: w.moved,
		Timings: Timings{RetryMin: time.Millisecond, RetryMax: 20 * time.Millisecond}, Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // a fixed draw the test replays
		BeforeImport: func(ctx context.Context, s Step) {
			if stop := w.stop.Load(); stop != nil {
				(*stop)(ctx, s)
			}
		},
		Log: slog.New(slog.DiscardHandler),
	}
	w.im = New(w.cfg)
	w.library = library.New(library.Config{Library: w.lib, Works: w.works, Roots: w.roots, Files: w.files, Grabs: noGrabs{}, Imports: w.im, Leaver: w.im})
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
func found(p string) File {
	f := File{
		Path: p, Size: 1 << 30, Seen: &storing.Seen{Modified: time.UnixMilli(1_700_000_000_000)},
		Probe:   &media.Record{Version: media.Version, Container: "matroska,webm", Video: media.Video{Codec: "hevc", Width: 1920, Height: 1080}},
		Quality: release.Bluray1080p, Cut: release.Standard, Position: -1,
	}
	f.waitFor(scanning.Reason{Code: scanning.NoMatch})
	return f
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
func (w *world) entries(t *testing.T, ignored bool) []Import {
	t.Helper()
	l, err := w.im.List(t.Context(), identity.Admin{}, Filter{Ignored: ignored})
	if err != nil {
		t.Fatal(err)
	}
	return l.Imports
}

// at is the entry of folder Pending imports lists, needing the Admin or
// ignored, nil when it lists none.
func (w *world) at(t *testing.T, folder string) *Import {
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

// paths is the paths of e's waiting files, the ones Pending imports
// lists, in path order.
func paths(e *Import) []string {
	if e == nil {
		return nil
	}
	var out []string
	for _, f := range e.waiting() {
		out = append(out, f.Path)
	}
	slices.Sort(out)
	return out
}

// position is the position of e's waiting file at p, -1 for none.
func position(e *Import, p string) int {
	for _, f := range e.waiting() {
		if f.Path == p {
			return f.Position
		}
	}
	return -1
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
			WorkID: work.ID, Items: []works.ItemID{work.Items[i].ID}, Cut: release.Standard, RootID: w.root.ID, Path: p, Size: 1 << 30, Quality: release.Bluray1080p,
			Media: media.Record{Version: media.Version, Video: media.Video{Codec: "hevc", Width: 1920, Height: 1080}},
			Seen:  &storing.Seen{Modified: time.UnixMilli(1_700_000_000_000)},
		}, []targets.TargetID{w.hd})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// itemsOfFile is the Items work's file id holds.
func (w *world) itemsOfFile(t *testing.T, work works.WorkID, id files.ID) []works.ItemID {
	t.Helper()
	l, err := w.files.WorkFiles(t.Context(), identity.Admin{}, work)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range l.Files {
		if f.ID == id {
			return f.Items
		}
	}
	return nil
}

// stored is Import id as its rows read.
func (w *world) stored(t *testing.T, id ImportID) *Import {
	t.Helper()
	imp, err := importOf(t.Context(), w.lib, id)
	if err != nil {
		t.Fatal(err)
	}
	return &imp
}

// probes is MediaService on every Node, answering w.probe for any file.
func (w *world) probes(role.NodeID) mediav1connect.MediaServiceHandler { return prober{w} }

type prober struct{ w *world }

func (p prober) Probe(context.Context, *mediav1.ProbeRequest) (*mediav1.ProbeResponse, error) {
	b, err := json.Marshal(p.w.probe)
	if err != nil {
		return nil, fmt.Errorf("encode the probe: %w", err)
	}
	return &mediav1.ProbeResponse{Record: b}, nil
}
