// Package subtitlestest is a library whose Controller subtitles its files
// through a real Store: files, roots and targets over one library.db, the
// Store's SidecarService on a temp root, and speechtest.Fixed for maps.
// Run a World inside synctest.Test, so the Keeper's waits are instant.
package subtitlestest

import (
	"context"
	"crypto/sha256"
	"errors"
	"io/fs"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	subtitlesv1 "github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1/subtitlesv1connect"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// World is one library, its one root on Node, and the Controller's
// subtitles over it.
type World struct {
	t        testing.TB
	Lib      *storage.DB[storage.Library]
	Fixture  *contenttypetest.Fixture
	Works    *works.Works
	Targets  *targets.Targets
	Root     roots.Root
	Node     role.NodeID
	Files    *files.Files
	Subs     *subtitles.Subtitles
	Timings  subtitles.Timings
	target   targets.Target
	held     *roots.Roots
	calls    storing.SidecarCalls
	holders  atomic.Bool
	restored atomic.Bool
	sealer   *secret.Sealer
	// Holder is SubtitlesService on the World's Subtitles holder, the
	// Node too. A test that checks a key sets it.
	Holder subtitlesv1connect.SubtitlesServiceHandler
	// Changed and ProvidersChanged are what the World's subtitles
	// published.
	Changed          Published[subtitles.Changed]
	ProvidersChanged Published[subtitles.ProvidersChanged]
	// Check runs after every step that writes: the records invariant.
	Check func(t testing.TB, w *World)
}

// Open is a World with a Subtitles holder and SubDL's Spec, its Keeper
// and the files owner running until the test ends, on the Controller's
// timings tm.
func Open(t *testing.T, tm subtitles.Timings) *World {
	t.Helper()
	lib := storagetest.Library(t, identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), subtitles.Schema(), naming.Schema())
	fx := contenttypetest.New()
	tree, err := works.Open(t.Context(), works.Config{
		ItemsAdded: works.NoItemsAdded, Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Source: fx.Source(), Artwork: fx, Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	w := &World{t: t, Lib: lib, Fixture: fx, Works: tree, Node: role.NodeID{UUID: storage.NewUUID()}, Timings: tm, sealer: secrettest.Sealer(t)}
	w.holders.Store(true)
	if err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, w.Node, "Node A")
	}); err != nil {
		t.Fatal(err)
	}
	w.Root = rootstest.Root(t, lib, w.Node, contenttypetest.ID, t.TempDir())
	var store *storing.Store
	held := rootstest.HeldOn(t, lib, w.Node, rootstest.On{Tune: func(cfg *roots.Config) {
		store, _ = cfg.Remote(w.Node).(*storing.Store)
	}}, contenttypetest.ID)
	maps := speechtest.Fixed{Runs: []speechtest.Run{{Start: time.Second, End: 2 * time.Second}}, Length: 4 * time.Second}
	sidecars, err := storing.NewSidecars(t.Context(), store, storagetest.Local(t, storing.SidecarSchema()), filepath.Join(t.TempDir(), "speechmaps"), maps, storing.DefaultSidecarTimings())
	if err != nil {
		t.Fatal(err)
	}
	w.calls = storing.SidecarCallsOf(sidecars)
	if w.Targets, err = targets.Open(t.Context(), targets.Config{
		Library: lib, Works: tree, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{},
		Subtitled: func(works.TypeID) bool { return true },
	}); err != nil {
		t.Fatal(err)
	}
	if w.target, err = w.Targets.CreateTarget(t.Context(), identity.Admin{}, targets.Spec{
		Scores: targets.Scores{Increment: 1}, Name: "HD", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p,
	}); err != nil {
		t.Fatal(err)
	}
	late := &lateSubtitles{Subtitles: files.NoSubtitles{}}
	w.Files, err = files.Open(t.Context(), files.Config{
		Library: lib, Targets: w.Targets, Grabs: files.NoGrabs{},
		Transfers: func(context.Context, storage.Reader[storage.Library]) (map[transfers.ID]bool, error) { return nil, nil },
		Naming: naming.New(naming.Config{
			Library: lib, Facets: map[works.TypeID]naming.Facet{contenttypetest.ID: fx}, Describe: tree.Describe, Budget: roots.SmallestBudget,
		}),
		Names: map[works.TypeID]release.Names{contenttypetest.ID: fx},
		Today: time.Now, Disk: held, Changed: discard[files.Changed]{}, Moved: discard[files.Moved]{}, Subtitles: late,
		Timings: files.Timings{RetryMin: time.Second, RetryMax: 4 * time.Second},
		Rand:    rand.New(&splitmix{x: 1}), Log: slog.New(slog.DiscardHandler), //nolint:gosec // seeded on purpose so tests replay
	})
	if err != nil {
		t.Fatal(err)
	}
	w.held = held
	w.Subs = w.Reopen()
	late.Subtitles = w.Subs
	w.run(t, w.Files.Run)
	w.run(t, w.Subs.Run)
	w.Settle()
	return w
}

// Reopen opens the Controller's subtitles again over the same library, as
// a new process does, without running it.
func (w *World) Reopen() *subtitles.Subtitles {
	w.t.Helper()
	s, err := subtitles.Open(w.t.Context(), w.config())
	if err != nil {
		w.t.Fatal(err)
	}
	return s
}

func (w *World) config() subtitles.Config {
	return subtitles.Config{
		Library: w.Lib, Files: w.Files, Targets: w.Targets, Roots: w.held,
		Types:   map[works.TypeID]subtitles.Facet{contenttypetest.ID: w.Fixture},
		Storage: func(role.NodeID) storing.SidecarCalls { return w.calls },
		Holders: func(role.NodeID) subtitling.Remote { return subtitling.NewRemote(w.Holder) },
		Sealer:  w.sealer,
		Describe: func(works.TypeID) (works.Description, bool) {
			return contenttypetest.Description, true
		},
		Videos: func(context.Context, storage.Reader[storage.Library]) (media.Videos, error) {
			return media.DefaultVideos, nil
		},
		Providers: []subtitling.Spec{
			{ID: "subdl", Label: "SubDL", Kinds: []subtitling.Kind{subtitling.Feature, subtitling.Episode}, Keyed: true, Position: 1},
			{ID: "gestdown", Label: "Gestdown", Kinds: []subtitling.Kind{subtitling.Episode}, Position: 2},
		},
		Tasks:     subtitles.Unscheduled{},
		Roster:    roster{w},
		Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) {
			return w.restored.Load(), nil
		},
		Changed:          &w.Changed,
		ProvidersChanged: &w.ProvidersChanged,
		Timings:          w.Timings,
		Clock:            time.Now,
		Rand:    rand.New(&splitmix{x: 2}), //nolint:gosec // seeded on purpose so tests replay
		Log:     slog.New(slog.DiscardHandler),
	}
}

func (w *World) run(t *testing.T, run func(context.Context) error) {
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- run(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("run = %v", err)
		}
	})
}

// Settle waits out the Keeper's wakes and the passes they start, then
// checks the records.
func (w *World) Settle() {
	w.t.Helper()
	for range 4 {
		time.Sleep(w.Timings.WakeDelay)
		synctest.Wait()
	}
	if w.Check != nil {
		w.Check(w.t, w)
	}
}

// Sweep lets a Keeper poll pass, which wakes what the rows hold.
func (w *World) Sweep() {
	w.t.Helper()
	time.Sleep(w.Timings.KeeperPoll)
	w.Settle()
}

// Restoring sets whether the Controller reads as Reconciling.
func (w *World) Restoring(on bool) { w.restored.Store(on) }

// Holders sets whether a Node holds the Subtitles Role.
func (w *World) Holders(on bool) { w.holders.Store(on) }

// Profile creates a Profile of the HD Target wanting langs.
func (w *World) Profile(name string, langs ...targets.WantedLanguage) targets.ProfileID {
	w.t.Helper()
	p, err := w.Targets.CreateProfile(w.t.Context(), identity.Admin{}, targets.ProfileSpec{
		Name: name, ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: w.target.ID, Sourcing: targets.Download}}, Languages: langs,
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return p.ID
}

// Languages saves p's Wanted Languages through files, as a Profile PATCH
// does, and answers its Impact.
func (w *World) Languages(p targets.ProfileID, name string, langs ...targets.WantedLanguage) files.Impact {
	w.t.Helper()
	spec := targets.ProfileSpec{Name: name, ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: w.target.ID, Sourcing: targets.Download}}, Languages: langs}
	i, err := w.Files.Change(w.t.Context(), identity.Admin{}, targets.UpdateProfile{ID: p, Spec: spec}, files.Mode{Unwanted: files.Keep})
	if err != nil {
		w.t.Fatal(err)
	}
	w.Settle()
	return i
}

// Preview is the Impact a save of langs would have.
func (w *World) Preview(p targets.ProfileID, name string, langs ...targets.WantedLanguage) files.Impact {
	w.t.Helper()
	spec := targets.ProfileSpec{Name: name, ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: w.target.ID, Sourcing: targets.Download}}, Languages: langs}
	i, err := w.Files.Change(w.t.Context(), identity.Admin{}, targets.UpdateProfile{ID: p, Spec: spec}, files.Mode{Unwanted: files.Keep, Preview: true})
	if err != nil {
		w.t.Fatal(err)
	}
	return i
}

// Work adds a Work of one Item wanting from p.
func (w *World) Work(key works.Key, p targets.ProfileID) works.Tree {
	w.t.Helper()
	w.Fixture.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: "Work " + string(key), Children: []works.Entry{
		{Key: "l1", Label: "Feature", Children: []works.Entry{{Key: "i1", Label: "Feature"}}},
	}}})
	tree, _, err := w.Works.Add(w.t.Context(), identity.Admin{}, contenttypetest.ID, key, works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], s works.Summary) error {
			if err := targets.Assign(ctx, tx, s, &p); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, s, nil)
		},
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return tree
}

// Place adds a library file of tree's Item at rel with m, its bytes on
// disk, and runs the bulk pass over it, as a Library scan's adoption does.
func (w *World) Place(tree works.Tree, rel string, m media.Record) files.ID {
	w.t.Helper()
	w.Put(rel, "video "+rel)
	var id files.ID
	err := w.Lib.Write(w.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		f, err := files.Add(ctx, tx, files.File{
			WorkID: tree.ID, Items: []works.ItemID{tree.Items[0].ID}, Cut: release.Standard, RootID: w.Root.ID, Path: rel, Size: 1024, Media: m, Quality: release.WEBDL1080p,
		}, []targets.TargetID{w.target.ID})
		id = f.ID
		return err
	})
	if err != nil {
		w.t.Fatal(err)
	}
	w.Reconcile()
	return id
}

// Add adds a library file of tree's Item at rel with m in tx, filling
// nothing, with no bytes and no hook.
func (w *World) Add(ctx context.Context, tx *storage.Tx[storage.Library], tree works.Tree, rel string, m media.Record) (files.ID, error) {
	f, err := files.Add(ctx, tx, files.File{
		WorkID: tree.ID, Items: []works.ItemID{tree.Items[0].ID}, Cut: release.Standard, RootID: w.Root.ID, Path: rel, Size: 1024, Media: m, Quality: release.WEBDL1080p,
	}, nil)
	return f.ID, err
}

// Reconcile runs the bulk pass and settles.
func (w *World) Reconcile() subtitles.Reconciled {
	w.t.Helper()
	got, err := w.Subs.Reconcile(w.t.Context())
	if err != nil {
		w.t.Fatal(err)
	}
	w.Settle()
	return got
}

// Put writes content at rel under the root.
func (w *World) Put(rel, content string) [32]byte {
	w.t.Helper()
	at := filepath.Join(w.Root.Path, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte(content), 0o600); err != nil {
		w.t.Fatal(err)
	}
	return sha256.Sum256([]byte(content))
}

// On is what rel holds under the root, "" when nothing.
func (w *World) On(rel string) string {
	w.t.Helper()
	b, err := os.ReadFile(filepath.Join(w.Root.Path, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	if err != nil {
		w.t.Fatal(err)
	}
	return string(b)
}

// Write runs fn in one library Write and settles.
func (w *World) Write(fn func(ctx context.Context, tx *storage.Tx[storage.Library]) error) {
	w.t.Helper()
	if err := w.Lib.Write(w.t.Context(), fn); err != nil {
		w.t.Fatal(err)
	}
	w.Settle()
}

// Doom deletes tree's files as a Work delete does.
func (w *World) Doom(tree works.Tree) {
	w.t.Helper()
	w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return w.Files.DoomIn(ctx, tx, []works.WorkID{tree.ID})
	})
}

// Wanted is a Wanted Language of tag, its HI and audio rule.
func Wanted(t testing.TB, tag string, forced bool, hi targets.HI, audio targets.AudioCondition) targets.WantedLanguage {
	t.Helper()
	l, err := language.Parse(tag)
	if err != nil {
		t.Fatal(err)
	}
	return targets.WantedLanguage{Language: l, Forced: forced, HI: hi, Audio: audio}
}

type roster struct{ w *World }

func (r roster) Holding(role.Name) []role.NodeID {
	if r.w.holders.Load() {
		return []role.NodeID{r.w.Node}
	}
	return nil
}

func (r roster) Name(id role.NodeID) string {
	if id == r.w.Node {
		return "Node A"
	}
	return id.String()
}

// Published is every value of one event a World published, in order.
type Published[T any] struct {
	mu sync.Mutex
	vs []T
}

func (p *Published[T]) Publish(_ context.Context, v T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.vs = append(p.vs, v)
}

// All is what was published so far.
func (p *Published[T]) All() []T {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.vs)
}

type lateSubtitles struct{ files.Subtitles }

// splitmix is SplitMix64, a replayable stream for the World, which has no
// Node to hand it one. It is safe for concurrent draws.
type splitmix struct {
	mu sync.Mutex
	x  uint64
}

func (s *splitmix) Uint64() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.x += 0x9e3779b97f4a7c15
	z := s.x
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

// KeyHolder is a Subtitles holder whose key checks answer Account and
// Plan with 19 of 50 downloads left, or the provider's refusal Refused
// when it is set.
type KeyHolder struct {
	subtitlesv1connect.UnimplementedSubtitlesServiceHandler
	Account, Plan, Refused string
}

func (h KeyHolder) CheckKey(_ context.Context, req *subtitlesv1.CheckKeyRequest) (*subtitlesv1.CheckKeyResponse, error) {
	at := time.Now().UnixMicro()
	if h.Refused != "" {
		return &subtitlesv1.CheckKeyResponse{Outcomes: []*subtitlesv1.Outcome{{
			Provider: req.GetProvider(), Kind: subtitlesv1.OutcomeKind_OUTCOME_KIND_REFUSED, Words: h.Refused, AtUnixMicro: at,
		}}}, nil
	}
	return &subtitlesv1.CheckKeyResponse{Account: h.Account, Plan: h.Plan, Outcomes: []*subtitlesv1.Outcome{{
		Provider: req.GetProvider(), Kind: subtitlesv1.OutcomeKind_OUTCOME_KIND_OK, AtUnixMicro: at,
		Allowance: &subtitlesv1.Allowance{Downloads: &subtitlesv1.Window{Used: 31, Limit: 50, Remaining: 19, ResetsUnixMicro: at + 3_600_000_000, Exact: true}},
	}}}, nil
}
