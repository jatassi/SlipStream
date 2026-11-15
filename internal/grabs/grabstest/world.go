package grabstest

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// Schemas is every schema a Grabs' library holds.
func Schemas() []storage.Schema[storage.Library] {
	return []storage.Schema[storage.Library]{
		secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), health.Schema(), roots.Schema(),
		downloadclients.Schema(), grabs.Schema(), indexers.Schema(), trackers.Schema(), files.Schema(), transfers.LibrarySchema(), imports.Schema(),
	}
}

// Options is what a World holds besides The Attic.
type Options struct {
	// Sites names an Indexer each, homed on the World's Node.
	Sites []string
	// Goals is the Work's Targets in Profile order, HD alone when nil.
	Goals []targets.Spec
	// Season is how many Items the Work's one level holds, one when 0:
	// the episodes E01, E02 and on, which Pack names.
	Season int
	// Remote answers the Indexers' jobs, as their Searching Node.
	Remote searching.Remote
	// Log is the Grabs' log, discarded when nil.
	Log *slog.Logger
	// Grabs, Indexers and Formats are the owners' Timings, as a Node's
	// Config holds them.
	Grabs    grabs.Timings
	Indexers indexers.Timings
	Formats  formats.Timings
}

// HD is The Attic's Target when Options name none: Bluray-1080p over
// WEB-DL 1080p.
var HD = []targets.Spec{{
	Scores: targets.Scores{Increment: 1}, Name: "HD", Cutoff: rel.Bluray1080p,
	Tiers: [][]rel.Quality{{rel.Bluray1080p}, {rel.WEBDL1080p}},
}}

// World is a library holding The Attic, one Item monitored, rooted and
// wanting its goals, with an Indexer per site whose jobs Remote answers,
// and a Grabs over it. Make it inside a testing/synctest bubble, so the
// Grabs' clock is the bubble's.
type World struct {
	Lib *storage.DB[storage.Library]
	// Grabs is the Grabs opened last over Lib.
	Grabs *grabs.Grabs
	// Config is what Restart opens a Grabs with.
	Config   grabs.Config
	Indexers *indexers.Indexers
	Work     works.WorkID
	Item     works.ItemID
	// Items are the level's Items in order, Item first.
	Items []works.ItemID
	// Level is the Item's level.
	Level works.LevelID
	// Home is the one Node, which holds every Role.
	Home role.NodeID
	// Sites is each site's Indexer, by name.
	Sites map[string]indexers.ID

	t           testing.TB
	reconciling atomic.Bool
	stop        func()
}

// New is a World with o.
func New(t testing.TB, o Options) *World {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, Schemas()...)
	w := &World{Lib: lib, Sites: map[string]indexers.ID{}, t: t}
	home := role.NodeID{UUID: storage.NewUUID()}
	w.Home = home
	sealer := secrettest.Sealer(t)
	fx := contenttypetest.New()
	tree, err := works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Removed: discard[works.Removed]{},
	})
	must(t, err)
	wanted, err := targets.Open(ctx, targets.Config{
		Library: lib, Works: tree, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{}, Formats: o.Formats,
	})
	must(t, err)
	goals := o.Goals
	if goals == nil {
		goals = HD
	}
	var entries []targets.Entry
	for _, spec := range goals {
		made, err := wanted.CreateTarget(ctx, identity.Admin{}, spec)
		must(t, err)
		entries = append(entries, targets.Entry{Target: made.ID, Sourcing: targets.Download})
	}
	profile, err := wanted.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{Name: "Standard", ContentType: contenttypetest.ID, Entries: entries})
	must(t, err)
	_, err = wanted.MakeDefault(ctx, identity.Admin{}, profile.ID)
	must(t, err)
	episodes := []works.Entry{{Key: "c1", Label: "Scarf"}}
	for i := 2; i <= o.Season; i++ {
		episodes = append(episodes, works.Entry{Key: works.Key("c" + strconv.Itoa(i)), Label: episode(i)})
	}
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: episodes},
	}}})
	added, _, err := tree.Add(ctx, identity.Admin{}, contenttypetest.ID, "w1", works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, nil); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	must(t, err)
	w.Work, w.Item, w.Level = added.ID, added.Items[0].ID, added.Items[0].LevelID
	for _, it := range added.Items {
		w.Items = append(w.Items, it.ID)
	}
	library, err := roots.Open(ctx, rootstest.Config(t, lib, contenttypetest.ID))
	must(t, err)
	r := alone{node: home}
	router := downloadclients.Router{Roster: r, Routes: trackers.Routing{Roster: r}}
	shelf, err := files.Open(ctx, files.Config{Library: lib, Targets: wanted, Grabs: imports.Cells{Grabs: grabs.GrabCells{InStep: r.InStep, Router: router}}, Today: time.Now})
	must(t, err)
	terms := attic{items: w.Items}
	w.Indexers = indexers.New(indexers.Config{
		Library: lib, Sealer: sealer, Roster: indexerstest.Alone(home), Searching: func(role.NodeID) searching.Remote { return o.Remote },
		Types: map[works.TypeID]indexers.Facet{contenttypetest.ID: terms}, Works: tree,
		Changed: discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Trackers: discard[indexers.TrackerChanged]{},
		Timings: o.Indexers, Log: slog.New(slog.DiscardHandler),
	})
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, home, "home"); err != nil {
			return err
		}
		for _, site := range o.Sites {
			id := indexers.ID{UUID: storage.NewUUID()}
			w.Sites[site] = id
			if err := indexerstest.Torznab(ctx, tx, sealer, id, site, home, "https://site.test/api", "k"); err != nil {
				return err
			}
		}
		return nil
	})
	rootstest.Root(t, lib, home, contenttypetest.ID, t.TempDir())
	log := o.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	w.Config = grabs.Config{
		Library: lib, Sealer: sealer, Works: tree, Targets: wanted, Files: shelf, Indexers: w.Indexers,
		Terms: map[works.TypeID]indexers.Facet{contenttypetest.ID: terms}, Names: map[works.TypeID]rel.Names{contenttypetest.ID: Dated{}},
		Roster: r, Downloads: noDownloads{}, Roots: library, Router: router, Imports: imports.GrabImports{},
		Timings: o.Grabs, Rand: rand.New(&splitmix{x: 1}), //nolint:gosec // seeded on purpose so tests replay
		Changed: discard[grabs.Changed]{}, SearchChanged: discard[grabs.SearchChanged]{}, FilesChanged: discard[files.Changed]{},
		BlocklistChanged: discard[grabs.BlocklistChanged]{}, QueueChanged: discard[grabs.QueueChanged]{},
		Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) { return w.reconciling.Load(), nil },
		GapCap:      grabs.NoGapCap, Clock: time.Now, Log: log,
	}
	w.Grabs = grabs.New(w.Config)
	t.Cleanup(w.halt)
	return w
}

// Reconciling sets whether the Controller is Reconciling after a restore.
func (w *World) Reconciling(on bool) { w.reconciling.Store(on) }

// Run runs the Grabs' reconciler until the test ends or Restart, and
// waits for it to settle.
func (w *World) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	g := w.Grabs
	go func() { done <- g.Run(ctx) }()
	w.stop = func() {
		cancel()
		if err := <-done; err != nil {
			w.t.Error(err)
		}
	}
	synctest.Wait()
}

func (w *World) halt() {
	if w.stop != nil {
		w.stop()
		w.stop = nil
	}
}

// Restart opens a new Grabs over the same library, as a Controller that
// starts again does, and runs it when the last one ran.
func (w *World) Restart() {
	ran := w.stop != nil
	w.halt()
	w.Grabs = grabs.New(w.Config)
	if ran {
		w.Run()
	}
}

// Queue is what the search queue holds, as one value a test compares:
// each Work's queue rows, oldest first, and its open automatic runs with
// their lines.
type Queue struct {
	Due  []Due
	Runs []Run
}

// Due is one queue row. Lane zero is not queued.
type Due struct {
	Work    works.WorkID
	Group   indexers.GroupKey
	Lane    pace.Lane
	Trigger searches.Trigger
	// Items nil is the whole group.
	Items []works.ItemID
	Gone  bool
	// Searching is whether the group has an open run.
	Searching bool
}

// Run is an open automatic run.
type Run struct {
	Work    works.WorkID
	Group   indexers.GroupKey
	Lane    pace.Lane
	Trigger searches.Trigger
	Gone    bool
	Phase   int
	Items   []works.ItemID
	Lines   []Line
}

// Line is one of a run's lines: Indexer by name, Item zero for the
// group's query, and State "planned", "sent" or its outcome.
type Line struct {
	Indexer string
	Item    works.ItemID
	State   string
}

// Queue reads what the queue holds now.
func (w *World) Queue(t testing.TB) Queue {
	t.Helper()
	synctest.Wait()
	var q Queue
	must(t, w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		q, err = queueIn(ctx, r)
		return err
	}))
	return q
}

func queueIn(ctx context.Context, r storage.Reader[storage.Library]) (Queue, error) {
	dq := db.New(r)
	var q Queue
	due, err := dq.ListDue(ctx)
	if err != nil {
		return Queue{}, err
	}
	for _, row := range due {
		items, err := dq.ListDueItems(ctx, db.ListDueItemsParams{WorkID: row.WorkID, GroupKey: row.GroupKey})
		if err != nil {
			return Queue{}, err
		}
		q.Due = append(q.Due, Due{
			Work: row.WorkID, Group: row.GroupKey, Lane: pace.Lane(deref(row.Lane)), Trigger: searches.Trigger(deref(row.Trigger)),
			Items: items, Gone: row.IncludeGone, Searching: !row.SearchID.IsZero(),
		})
	}
	open, err := dq.ListOpenSearches(ctx)
	if err != nil {
		return Queue{}, err
	}
	for _, row := range open {
		if row.GroupKey == nil {
			continue
		}
		run := Run{
			Work: row.WorkID, Group: *row.GroupKey, Lane: pace.Lane(deref(row.Lane)), Trigger: searches.Trigger(deref(row.Trigger)),
			Gone: row.IncludeGone, Phase: int(row.Phase),
		}
		items, err := dq.ListSearchItems(ctx, []db.SearchID{row.ID})
		if err != nil {
			return Queue{}, err
		}
		for _, i := range items {
			run.Items = append(run.Items, i.ItemID)
		}
		lines, err := dq.ListSearchIndexersOfSearch(ctx, row.ID)
		if err != nil {
			return Queue{}, err
		}
		for _, l := range lines {
			run.Lines = append(run.Lines, Line{Indexer: l.IndexerName, Item: l.ItemID, State: stateOf(l)})
		}
		q.Runs = append(q.Runs, run)
	}
	return q, nil
}

func stateOf(l db.GrabsSearchIndexer) string {
	switch {
	case l.Outcome != nil:
		return string(*l.Outcome)
	case l.QueryID.IsZero():
		return "planned"
	}
	return "sent"
}

func deref[T any](p *T) T {
	var v T
	if p != nil {
		v = *p
	}
	return v
}

// attic is The Attic's search terms: its label, and the Items a name's
// mark covers, its first Item for a name with none.
type attic struct{ items []works.ItemID }

func (a attic) Expect(context.Context, storage.Reader[storage.Library], works.WorkID) (indexers.Expect, error) {
	covers := func(n rel.Name) []works.ItemID {
		if len(n.Marks) == 0 {
			return a.items[:1]
		}
		from, to, _ := strings.Cut(n.Marks[0].Key, "-")
		if to == "" {
			to = from
		}
		first, _ := strconv.Atoi(from[1:])
		last, _ := strconv.Atoi(to[1:])
		return a.items[first-1 : min(last, len(a.items))]
	}
	return indexers.Expect{Claim: rel.Claim{Titles: []string{"The Attic"}}, Covers: covers}, nil
}

func (a attic) SearchTerms(ctx context.Context, r storage.Reader[storage.Library], _ works.ItemID) (indexers.Terms, error) {
	e, err := a.Expect(ctx, r, works.WorkID{})
	return indexers.Terms{Expect: e, Mode: "search", Query: "The Attic"}, err
}

func (attic) Groups(_ context.Context, _ storage.Reader[storage.Library], _ works.WorkID, items []works.ItemID) ([]indexers.Group, error) {
	return indexers.Singles(items, "The Attic"), nil
}

func (a attic) GroupTerms(ctx context.Context, r storage.Reader[storage.Library], _ works.WorkID, g indexers.Group) (indexers.Terms, error) {
	return a.SearchTerms(ctx, r, g.Items[0])
}

func (attic) Categories() []newznab.ID { return newznab.Family(2000) }

// Dated reads a name's words up to its year as its title, and a word
// such as E03 or E01-E09 after it as its mark. A name with no year reads
// no title. Its clean rule folds case.
type Dated struct{}

func (Dated) ReadName(p rel.Prepared) rel.Name {
	words := strings.Split(p.Title.String(), ".")
	for i, w := range words {
		if y, err := strconv.Atoi(w); err == nil && y > 1900 && i > 0 {
			n := rel.Name{Titles: []string{strings.Join(words[:i], " ")}, Year: rel.Known(y)}
			for _, m := range words[i+1:] {
				if mark.MatchString(m) {
					n.Marks = []rel.Mark{{Words: m, Key: m}}
				}
			}
			return n
		}
	}
	return rel.Name{}
}

var mark = regexp.MustCompile(`^E\d\d(-E\d\d)?$`)

// episode is the mark of the i-th episode, from 1: E03.
func episode(i int) string { return fmt.Sprintf("E%02d", i) }

// Pack is the mark of a Release covering episodes first to last, from 1:
// E01-E09, or E10 alone.
func Pack(first, last int) string {
	if first == last {
		return episode(first)
	}
	return episode(first) + "-" + episode(last)
}

func (Dated) CleanTitle(title string) string { return strings.ToLower(title) }

// alone is a roster of one Node, in step, that holds every Role.
type alone struct{ node role.NodeID }

func (r alone) Self() role.NodeID                                  { return r.node }
func (r alone) Pick(role.Name, role.NodeID) (role.NodeID, bool)    { return r.node, true }
func (alone) Away(role.Name, role.NodeID) (nodes.Absence, bool)    { return nodes.Absence{}, false }
func (alone) WrittenOff(role.Name, role.NodeID) bool               { return false }
func (alone) Holds(role.Name, role.NodeID) bool                    { return true }
func (r alone) Holding(role.Name) []role.NodeID                    { return []role.NodeID{r.node} }
func (r alone) Serving(role.Name) []role.NodeID                    { return []role.NodeID{r.node} }
func (alone) Name(role.NodeID) string                              { return "home" }
func (alone) PickOther(role.Name, role.NodeID) (role.NodeID, bool) { return role.NodeID{}, false }
func (r alone) Route(role.Name, role.NodeID) (role.NodeID, error)  { return r.node, nil }
func (alone) InStep(role.NodeID) bool                              { return true }
func (alone) Changed() <-chan struct{}                             { return nil }

// noDownloads is a Downloading Role with no Torrents.
type noDownloads struct{}

func (noDownloads) Touch(...role.NodeID)                  {}
func (noDownloads) Moved() <-chan struct{}                { return nil }
func (noDownloads) Retargeted(downloadclients.Retargeted) {}
func (noDownloads) Touched(...downloadclients.Touched)    {}
func (noDownloads) Sample(downloadclients.ID) (downloadclients.Sample, bool) {
	return downloadclients.Sample{}, false
}

func (noDownloads) RemoveIn(context.Context, *storage.Tx[storage.Library], []downloadclients.ID, downloadclients.RemoveReason, bool) ([]downloadclients.Result, downloadclients.Touched, error) {
	panic("grabstest: the World has no Torrent to remove")
}

func (noDownloads) Resolve(context.Context, role.NodeID, magnet.Link, time.Time) (downloadclients.Resolution, error) {
	panic("grabstest: the World asks no Node for metadata")
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

func write(t testing.TB, lib *storage.DB[storage.Library], fn func(ctx context.Context, tx *storage.Tx[storage.Library]) error) {
	t.Helper()
	must(t, lib.Write(t.Context(), fn))
}

func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
