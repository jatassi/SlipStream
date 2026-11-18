// Package subtitlestest is a library whose Controller subtitles its files
// through a real Store: files, roots and targets over one library.db, the
// Store's SidecarService on a temp root, and speechtest.Fixed for maps.
// Run a World inside synctest.Test, so the Keeper's waits are instant.
package subtitlestest

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
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
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	subtitlesv1 "github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1/subtitlesv1connect"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
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
	t       testing.TB
	Lib     *storage.DB[storage.Library]
	Fixture *contenttypetest.Fixture
	Works   *works.Works
	Targets *targets.Targets
	Root    roots.Root
	Node    role.NodeID
	Files   *files.Files
	Subs    *subtitles.Subtitles
	Timings subtitles.Timings
	// Maps is the Storage Node's Mapper.
	Maps *Maps
	// Writes holds the Keeper's WriteSidecar calls on request.
	Writes *Writes
	// Servers keeps every Sidecar change told to Media Servers.
	Servers *Servers
	// History keeps every entry Subtitles records.
	History *entrytest.Recorder
	history entry.Recorder
	target  targets.Target
	held    *roots.Roots
	// Syncs holds the holder's Sync calls on request.
	Syncs *Syncs
	// Catalog is the holder's Subtitle Providers.
	Catalog *Catalog
	// Cadences is Subtitle search's cadence.
	Cadences *Cadences
	shows    atomic.Bool
	mu       sync.Mutex
	aired    map[works.ItemID]int
	late     *lateSubtitles
	fetchCap int
	stop     func()
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
	Attempts         Published[subtitles.Attempted]
	// Check runs after every step that writes: the records invariant.
	Check func(t testing.TB, w *World)
}

// Open is a World with a Subtitles holder and SubDL's Spec, its Keeper
// and the files owner running until the test ends, on the Controller's
// timings tm, and a holder on ht. Its library holds the secret and
// identity schemas, so an API test can serve over it.
func Open(t *testing.T, tm subtitles.Timings, ht subtitling.Timings) *World {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), subtitles.Schema(), naming.Schema())
	fx := contenttypetest.New()
	tree, err := works.Open(t.Context(), works.Config{
		ItemsAdded: works.NoItemsAdded, Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Source: fx.Source(), Artwork: fx, Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	w := &World{
		t: t, Lib: lib, Fixture: fx, Works: tree, Node: role.NodeID{UUID: storage.NewUUID()}, Timings: tm, sealer: secrettest.Sealer(t),
		Cadences: &Cadences{}, aired: map[works.ItemID]int{},
	}
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
	w.Maps = &Maps{fail: map[string]error{}}
	sidecars, err := storing.NewSidecars(t.Context(), store, storagetest.Local(t, storing.SidecarSchema()), filepath.Join(t.TempDir(), "speechmaps"), w.Maps, storing.DefaultSidecarTimings())
	if err != nil {
		t.Fatal(err)
	}
	w.Writes = &Writes{SidecarCalls: storing.SidecarCallsOf(sidecars)}
	w.Servers = &Servers{}
	w.history, w.History = entrytest.New("subtitles", subtitles.Entries())
	w.Catalog = newCatalog(ht)
	w.Syncs = &Syncs{SubtitlesServiceHandler: w.Catalog}
	w.fetchCap = ht.FetchCap
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
	w.held, w.late = held, late
	w.run(t, w.Files.Run)
	w.start()
	w.Settle()
	return w
}

// start opens the Controller's subtitles and runs them until Restart or
// the test's end.
func (w *World) start() {
	w.Subs = w.Reopen()
	w.late.Subtitles = w.Subs
	w.stop = w.run(w.t, w.Subs.Run)
}

// Restart stops the Controller's subtitles as a crash does, between any
// two steps, and starts a new process over the same library.
func (w *World) Restart() {
	w.t.Helper()
	w.stop()
	w.start()
	w.Settle()
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
		Types:   map[works.TypeID]subtitles.Facet{contenttypetest.ID: episodes{w}},
		Storage: func(role.NodeID) storing.SidecarCalls { return w.Writes },
		Holders: func(role.NodeID) subtitling.Remote {
			return subtitling.NewRemote(keyed{Syncs: w.Syncs, keys: w.Holder})
		},
		Sealer: w.sealer,
		Describe: func(works.TypeID) (works.Description, bool) {
			return contenttypetest.Description, true
		},
		Videos: func(context.Context, storage.Reader[storage.Library]) (media.Videos, error) {
			return media.DefaultVideos, nil
		},
		Providers: Specs,

		MediaServers: w.Servers,
		Works:        w.Works,
		FetchCap:     w.fetchCap,

		Tasks:  w.Cadences,
		GapCap: subtitles.NoGapCap,
		Roster: roster{w},
		Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) {
			return w.restored.Load(), nil
		},
		History:          w.history,
		Changed:          &w.Changed,
		ProvidersChanged: &w.ProvidersChanged,
		Attempt:          &w.Attempts,
		Timings:          w.Timings,
		Clock:            time.Now,
		Rand:             rand.New(&splitmix{x: 2}), //nolint:gosec // seeded on purpose so tests replay
		Log:              slog.New(slog.DiscardHandler),
	}
}

// run runs run until the test ends or the answered stop.
func (w *World) run(t testing.TB, run func(context.Context) error) func() {
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- run(ctx) }()
	stop := sync.OnceFunc(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("run = %v", err)
		}
	})
	t.Cleanup(stop)
	return stop
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
	return w.add(key, p, works.Fetched{Label: "Work " + string(key), Children: []works.Entry{
		{Key: "l1", Label: "Feature", Children: []works.Entry{{Key: "i1", Label: "Feature"}}},
	}})
}

func (w *World) add(key works.Key, p targets.ProfileID, f works.Fetched) works.Tree {
	w.t.Helper()
	w.Fixture.Put(key, contenttypetest.Record{Tree: f})
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
	return w.PlaceItems(tree, []works.ItemID{tree.Items[0].ID}, rel, m)
}

// PlaceItems is Place for a file holding items.
func (w *World) PlaceItems(tree works.Tree, items []works.ItemID, rel string, m media.Record) files.ID {
	w.t.Helper()
	w.Put(rel, "video "+rel)
	var id files.ID
	err := w.Lib.Write(w.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		f, err := files.Add(ctx, tx, files.File{
			WorkID: tree.ID, Items: items, Cut: release.Standard, RootID: w.Root.ID, Path: rel, Size: 1024, Media: m, Quality: release.WEBDL1080p,
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

// Episodes makes the World's files episodes, numbered by Aired. Call it
// before placing any file.
func (w *World) Episodes() { w.shows.Store(true) }

// Aired numbers item as episode n of season 1.
func (w *World) Aired(item works.ItemID, n int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.aired[item] = n
}

// Unair gives item no aired number.
func (w *World) Unair(item works.ItemID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.aired, item)
}

// Show adds a Work of n Items wanting from p, each aired as its episode
// of season 1.
func (w *World) Show(key works.Key, p targets.ProfileID, n int) works.Tree {
	w.t.Helper()
	var items []works.Entry
	for i := range n {
		k := works.Key(fmt.Sprintf("e%d", i+1))
		items = append(items, works.Entry{Key: k, Label: fmt.Sprintf("Episode %d", i+1)})
	}
	tree := w.add(key, p, works.Fetched{Label: "Show " + string(key), Children: []works.Entry{{Key: "s1", Label: "Season 1", Children: items}}})
	for i, it := range tree.Items {
		w.Aired(it.ID, i+1)
	}
	return tree
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

// Speech is where every file's map hears speech: from 1 s to 2 s of 4 s.
var Speech = speechtest.Fixed{Runs: []speechtest.Run{{Start: time.Second, End: 2 * time.Second}}, Length: 4 * time.Second}

// Maps answers Speech's map for every file, unless Fail names an error
// for its base name. While held, each build waits for a Release.
type Maps struct {
	mu      sync.Mutex
	fail    map[string]error
	gate    chan struct{}
	started []string
}

func (m *Maps) Map(ctx context.Context, path string, stream int) (speech.Map, error) {
	name := filepath.Base(path)
	m.mu.Lock()
	m.started = append(m.started, name)
	gate, err := m.gate, m.fail[name]
	m.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return speech.Map{}, fmt.Errorf("the build: %w", context.Cause(ctx))
		}
	}
	if err != nil {
		return speech.Map{}, err
	}
	return Speech.Map(ctx, path, stream)
}

// Fail makes the builds of the file named name fail with err, nil for
// none.
func (m *Maps) Fail(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fail[name] = err
}

// Hold makes every build wait for a Release, until Unhold.
func (m *Maps) Hold() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gate = make(chan struct{})
}

// Release lets one waiting build finish.
func (m *Maps) Release() { m.gate <- struct{}{} }

// Unhold lets every build finish.
func (m *Maps) Unhold() {
	m.mu.Lock()
	defer m.mu.Unlock()
	close(m.gate)
	m.gate = nil
}

// Started is the base names of the files whose builds started, in order.
func (m *Maps) Started() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.started)
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

// Writes passes each SidecarService call through, and while held stops
// each WriteSidecar until Let decides it.
type Writes struct {
	storing.SidecarCalls

	mu      sync.Mutex
	held    bool
	waiting chan chan let
}

// Hold stops the next WriteSidecar calls until Let.
func (h *Writes) Hold() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.held, h.waiting = true, make(chan chan let, 1)
}

// Let ends the held WriteSidecar call: land writes its bytes, and answer
// returns the Node's answer, else the call answers as lost, as when the
// Controller dies with it in flight. It holds no more calls after.
func (h *Writes) Let(land, answer bool) {
	h.mu.Lock()
	waiting := h.waiting
	h.held = false
	h.mu.Unlock()
	(<-waiting) <- let{land: land, answer: answer}
}

type let struct{ land, answer bool }

// errLost is a call whose answer never came back.
var errLost = errors.New("the call was lost")

func (h *Writes) Write(ctx context.Context, sw storing.SidecarWrite) (storing.Rel, error) {
	h.mu.Lock()
	held, waiting := h.held, h.waiting
	h.mu.Unlock()
	if !held {
		return h.SidecarCalls.Write(ctx, sw)
	}
	decide := make(chan let, 1)
	select {
	case waiting <- decide:
	case <-ctx.Done():
		return storing.Rel{}, fmt.Errorf("the write: %w", context.Cause(ctx))
	}
	select {
	case l := <-decide:
		if !l.land {
			return storing.Rel{}, errLost
		}
		at, err := h.SidecarCalls.Write(ctx, sw)
		if l.answer || err != nil {
			return at, err
		}
		return storing.Rel{}, errLost
	case <-ctx.Done():
		return storing.Rel{}, fmt.Errorf("the write: %w", context.Cause(ctx))
	}
}

// Syncs passes each SubtitlesService call through, and while held stops
// each Sync until Release.
type Syncs struct {
	subtitlesv1connect.SubtitlesServiceHandler

	mu   sync.Mutex
	gate chan struct{}
}

// Hold stops the next Sync calls until Release.
func (h *Syncs) Hold() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.gate = make(chan struct{})
}

// Release lets the held Sync calls answer, and holds no more.
func (h *Syncs) Release() {
	h.mu.Lock()
	defer h.mu.Unlock()
	close(h.gate)
	h.gate = nil
}

func (h *Syncs) Sync(ctx context.Context, req *subtitlesv1.SyncRequest) (*subtitlesv1.SyncResponse, error) {
	h.mu.Lock()
	gate := h.gate
	h.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return nil, fmt.Errorf("the sync: %w", context.Cause(ctx))
		}
	}
	return h.SubtitlesServiceHandler.Sync(ctx, req)
}

// Servers keeps the Sidecar changes told to Media Servers, in order.
type Servers struct {
	mu      sync.Mutex
	changes []mediaservers.SidecarChange
}

func (s *Servers) SidecarsChanged(_ context.Context, _ *storage.Tx[storage.Library], cs []mediaservers.SidecarChange) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changes = append(s.changes, cs...)
	return nil
}

// Changes is every change told so far.
func (s *Servers) Changes() []mediaservers.SidecarChange {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.changes)
}

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

// keyed answers key checks with keys when a test set it, and every other
// call through Syncs.
type keyed struct {
	*Syncs
	keys subtitlesv1connect.SubtitlesServiceHandler
}

func (k keyed) CheckKey(ctx context.Context, req *subtitlesv1.CheckKeyRequest) (*subtitlesv1.CheckKeyResponse, error) {
	if k.keys == nil {
		return k.Syncs.CheckKey(ctx, req)
	}
	return k.keys.CheckKey(ctx, req)
}

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
			Provider: req.GetProvider(), Kind: subtitlesv1.OutcomeKind_OUTCOME_KIND_REFUSED, Words: "refused the key: " + h.Refused, AtUnixMicro: at,
		}}}, nil
	}
	return &subtitlesv1.CheckKeyResponse{Account: h.Account, Plan: h.Plan, Outcomes: []*subtitlesv1.Outcome{{
		Provider: req.GetProvider(), Kind: subtitlesv1.OutcomeKind_OUTCOME_KIND_OK, AtUnixMicro: at,
		Allowance: &subtitlesv1.Allowance{Downloads: &subtitlesv1.Window{Used: 31, Limit: 50, Remaining: 19, ResetsUnixMicro: at + 3_600_000_000, Exact: true}},
	}}}, nil
}
