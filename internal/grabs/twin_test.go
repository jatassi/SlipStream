package grabs_test

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// twin is The Attic's one Item searched over scripted answers: what each
// Indexer lists, best priority first. Automatic search and Manual search
// of it must agree on the Release to grab.
type twin struct {
	name  string
	sites []twinSite
	// grabbed is the Release automatic search grabs, as "<site>
	// <title>", empty when it grabs none.
	grabbed string
}

// twinSite is an Indexer of a twin, at priority, listing releases. spent
// has a grab limit of one a day and its one grab taken, and manualOnly
// has Automatic search off. A fetch from it answers torrent, else bytes
// that aren't bencode.
type twinSite struct {
	name       string
	priority   int
	spent      bool
	manualOnly bool
	releases   []searching.Release
	torrent    []byte
}

func listing(title string, seeders int, hash string) searching.Release {
	r := searching.Release{Title: title, Size: 4096, Seeders: seeders, Link: searching.LinkOf("https://site.test/dl/" + title)}
	if hash != "" {
		h := bencode.InfoHash([]byte(hash + "aaaaaaaaaaaaaaaaaaa"))
		r.InfoHash = &h
	}
	return r
}

var twins = []twin{
	{
		name: "the chain's best across Indexers",
		sites: []twinSite{
			{name: "Better Site", priority: 10, releases: []searching.Release{
				listing("The.Attic.2020.1080p.WEB-DL-GRP", 50, "a"),
				listing("The.Attic.2020.720p.WEB-DL-GRP", 90, "b"),
			}},
			{name: "Other Site", priority: 25, releases: []searching.Release{listing("The.Attic.2020.1080p.BluRay.x264-GRP", 3, "c")}},
		},
		grabbed: "Other Site The.Attic.2020.1080p.BluRay.x264-GRP",
	},
	{
		name: "a copy from a worse Indexer when the better one is at its grab limit",
		sites: []twinSite{
			{name: "Better Site", priority: 10, spent: true, releases: []searching.Release{listing("The.Attic.2020.1080p.BluRay.x264-GRP", 9, "a")}},
			{name: "Other Site", priority: 25, releases: []searching.Release{
				listing("The.Attic.2020.1080p.BluRay.x264-GRP", 9, "a"),
				listing("The.Attic.2020.1080p.WEB-DL-GRP", 90, "b"),
			}},
		},
		grabbed: "Other Site The.Attic.2020.1080p.BluRay.x264-GRP",
	},
	{
		name: "past a better-ranked Release of another title",
		sites: []twinSite{
			{name: "Better Site", priority: 10, releases: []searching.Release{
				listing("The.Atticus.2020.1080p.BluRay.x264-GRP", 90, "a"),
				listing("The.Attic.2020.1080p.WEB-DL-GRP", 5, "b"),
			}},
		},
		grabbed: "Better Site The.Attic.2020.1080p.WEB-DL-GRP",
	},
	{
		name: "nothing when no Release is acceptable",
		sites: []twinSite{
			{name: "Better Site", priority: 10, releases: []searching.Release{
				listing("The.Attic.2020.720p.WEB-DL-GRP", 90, "a"),
				listing("The.Cellar.2020.1080p.BluRay.x264-GRP", 90, "b"),
			}},
		},
	},
}

// scripted answers each Indexer as its twin site.
type scripted map[string]twinSite

func (scripted) ReachSolver(context.Context, searching.Solver) (int, error) { return 0, nil }

func (s scripted) Run(_ context.Context, job searching.Job) (searching.Poll, error) {
	var a searching.Answer
	switch job.Call.(type) {
	case searching.SearchCall:
		a.Releases = s[job.Indexer].releases
	case searching.FetchCall:
		raw := s[job.Indexer].torrent
		if raw == nil {
			raw = []byte("not bencode")
		}
		a.Got = searching.Torrent{Raw: raw}
	}
	return searching.Poll{Answer: &a}, nil
}

// attic is The Attic's search terms: its label, and every name covering
// its one Item.
type attic struct{ item works.ItemID }

func (a *attic) Expect(context.Context, storage.Reader[storage.Library], works.WorkID) (indexers.Expect, error) {
	return indexers.Expect{Claim: rel.Claim{Titles: []string{"The Attic"}}, Covers: indexers.CoversOnly(a.item)}, nil
}

func (a *attic) SearchTerms(ctx context.Context, r storage.Reader[storage.Library], _ works.ItemID) (indexers.Terms, error) {
	e, err := a.Expect(ctx, r, works.WorkID{})
	return indexers.Terms{Expect: e, Mode: "search", Query: "The Attic"}, err
}

func (*attic) Categories() []newznab.ID { return newznab.Family(2000) }

// newTwinWorld is The Attic whose Indexers answer as tw's sites.
func newTwinWorld(t *testing.T, tw twin) atticWorld {
	t.Helper()
	answers := scripted{}
	a := newAttic(t, answers, tw.sites, slog.New(slog.DiscardHandler))
	for _, site := range tw.sites {
		answers[a.ids[site.name].String()] = site
	}
	return a
}

// atticWorld is a library holding The Attic, monitored, rooted and
// wanting HD: Bluray-1080p over WEB-DL 1080p, with an Indexer for each
// site, whose searches remote answers. g is the first Grabs opened over
// it.
type atticWorld struct {
	g     *grabs.Grabs
	lib   library
	cfg   grabs.Config
	work  works.WorkID
	ids   map[string]indexers.ID
	names map[indexers.ID]string
}

func newAttic(t *testing.T, remote searching.Remote, sites []twinSite, log *slog.Logger) atticWorld {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, schemas()...)
	home := role.NodeID{UUID: storage.NewUUID()}
	sealer := secrettest.Sealer(t)
	fx := contenttypetest.New()
	w, err := works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Removed: discard[works.Removed]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	goals, err := targets.Open(ctx, targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{}, Formats: formats.DefaultTimings(),
	})
	if err != nil {
		t.Fatal(err)
	}
	hd, err := goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{
		Scores: targets.Scores{Increment: 1}, Name: "HD", Cutoff: rel.Bluray1080p,
		Tiers: [][]rel.Quality{{rel.Bluray1080p}, {rel.WEBDL1080p}},
	})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{
		Name: "Standard", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := goals.MakeDefault(ctx, identity.Admin{}, profile.ID); err != nil {
		t.Fatal(err)
	}
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	tree, _, err := w.Add(ctx, identity.Admin{}, contenttypetest.ID, "w1", works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, nil); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	library, err := roots.Open(ctx, rootstest.Config(t, lib, contenttypetest.ID))
	if err != nil {
		t.Fatal(err)
	}
	r := &roster{node: home, changed: make(chan struct{})}
	router := downloadclients.Router{Roster: r, Routes: trackers.Routing{Roster: r}}
	shelf, err := files.Open(ctx, files.Config{Library: lib, Targets: goals, Grabs: grabs.GrabCells{InStep: r.InStep, Router: router}, Today: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	terms := &attic{item: tree.Items[0].ID}
	xs := indexers.New(indexers.Config{
		Library: lib, Sealer: sealer, Roster: indexerstest.Alone(home), Searching: func(role.NodeID) searching.Remote { return remote },
		Types: map[works.TypeID]indexers.Facet{contenttypetest.ID: terms}, Works: w,
		Changed: discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Trackers: discard[indexers.TrackerChanged]{},
		Timings: indexers.DefaultTimings(), Log: slog.New(slog.DiscardHandler),
	})
	world := atticWorld{lib: lib, work: tree.ID, ids: map[string]indexers.ID{}, names: map[indexers.ID]string{}}
	run(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, home, "home")
	})
	for _, site := range sites {
		id := indexers.ID{UUID: storage.NewUUID()}
		world.names[id], world.ids[site.name] = site.name, id
		run(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return indexerstest.Torznab(ctx, tx, sealer, id, site.name, home, "https://site.test/api", "k")
		})
		edit := indexers.Edit{Priority: new(site.priority), AutoSearch: new(!site.manualOnly)}
		if site.spent {
			edit.Grabs = &indexers.Limit{Count: 1, Per: indexers.Day}
		}
		if _, err := xs.Update(ctx, identity.Admin{}, id, edit, indexers.Typed{}); err != nil {
			t.Fatal(err)
		}
		if site.spent {
			run(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				_, err := xs.Take(ctx, tx, indexers.Release{Indexer: id, Release: listing("The.Attic.2020.Earlier", 1, "")})
				return err
			})
		}
	}
	rootstest.Root(t, lib, home, contenttypetest.ID, t.TempDir())
	world.cfg = grabs.Config{
		Library: lib, Sealer: sealer, Works: w, Targets: goals, Files: shelf, Indexers: xs,
		Terms: map[works.TypeID]indexers.Facet{contenttypetest.ID: terms}, Names: map[works.TypeID]rel.Names{contenttypetest.ID: dated{}},
		Roster: r, Downloads: &swarm{}, Roots: library, Router: router,
		Timings: grabs.DefaultTimings(), Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Changed: discard[grabs.Changed]{}, SearchChanged: discard[grabs.SearchChanged]{}, FilesChanged: discard[files.Changed]{},
		BlocklistChanged: discard[grabs.BlocklistChanged]{}, Reconciling: trackers.NotReconciling, Log: log,
	}
	world.g = world.open()
	return world
}

// open is a new Grabs over the world's library, as a Controller that
// starts again opens one.
func (a atticWorld) open() *grabs.Grabs {
	cfg := a.cfg
	cfg.Transfers = transfers.New(transfers.ControllerConfig{Library: a.lib})
	return grabs.New(cfg)
}

// searchedAutomatically is what an automatic search of the Work grabs, as
// "<site> <title>", empty when it grabs none.
func (a atticWorld) searchedAutomatically(t *testing.T) string {
	t.Helper()
	ctx := t.Context()
	if _, err := a.g.SearchWork(ctx, identity.Admin{}, a.work, grabs.Scope{}); err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(ctx)
	done := make(chan error)
	go func() { done <- a.g.Run(runCtx) }()
	synctest.Wait()
	stop()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	runs, err := a.g.Searches(ctx, identity.Admin{}, a.work)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Finished.IsZero() {
		t.Fatalf("the Work has runs %+v, want one finished", runs)
	}
	if runs[0].GrabID.IsZero() {
		return ""
	}
	v, err := a.g.Get(ctx, identity.Admin{}, runs[0].GrabID)
	if err != nil {
		t.Fatal(err)
	}
	return a.names[v.Ticket.Indexer] + " " + v.Release.Title
}

func TestTwinsAutomaticSearchGrabsTheChainsFirstOfferAnIndexerTakes(t *testing.T) {
	for _, tw := range twins {
		t.Run(tw.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				if got := newTwinWorld(t, tw).searchedAutomatically(t); got != tw.grabbed {
					t.Fatalf("automatic search grabbed %q, want %q", got, tw.grabbed)
				}
			})
		})
	}
}
