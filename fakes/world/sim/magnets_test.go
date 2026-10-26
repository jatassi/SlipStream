package sim_test

import (
	"fmt"
	"net/http"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// magnetic is home with its 1080p release, the one a search grabs, as
// edit makes it.
func magnetic(t *testing.T, edit func(*scenario.Release)) scenario.Scenario {
	t.Helper()
	s := home(t)
	edit(&s.Releases[0])
	return s
}

var (
	publicMagnet  = func(r *scenario.Release) { r.Public, r.Link = true, scenario.MagnetLink }
	trackerless   = func(r *scenario.Release) { r.Public, r.Link, r.Trackerless = true, scenario.MagnetLink, true }
	privateMagnet = func(r *scenario.Release) { r.Link = scenario.MagnetLink }
)

// fetches counts the metadata searches seedbox's engines start, across
// its starts.
func fetches(n *atomic.Int64) sim.Option {
	return onSeedbox(func(e *node.Env) {
		open := e.Engine
		e.Engine = func(cfg engine.Config) (engine.Engine, error) {
			eng, err := open(cfg)
			return metadataFetches{Engine: eng, n: n}, err
		}
	})
}

type metadataFetches struct {
	engine.Engine
	n *atomic.Int64
}

func (s metadataFetches) FetchMetadata(p engine.MetadataParams) (engine.Handle, error) {
	s.n.Add(1)
	return s.Engine.FetchMetadata(p)
}

// metadataWait sets downloading.metadataWait through the API.
func metadataWait(t *testing.T, w *sim.World, wait time.Duration) {
	t.Helper()
	body := fmt.Sprintf(`{"metadataWait":%d}`, int64(wait/time.Second))
	if r := w.API().Admin("home", http.MethodPatch, "/api/v1/download-clients/thresholds", body); r.Status != http.StatusOK {
		t.Fatalf("setDownloadClientThresholds(%s) = %d %s", body, r.Status, r.Body)
	}
}

type torrentRows struct {
	Rows []struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	} `json:"rows"`
}

// expectTorrent checks the one Torrent is the release's, at its real
// size.
func expectTorrent(t *testing.T, w *sim.World) {
	t.Helper()
	rows := decode[torrentRows](t, "listTorrents", w.API().Admin("home", http.MethodGet, "/api/v1/torrents?tab=all", "")).Rows
	if len(rows) != 1 || rows[0].Name != release+".mkv" || rows[0].Size != sim.PayloadSize {
		t.Fatalf("listTorrents = %+v, want the one Torrent %s.mkv of %d bytes", rows, release, sim.PayloadSize)
	}
}

func TestAPublicMagnetLinkResolvesAndImports(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var n atomic.Int64
		w := sim.Launch(t, magnetic(t, publicMagnet), sim.WithWalk(), fetches(&n))
		search(t, w, addOrchard(t, w))
		g := await(t, w)
		expectImported(t, g)
		expectPlaced(t, w)
		expectTorrent(t, w)
		if g.InfoHash != w.InfoHash(release).String() || n.Load() != 1 {
			t.Fatalf("the Grab's info hash is %s after %d searches, want %s after one", g.InfoHash, n.Load(), w.InfoHash(release))
		}
	})
}

func TestATrackerlessMagnetLinkResolvesThroughTheDHT(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, magnetic(t, trackerless), sim.WithWalk())
		search(t, w, addOrchard(t, w))
		expectImported(t, await(t, w))
		expectPlaced(t, w)
		swarm, h := w.Node("seedbox").Swarm(), w.InfoHash(release)
		if swarm.Lookups(h) == 0 {
			t.Fatal("seedbox never looked the release up in the DHT")
		}
		for _, a := range swarm.Announces(w.Announce()) {
			if a.InfoHash == h {
				t.Fatalf("seedbox announced the trackerless release to the tracker: %+v", a)
			}
		}
	})
}

func TestATrackerlessMagnetLinkWithTheDHTOffFailsAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var n atomic.Int64
		w := sim.Launch(t, magnetic(t, trackerless), sim.WithWalk(), fetches(&n))
		box := nodeID(t, w, "seedbox").String()
		if r := w.API().Admin("home", http.MethodPatch, "/api/v1/download-clients/"+box+"/settings", `{"dht":false}`); r.Status != http.StatusOK {
			t.Fatalf("setDownloadClientSettings = %d %s", r.Status, r.Body)
		}
		synctest.Wait()
		start := time.Now()
		search(t, w, addOrchard(t, w))
		g := await(t, w)
		expectFailed(t, g, "fetching", "no_peer_source", "The magnet link names no tracker, and DHT is off on the Node that downloads it")
		if took := time.Since(start); took > time.Minute || n.Load() != 0 || !slices.Equal(g.stages(), []string{"fetching", "failed"}) {
			t.Fatalf("the Grab failed after %s and %d searches through %v, want at once with none", took, n.Load(), g.stages())
		}
	})
}

func TestAPrivateMagnetLinkFailsOnceTheMetadataWaitPasses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, magnetic(t, privateMagnet), sim.WithWalk())
		metadataWait(t, w, 2*time.Minute)
		start := time.Now()
		search(t, w, addOrchard(t, w))
		g := await(t, w)
		took := time.Since(start)
		expectFailed(t, g, "fetching", "no_metadata", "No peer sent the metadata in 2 min")
		if took < 2*time.Minute || took > 2*time.Minute+time.Minute {
			t.Fatalf("the Grab failed %s after the search, want once the 2 minute wait passed", took)
		}
		if rows := decode[torrentRows](t, "listTorrents", w.API().Admin("home", http.MethodGet, "/api/v1/torrents?tab=all", "")).Rows; len(rows) != 0 {
			t.Fatalf("listTorrents = %+v, want none", rows)
		}
		if n := w.Node("seedbox").Swarm().Lookups(w.InfoHash(release)); n != 0 {
			t.Fatalf("the DHT saw %d lookups of the private release, want none", n)
		}
	})
}

func TestARedirectToAMagnetLinkResolvesAndImports(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, magnetic(t, func(r *scenario.Release) { r.Public, r.Link = true, scenario.RedirectLink }), sim.WithWalk())
		search(t, w, addOrchard(t, w))
		expectImported(t, await(t, w))
		expectPlaced(t, w)
		expectTorrent(t, w)
	})
}

// TestTwoGrabsOfOneMagnetLinkMakeOneSearch lists the release for a second
// movie under the same magnet link, as a pack holds both, and searches
// both at once.
func TestTwoGrabsOfOneMagnetLinkMakeOneSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := magnetic(t, publicMagnet)
		s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: 9000002, IMDbID: "tt9000002", Title: "Harbor Lights", Released: "2019-09-06", Runtime: 98 * time.Minute})
		second := s.Releases[0]
		second.Name, second.TMDBID = "Harbor.Lights.2019.1080p.WEB-DL.H264-FKW", 9000002
		s.Releases = append(s.Releases, second)
		_, orchard, err := releases.Metainfo(release+".mkv", sim.Payload(release), "https://127.0.0.1/announce/"+releases.Passkey(s), true)
		if err != nil {
			t.Fatal(err)
		}
		var n atomic.Int64
		w := sim.Launch(t, s, sim.WithWalk(), fetches(&n), sim.Listed(func(r *site.Release) {
			if r.Name == second.Name {
				r.InfoHash = orchard.String()
			}
		}))
		harbor := decode[struct {
			Work struct {
				ID string `json:"id"`
			} `json:"work"`
		}](t, "addWork", w.API().Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000002","monitor":true}`)).Work.ID
		search(t, w, addOrchard(t, w))
		search(t, w, harbor)
		var gs []grabView
		eventually(t, "both Grabs leave fetching", func() bool {
			gs = grabsListed(t, w)
			return len(gs) == 2 && gs[0].Stage != "fetching" && gs[1].Stage != "fetching"
		})
		for _, g := range gs {
			if g.InfoHash != orchard.String() || g.Failure != nil {
				t.Fatalf("the Grabs are %+v, want both past fetching with %s", gs, orchard)
			}
		}
		if got := n.Load(); got != 1 {
			t.Fatalf("seedbox started %d metadata searches, want one for both Grabs", got)
		}
	})
}

func TestAControllerRestartedMidSearchJoinsIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var n atomic.Int64
		reached, release := make(chan struct{}), make(chan struct{})
		w := sim.Launch(t, magnetic(t, publicMagnet), sim.WithWalk(), onSeedbox(func(e *node.Env) {
			e.Engine = holdAt[engine.MetadataReceived](e.Engine, reached, release)
		}), fetches(&n))
		search(t, w, addOrchard(t, w))
		<-reached
		h := w.Node("home")
		link := h.Ready().Listeners.Link
		h.Stop()
		h.Start(func(e *node.Env) { e.Environ = append(e.Environ, "SLIPSTREAM_LINK_LISTEN="+link.String()) })
		time.Sleep(time.Minute)
		close(release)
		expectImported(t, await(t, w))
		expectPlaced(t, w)
		if got := n.Load(); got != 1 {
			t.Fatalf("seedbox started %d metadata searches, want the restarted Controller to join the first", got)
		}
	})
}

func TestANodeRestartedMidSearchKeepsTheDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var n atomic.Int64
		w := sim.Launch(t, magnetic(t, privateMagnet), sim.WithWalk(), fetches(&n))
		metadataWait(t, w, 3*time.Minute)
		start := time.Now()
		search(t, w, addOrchard(t, w))
		time.Sleep(2 * time.Minute)
		box := w.Node("seedbox")
		box.Stop()
		box.Start()
		g := await(t, w)
		took := time.Since(start)
		expectFailed(t, g, "fetching", "no_metadata", "No peer sent the metadata in 3 min")
		if took < 3*time.Minute || took >= 4*time.Minute || n.Load() != 2 {
			t.Fatalf("the Grab failed %s after the search with %d searches, want within a minute of the 3 minute wait with a second search after the restart", took, n.Load())
		}
		if got := box.Swarm().Lookups(w.InfoHash(release)); got != 0 {
			t.Fatalf("the DHT saw %d lookups of the private release, want none", got)
		}
	})
}
