package grabs_test

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// twin is The Attic's one Item searched over scripted answers: what each
// Indexer lists, best priority first. Automatic search and Manual search
// of it must agree on the Release to grab.
type twin struct {
	name  string
	sites []twinSite
	// goals is the Work's Targets in Profile order, HD alone when nil.
	goals []targets.Spec
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

// newTwinWorld is The Attic whose Indexers answer as tw's sites.
func newTwinWorld(t *testing.T, tw twin) atticWorld {
	t.Helper()
	answers := scripted{}
	a := newAttic(t, answers, tw.sites, tw.goals, slog.New(slog.DiscardHandler))
	for _, site := range tw.sites {
		answers[a.ids[site.name].String()] = site
	}
	return a
}

// atticWorld is a library holding The Attic, monitored, rooted and
// wanting its goals, or HD (Bluray-1080p over WEB-DL 1080p) when none
// are given, with an Indexer for each site, whose searches remote
// answers. A cell stops at its first failed Grab: a fetch that isn't a
// torrent fails, and the search again that would follow refuses the
// failed Release and replaces the run that grabbed it. g is the first
// Grabs opened over it.
type atticWorld struct {
	g     *grabs.Grabs
	lib   library
	cfg   grabs.Config
	home  role.NodeID
	work  works.WorkID
	ids   map[string]indexers.ID
	names map[indexers.ID]string
	// notices keeps the events the Grabs record.
	notices *entrytest.Recorder
}

func newAttic(t *testing.T, remote searching.Remote, sites []twinSite, goals []targets.Spec, log *slog.Logger) atticWorld {
	t.Helper()
	ctx := t.Context()
	names := make([]string, len(sites))
	for i, site := range sites {
		names[i] = site.name
	}
	w := grabstest.New(t, grabstest.Options{
		Sites: names, Goals: goals, Remote: remote, Log: log,
		Grabs: grabs.DefaultTimings(), Indexers: indexers.DefaultTimings(), Formats: formats.DefaultTimings(),
	})
	settings(t, w, map[string]string{grabs.StopCount.Key: "1"})
	world := atticWorld{lib: w.Lib, cfg: w.Config, home: w.Home, work: w.Work, ids: w.Sites, names: map[indexers.ID]string{}, notices: w.History}
	for _, site := range sites {
		id := w.Sites[site.name]
		world.names[id] = site.name
		edit := indexers.Edit{Priority: new(site.priority), AutoSearch: new(!site.manualOnly)}
		if site.spent {
			edit.Grabs = &indexers.Limit{Count: 1, Per: indexers.Day}
		}
		if _, err := w.Indexers.Update(ctx, identity.Admin{}, id, edit, indexers.Typed{}); err != nil {
			t.Fatal(err)
		}
		if site.spent {
			run(t, w.Lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				_, err := w.Indexers.Take(ctx, tx, indexers.Release{Indexer: id, Release: listing("The.Attic.2020.Earlier", 1, "")})
				return err
			})
		}
	}
	world.g = world.open()
	return world
}

// open is a new Grabs over the world's library, as a Controller that
// starts again opens one.
func (a atticWorld) open() *grabs.Grabs {
	return grabs.New(a.cfg)
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
	if len(runs[0].Grabs) == 0 {
		return ""
	}
	v, err := a.g.Get(ctx, identity.Admin{}, runs[0].Grabs[0])
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
				got := ""
				if p := newTwinWorld(t, tw).searchedByHand(t).Plan.Rows[0].Pick; p != nil {
					got = p.Release.Title
				}
				if _, want, _ := strings.Cut(tw.grabbed, "Site "); got != want {
					t.Fatalf("the plan's first row is %q, want automatic search's Grab %q", got, want)
				}
			})
		})
	}
}

// searchedByHand is a Manual search of the Work once every Indexer has
// responded.
func (a atticWorld) searchedByHand(t *testing.T) grabs.ManualView {
	t.Helper()
	ctx := t.Context()
	if _, _, err := a.g.StartManualSearch(ctx, identity.Admin{}, a.work, grabs.ManualScope{}); err != nil {
		t.Fatal(err)
	}
	stop := running(t, a.g)
	synctest.Wait()
	stop()
	v, err := a.g.ManualSearch(ctx, identity.Admin{}, a.work, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v.Searching {
		t.Fatalf("the Manual search is still searching: %s", v.Line)
	}
	return v
}

func TestAManualOnlyIndexersBetterReleaseLeadsThePlan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tw := twin{sites: []twinSite{
			{name: "Better Site", priority: 10, releases: []searching.Release{listing("The.Attic.2020.1080p.WEB-DL-GRP", 90, "a")}},
			{name: "Hidden Site", priority: 25, manualOnly: true, releases: []searching.Release{
				listing("The.Attic.2020.1080p.BluRay.x264-HID", 50, "b"),
				listing("The.Attic.2020.1080p.BluRay.x264-HID2", 5, "c"),
			}},
		}}
		if got := newTwinWorld(t, tw).searchedAutomatically(t); got != "Better Site The.Attic.2020.1080p.WEB-DL-GRP" {
			t.Fatalf("automatic search grabbed %q, want Better Site's WEB-DL", got)
		}
		row := newTwinWorld(t, tw).searchedByHand(t).Plan.Rows[0]
		if row.Pick == nil || row.Pick.Release.Title != "The.Attic.2020.1080p.BluRay.x264-HID" {
			t.Fatalf("the plan's first row is %+v, want Hidden Site's BluRay", row.Pick)
		}
		var choices []string
		for _, c := range row.Choices {
			choices = append(choices, c.Release.Title+": "+c.Note)
		}
		want := []string{
			"The.Attic.2020.1080p.BluRay.x264-HID2: From Hidden Site, which automatic search doesn't use.",
			"The.Attic.2020.1080p.WEB-DL-GRP: ",
		}
		if !slices.Equal(choices, want) {
			t.Fatalf("Other choices are %q, want %q", choices, want)
		}
	})
}
