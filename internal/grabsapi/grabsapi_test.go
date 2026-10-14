package grabsapi_test

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"regexp"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabsapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/search/v1/searchv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
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

// nobody is a roster where no Node holds Downloading.
type nobody struct{}

func (nobody) Holders(role.Name) []role.NodeID                { return nil }
func (nobody) AwaitInStep(context.Context, role.NodeID) error { return nil }
func (nobody) Touch(...role.NodeID)                           {}
func (nobody) Facts(role.NodeID) (*dlv1.Facts, bool)          { return nil, false }

// site answers every search with one 1080p release.
type site struct {
	searchv1connect.UnimplementedSearchingServiceHandler
}

func (site) Search(context.Context, *searchv1.SearchRequest) (*searchv1.SearchResponse, error) {
	body := `<rss><channel><item><title>The.Attic.1080p.WEB</title><size>4096</size><link>https://site.test/dl/1</link>` +
		`<attr name="seeders" value="5"/></item></channel></rss>`
	return &searchv1.SearchResponse{Body: []byte(body), Status: http.StatusOK}, nil
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

func TestSearchingAWorkAnswersEachProblemThenGrabs(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(),
		roots.Schema(), indexers.Schema(), grabs.Schema(), files.Schema())
	fx := contenttypetest.New()
	w := must[*works.Works](t)(works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	}))
	types := []works.TypeID{contenttypetest.ID}
	goals := targets.New(targets.Config{Library: lib, Types: types, Changed: discard[targets.Changed]{}})
	hd := must[targets.Target](t)(goals.CreateTarget(ctx, admin, "1080p", []targets.Resolution{targets.R1080p}))
	must[targets.Profile](t)(goals.CreateProfile(ctx, admin, "Standard", contenttypetest.ID, []targets.TargetID{hd.ID}))
	attic := works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}
	fx.Put("w1", contenttypetest.Record{Tree: attic})
	fx.Put("w2", contenttypetest.Record{Tree: attic})
	tree, _, err := w.Add(ctx, admin, contenttypetest.ID, "w1", true)
	if err != nil {
		t.Fatal(err)
	}
	unmonitored, _, err := w.Add(ctx, admin, contenttypetest.ID, "w2", false)
	if err != nil {
		t.Fatal(err)
	}
	terms := &terms{}
	changed, searched := make(sent[grabs.Changed], 4), make(sent[grabs.SearchChanged], 4)
	g := grabs.New(grabs.Config{
		Library: lib, Works: w, Files: files.New(files.Config{Library: lib, Works: w, Grabs: grabs.GrabCells{}}),
		Indexers: indexers.New(indexers.Config{
			Library: lib, Searching: func(role.NodeID) searchv1connect.SearchingServiceHandler { return site{} },
			Changed: discard[indexers.Changed]{},
		}),
		Terms: map[works.TypeID]indexers.Facet{contenttypetest.ID: terms}, Roster: nobody{}, Downloads: nobody{},
		Timings: grabs.DefaultTimings(), Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Changed: changed, SearchChanged: searched, FilesChanged: discard[files.Changed]{}, Log: slog.New(slog.DiscardHandler),
	})
	s := ownertest.Serve(t, lib, func(r *api.Router) { grabsapi.Routes(r, pipeline{g}) })
	search := "/api/v1/works/" + tree.ID.String() + "/search"

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
	terms.t = indexers.Terms{Mode: "search", Query: "The Attic"}
	expect(t, s, http.MethodPost, search, http.StatusConflict,
		`{"status":409,"code":"no_indexer","title":"No Indexer is set up"}`)

	home, indexer := storage.NewUUID(), storage.NewUUID()
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO nodes_node VALUES (?, randomblob(32), 'home', 0)`, home); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', 'k', ?, '{}', 0)`, indexer, home)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	expect(t, s, http.MethodPost, search, http.StatusConflict,
		`{"status":409,"code":"no_root","title":"The Work's Content Type has no library root"}`)
	must[roots.Root](t)(roots.New(roots.Config{Library: lib, Types: types, Self: role.NodeID{UUID: home}, Changed: discard[roots.Changed]{}}).
		Create(ctx, admin, contenttypetest.ID, t.TempDir()))
	expect(t, s, http.MethodPost, "/api/v1/works/"+unmonitored.ID.String()+"/search", http.StatusConflict,
		`{"status":409,"code":"nothing_wanted","title":"The Work has no wanted cell to search for"}`)

	code, body := s.Call(t, http.MethodPost, search, "")
	run := ownertest.FirstID(body)
	if want := `{"searchIds":["` + run + `"]}` + "\n"; code != http.StatusAccepted || run == "" || body != want {
		t.Fatalf("searchWork = %d %s, want 202 %s", code, body, want)
	}
	item := tree.Items[0].ID.String()
	run1 := `{"id":"` + run + `","itemId":"` + item + `","indexerId":"` + indexer.String() + `","startedAt":"T",`
	expect(t, s, http.MethodGet, search+"es", http.StatusOK,
		`{"searches":[`+run1+`"finishedAt":null,"outcome":null,"candidates":0,"grabId":null}]}`)
	if got := <-searched; got != (grabs.SearchChanged{SearchID: run, WorkID: tree.ID.String()}) {
		t.Fatalf("grabs.search-changed = %+v, want the run %s", got, run)
	}

	runCtx, stop := context.WithCancel(ctx)
	ran := make(chan error)
	go func() { ran <- g.Run(runCtx) }()
	defer func() {
		stop()
		if err := <-ran; err != nil {
			t.Error(err)
		}
	}()
	<-searched
	grab := (<-changed).GrabID
	if failed := <-changed; failed.GrabID != grab || failed.WorkID != tree.ID.String() {
		t.Fatalf("grabs.changed = %+v, want the Grab %s failing", failed, grab)
	}
	expect(t, s, http.MethodGet, search+"es", http.StatusOK,
		`{"searches":[`+run1+`"finishedAt":"T","outcome":"grabbed","candidates":1,"grabId":"`+grab+`"}]}`)
	want := `{"id":"` + grab + `","workId":"` + tree.ID.String() + `","itemId":"` + item + `","edition":"standard","targetIds":["` + hd.ID.String() + `"],` +
		`"releaseTitle":"The.Attic.1080p.WEB","releaseSize":4096,"indexerId":"` + indexer.String() + `","nodeId":"` + home.String() + `","infoHash":null,` +
		`"stage":"failed","failure":{"stage":"fetching","code":"no_download_client","message":"the Indexer's home Node doesn't hold Downloading"},` +
		`"history":[{"stage":"fetching","at":"T"},{"stage":"failed","at":"T"}],"createdAt":"T"}`
	expect(t, s, http.MethodGet, "/api/v1/grabs/"+grab, http.StatusOK, want)
	expect(t, s, http.MethodGet, "/api/v1/grabs", http.StatusOK, `{"grabs":[`+want+`]}`)
}
