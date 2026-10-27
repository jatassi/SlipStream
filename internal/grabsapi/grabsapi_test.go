package grabsapi_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabsapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

var admin identity.Admin

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type sent[T any] chan T

func (s sent[T]) Publish(_ context.Context, v T) { s <- v }

type pipeline struct{ g *grabs.Grabs }

func (p pipeline) Grabs() (*grabs.Grabs, bool) { return p.g, true }

// terms is a Content Type's search terms, empty until set.
type terms struct{ t indexers.Terms }

func (s *terms) SearchTerms(context.Context, storage.Reader[storage.Library], works.ItemID) (indexers.Terms, error) {
	return s.t, nil
}

func (*terms) Categories() []newznab.ID { return newznab.Family(2000) }

// alone is a roster of one Node holding every Role.
type alone struct{ self role.NodeID }

func (a alone) Self() role.NodeID                                  { return a.self }
func (a alone) Pick(role.Name, role.NodeID) (role.NodeID, bool)    { return a.self, true }
func (a alone) Route(role.Name, role.NodeID) (role.NodeID, error)  { return a.self, nil }
func (alone) Away(role.Name, role.NodeID) (nodes.Absence, bool)    { return nodes.Absence{}, false }
func (alone) InStep(role.NodeID) bool                              { return true }
func (alone) WrittenOff(role.Name, role.NodeID) bool               { return false }
func (a alone) Holding(role.Name) []role.NodeID                    { return []role.NodeID{a.self} }
func (alone) PickOther(role.Name, role.NodeID) (role.NodeID, bool) { return role.NodeID{}, false }
func (alone) Name(role.NodeID) string                              { return "home" }
func (alone) Changed() <-chan struct{}                             { return nil }
func (alone) Touch(...role.NodeID)                                 {}
func (alone) Moved() <-chan struct{}                               { return nil }
func (alone) Sample(downloadclients.ID) (downloadclients.Sample, bool) {
	return downloadclients.Sample{}, false
}

func (alone) Resolve(context.Context, role.NodeID, magnet.Link, time.Time) (downloadclients.Resolution, error) {
	panic("a .torrent link needs no metadata search")
}

type lapsing struct{ lapsed atomic.Bool }

func (*lapsing) Holds(role.Name, role.NodeID) bool { return true }
func (l *lapsing) InStep(role.NodeID) bool         { return !l.lapsed.Load() }

// site answers every search with a 1080p and a 720p release, and every
// .torrent fetch with bytes that aren't bencode.
type site struct{}

func (site) Run(_ context.Context, job searching.Job) (searching.Poll, error) {
	var a searching.Answer
	switch job.Call.(type) {
	case searching.SearchCall:
		a.Releases = []searching.Release{
			{Title: "The.Attic.1080p.WEB", Size: 4096, Link: searching.LinkOf("https://site.test/dl/1"), Seeders: 5},
			{Title: "The.Attic.720p.WEB", Size: 2048, Link: searching.LinkOf("https://site.test/dl/2"), Seeders: 50},
		}
	case searching.FetchCall:
		a.Got = searching.Torrent{Raw: []byte("not bencode")}
	}
	return searching.Poll{Answer: &a}, nil
}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

var stamps = regexp.MustCompile(`"\d{4}-\d{2}-\d{2}T[0-9:.]+Z"`)

func expect(t *testing.T, s *ownertest.Owner, method, path string, status int, want string) {
	t.Helper()
	code, got := s.Call(t, method, path, "")
	if got = stamps.ReplaceAllString(got, `"T"`); code != status || got != want+"\n" {
		t.Fatalf("%s %s = %d %s, want %d %s", method, path, code, got, status, want)
	}
}

// atticFixture is a Work of the fixture Content Type to search, with one
// Item, the Target HD for it and a Grabs whose Indexers ask remote. It has
// no Node, Indexer or root yet, and its terms are empty.
type atticFixture struct {
	ctx               context.Context
	lib               *storage.DB[storage.Library]
	sealer            *secret.Sealer
	home              role.NodeID
	tree, unmonitored works.Tree
	hd                targets.Target
	terms             *terms
	storageNodes      *lapsing
	xs                *indexers.Indexers
	g                 *grabs.Grabs
	changed           sent[grabs.Changed]
	searched          sent[grabs.SearchChanged]
	s                 *ownertest.Owner
	search            string
}

func searchable(t *testing.T, remote searching.Remote) *atticFixture {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(),
		roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema())
	fx := contenttypetest.New()
	w := must[*works.Works](t)(works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	}))
	goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{},
	}))
	hd := must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{Name: "HD", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p}))
	standard := must[targets.Profile](t)(goals.CreateProfile(ctx, admin, targets.ProfileSpec{
		Name: "Standard", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}},
	}))
	must[targets.Profile](t)(goals.MakeDefault(ctx, admin, standard.ID))
	assign := func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
		if err := targets.Assign(ctx, tx, w, nil); err != nil {
			return err
		}
		return roots.Assign(ctx, tx, w, nil)
	}
	attic := works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}
	fx.Put("w1", contenttypetest.Record{Tree: attic})
	fx.Put("w2", contenttypetest.Record{Tree: attic})
	tree, _, err := w.Add(ctx, admin, contenttypetest.ID, "w1", works.Adding{Monitor: true, Also: assign})
	if err != nil {
		t.Fatal(err)
	}
	unmonitored, _, err := w.Add(ctx, admin, contenttypetest.ID, "w2", works.Adding{Also: assign})
	if err != nil {
		t.Fatal(err)
	}
	terms := &terms{}
	changed, searched := make(sent[grabs.Changed], 4), make(sent[grabs.SearchChanged], 4)
	sealer := secrettest.Sealer(t)
	home := storage.NewUUID()
	roster := alone{role.NodeID{UUID: home}}
	storageNodes := &lapsing{}
	shelves := rootstest.Config(t, lib, contenttypetest.ID)
	shelves.Roster = storageNodes
	library, err := roots.Open(ctx, shelves)
	if err != nil {
		t.Fatal(err)
	}
	xs := indexers.New(indexers.Config{
		Library: lib, Sealer: sealer, Roster: roster, Searching: func(role.NodeID) searching.Remote { return remote },
		Types: map[works.TypeID]indexers.Facet{contenttypetest.ID: terms}, Works: w,
		Changed: discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Timings: indexers.DefaultTimings(), Log: slog.New(slog.DiscardHandler),
	})
	g := grabs.New(grabs.Config{
		Library: lib, Sealer: sealer, Works: w, Targets: goals, Files: files.New(files.Config{Library: lib, Works: w, Targets: goals, Grabs: grabs.GrabCells{}, Today: time.Now}),
		Indexers: xs,
		Terms:    map[works.TypeID]indexers.Facet{contenttypetest.ID: terms}, Roster: roster, Downloads: roster, Roots: library,
		Transfers: transfers.NewReceiver(transfers.Config{Local: storagetest.Local(t, transfers.Schema())}),
		Timings:   grabs.DefaultTimings(), Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Changed: changed, SearchChanged: searched, FilesChanged: discard[files.Changed]{}, Log: slog.New(slog.DiscardHandler),
	})
	return &atticFixture{
		ctx: ctx, lib: lib, sealer: sealer, home: role.NodeID{UUID: home}, tree: tree, unmonitored: unmonitored, hd: hd, terms: terms,
		storageNodes: storageNodes, xs: xs, g: g, changed: changed, searched: searched,
		s: ownertest.Serve(t, lib, func(r *api.Router) { grabsapi.Routes(r, pipeline{g}) }), search: "/api/v1/works/" + tree.ID.String() + "/search",
	}
}

// running runs the Grabs until the test ends.
func (f *atticFixture) running(t *testing.T) {
	runCtx, stop := context.WithCancel(f.ctx)
	ran := make(chan error)
	go func() { ran <- f.g.Run(runCtx) }()
	t.Cleanup(func() {
		stop()
		if err := <-ran; err != nil {
			t.Error(err)
		}
	})
}

func TestSearchingAWorkAnswersEachProblemThenGrabs(t *testing.T) {
	f := searchable(t, site{})
	ctx, lib, sealer, home, tree, hd, s, search, changed, searched := f.ctx, f.lib, f.sealer, f.home.UUID, f.tree, f.hd, f.s, f.search, f.changed, f.searched
	indexer := indexers.ID{UUID: storage.NewUUID()}

	expect(t, s, http.MethodGet, "/api/v1/grabs", http.StatusOK, `{"grabs":[]}`)
	expect(t, s, http.MethodGet, "/api/v1/grabs/"+storage.NewUUID().String(), http.StatusNotFound,
		`{"status":404,"code":"grab_not_found","title":"No Grab has that id"}`)
	none := storage.NewUUID().String()
	expect(t, s, http.MethodPost, "/api/v1/works/"+none+"/search", http.StatusNotFound,
		`{"status":404,"code":"work_not_found","title":"No Work has that id"}`)
	expect(t, s, http.MethodGet, "/api/v1/works/"+none+"/searches", http.StatusNotFound,
		`{"status":404,"code":"work_not_found","title":"No Work has that id"}`)
	expect(t, s, http.MethodPost, search, http.StatusConflict,
		`{"status":409,"code":"not_searchable","title":"The Work's Content Type can't be searched"}`)
	f.terms.t = indexers.Terms{Mode: "search", Query: "The Attic"}
	expect(t, s, http.MethodPost, search, http.StatusConflict,
		`{"status":409,"code":"no_indexer","title":"No Indexer is set up"}`)

	err := lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, role.NodeID{UUID: home}, "home"); err != nil {
			return err
		}
		return indexerstest.Torznab(ctx, tx, sealer, indexer, "Fixture Site", role.NodeID{UUID: home}, "https://site.test/api", "k")
	})
	if err != nil {
		t.Fatal(err)
	}
	expect(t, s, http.MethodPost, search, http.StatusConflict,
		`{"status":409,"code":"no_root","title":"The Work's Content Type has no library root"}`)
	rootstest.Root(t, lib, role.NodeID{UUID: home}, contenttypetest.ID, t.TempDir())
	f.storageNodes.lapsed.Store(true)
	expect(t, s, http.MethodPost, search, http.StatusConflict,
		`{"status":409,"code":"root_unreachable","title":"The Work's library root is on a Node that takes no new files now","detail":"home is offline, so the library root there takes no new files."}`)
	f.storageNodes.lapsed.Store(false)
	expect(t, s, http.MethodPost, "/api/v1/works/"+f.unmonitored.ID.String()+"/search", http.StatusConflict,
		`{"status":409,"code":"nothing_wanted","title":"The Work has no wanted cell to search for"}`)

	code, body := s.Call(t, http.MethodPost, search, "")
	run := ownertest.FirstID(body)
	if want := `{"searchIds":["` + run + `"]}` + "\n"; code != http.StatusAccepted || run == "" || body != want {
		t.Fatalf("searchWork = %d %s, want 202 %s", code, body, want)
	}
	item := tree.Items[0].ID.String()
	run1 := `{"id":"` + run + `","itemId":"` + item + `","startedAt":"T","indexers":[{"indexerId":"` + indexer.String() + `","name":"Fixture Site",`
	expect(t, s, http.MethodGet, search+"es", http.StatusOK,
		`{"searches":[`+run1+`"outcome":null,"releases":0,"sentence":"Searching"}],"finishedAt":null,"outcome":null,"candidates":0,"refused":[],"grabId":null,"notes":[]}]}`)
	if got := <-searched; got != (grabs.SearchChanged{SearchID: run, WorkID: tree.ID.String()}) {
		t.Fatalf("grabs.search-changed = %+v, want the run %s", got, run)
	}

	f.running(t)
	<-searched
	grab := (<-changed).GrabID
	if failed := <-changed; failed.GrabID != grab || failed.WorkID != tree.ID.String() {
		t.Fatalf("grabs.changed = %+v, want the Grab %s failing", failed, grab)
	}
	expect(t, s, http.MethodGet, search+"es", http.StatusOK,
		`{"searches":[`+run1+`"outcome":"found","releases":2,"sentence":"Found 2 Releases"}],"finishedAt":"T","outcome":"grabbed","candidates":2,`+
			`"refused":[{"title":"The.Attic.720p.WEB","reasons":[{"code":"quality_not_in_target","targetId":"`+hd.ID.String()+`","value":"webdl-720p"}]}],"grabId":"`+grab+`",`+
			`"notes":[{"code":"size_not_checked","targetId":"`+hd.ID.String()+`"}]}]}`)
	want := `{"id":"` + grab + `","workId":"` + tree.ID.String() + `","itemId":"` + item + `","edition":"any","targetIds":["` + hd.ID.String() + `"],` +
		`"releaseTitle":"The.Attic.1080p.WEB","quality":"webdl-1080p","releaseSize":4096,"indexerId":"` + indexer.String() + `","nodeId":"` + home.String() + `","fetchNodeId":null,"torrentId":null,"infoHash":null,` +
		`"stage":"failed","failure":{"stage":"fetching","code":"torrent_invalid","message":"bencode: the metainfo is not a dictionary"},` +
		`"history":[{"stage":"fetching","at":"T"},{"stage":"failed","at":"T"}],"createdAt":"T"}`
	expect(t, s, http.MethodGet, "/api/v1/grabs/"+grab, http.StatusOK, want)
	expect(t, s, http.MethodGet, "/api/v1/grabs", http.StatusOK, `{"grabs":[`+want+`]}`)
}

type failedGrab struct {
	ID        string `json:"id"`
	IndexerID string `json:"indexerId"`
	Stage     string `json:"stage"`
	Failure   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"failure"`
}

// failed reads the Grabs at each grabs.changed until n of them failed,
// and returns them by Indexer.
func (f *atticFixture) failed(t *testing.T, n int) map[string]failedGrab {
	t.Helper()
	for {
		<-f.changed
		_, got := f.s.Call(t, http.MethodGet, "/api/v1/grabs", "")
		var list struct {
			Grabs []failedGrab `json:"grabs"`
		}
		if err := json.Unmarshal([]byte(got), &list); err != nil {
			t.Fatal(err)
		}
		byIndexer := map[string]failedGrab{}
		for _, g := range list.Grabs {
			if g.Stage == "failed" {
				byIndexer[g.IndexerID] = g
			}
		}
		if len(byIndexer) == n {
			return byIndexer
		}
	}
}

// mirrors answers each Indexer's search with its own 1080p Release, a
// fetch from stopped with a refused account, and any other fetch with
// bytes that aren't bencode.
type mirrors struct {
	titles  map[string]string
	stopped string
}

func (m mirrors) Run(_ context.Context, job searching.Job) (searching.Poll, error) {
	var a searching.Answer
	switch job.Call.(type) {
	case searching.SearchCall:
		a.Releases = []searching.Release{{Title: m.titles[job.Indexer], Size: 4096, Link: searching.LinkOf("https://site.test/dl/" + job.Indexer), Seeders: 5}}
	case searching.FetchCall:
		if job.Indexer == m.stopped {
			a.Outcome = &searching.Stopped{Cause: searching.AccountRefused, Text: "Account suspended"}
		} else {
			a.Got = searching.Torrent{Raw: []byte("not bencode")}
		}
	}
	return searching.Poll{Answer: &a}, nil
}

func TestAGrabDroppedAtFetchingRerunsTheDecisionOverTheKeptReleases(t *testing.T) {
	stopping, other := indexers.ID{UUID: storage.NewUUID()}, indexers.ID{UUID: storage.NewUUID()}
	f := searchable(t, mirrors{titles: map[string]string{stopping.String(): "The.Attic.1080p.WEB-STOP", other.String(): "The.Attic.1080p.WEB-GO"}, stopped: stopping.String()})
	f.terms.t = indexers.Terms{Mode: "search", Query: "The Attic"}
	err := f.lib.Write(f.ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, f.home, "home"); err != nil {
			return err
		}
		if err := indexerstest.Torznab(ctx, tx, f.sealer, stopping, "Stopping Site", f.home, "https://site.test/api", "k"); err != nil {
			return err
		}
		return indexerstest.Torznab(ctx, tx, f.sealer, other, "Other Site", f.home, "https://site.test/api", "k")
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.xs.Update(f.ctx, admin, stopping, indexers.Edit{Priority: new(10)}); err != nil {
		t.Fatal(err)
	}
	rootstest.Root(t, f.lib, f.home, contenttypetest.ID, t.TempDir())
	code, body := f.s.Call(t, http.MethodPost, f.search, "")
	if code != http.StatusAccepted {
		t.Fatalf("searchWork = %d %s, want 202", code, body)
	}
	f.running(t)

	failed := f.failed(t, 2)
	dropped, rerun := failed[stopping.String()], failed[other.String()]
	if dropped.Failure.Code != "indexer_stopped" || dropped.Failure.Message != "The Indexer stopped before the .torrent was fetched" || rerun.Failure.Code != "torrent_invalid" {
		t.Fatalf("the Grabs are %+v, want Stopping Site's dropped indexer_stopped and Other Site's from the rerun", failed)
	}
	_, got := f.s.Call(t, http.MethodGet, f.search+"es", "")
	var runs struct {
		Searches []struct {
			Outcome    string `json:"outcome"`
			Candidates int    `json:"candidates"`
			GrabID     string `json:"grabId"`
		} `json:"searches"`
	}
	if err := json.Unmarshal([]byte(got), &runs); err != nil || len(runs.Searches) != 1 {
		t.Fatalf("listSearches = %s, %v; want one run", got, err)
	}
	if r := runs.Searches[0]; r.Outcome != "grabbed" || r.Candidates != 2 || r.GrabID != rerun.ID {
		t.Fatalf("the run is %+v, want grabbed of 2 candidates with the rerun's Grab %s", r, rerun.ID)
	}
	var kept int
	if err := f.lib.QueryRowContext(f.ctx, `SELECT count(*) FROM grabs_release`).Scan(&kept); err != nil || kept != 0 {
		t.Fatalf("grabs_release holds %d rows (%v) once the rerun's Grab failed, want none", kept, err)
	}
}
