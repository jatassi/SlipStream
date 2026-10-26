package worksapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
	"github.com/jatassi/slipstream/internal/worksapi"
)

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

// recorder keeps every event published to it.
type recorder[T any] struct {
	mu   sync.Mutex
	sent []T
}

func (r *recorder[T]) Publish(_ context.Context, v T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, v)
}

// take is what was published since the last take.
func (r *recorder[T]) take() []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.sent
	r.sent = nil
	return out
}

type shelf struct {
	w *works.Works
	t *targets.Targets
	l *library.Library
}

func (s shelf) Works() (*works.Works, bool)       { return s.w, true }
func (s shelf) Targets() (*targets.Targets, bool) { return s.t, true }

func (shelf) Minter() (artwork.Minter, bool) { return minter, true }

func (s shelf) Library() (*library.Library, bool) { return s.l, true }

var minter = artwork.NewMinter([]byte("worksapi"))

// named describes every Content Type by its id.
type named struct{}

func (named) Describe(t works.TypeID) (works.Description, bool) {
	return works.Description{Name: string(t)}, true
}

type server struct {
	handler http.Handler
	lib     *storage.DB[storage.Library]
	fx      *contenttypetest.Fixture
	goals   *targets.Targets
	homes   *roots.Roots
	// targetsChanged and rootsChanged record targets.changed and
	// roots.changed.
	targetsChanged *recorder[targets.Changed]
	rootsChanged   *recorder[roots.Changed]
	apiKey         string
	session        string
	log            *bytes.Buffer
}

func serve(t *testing.T) *server {
	t.Helper()
	logged := &bytes.Buffer{}
	log := slog.New(slog.NewTextHandler(logged, &slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}}))
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	keys, err := secret.Open(t.Context(), lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := identity.New(t.Context(), identity.Config{Library: lib, Pepper: keys.PINPepper(), Hub: identitytest.Hub(t)})
	if err != nil {
		t.Fatal(err)
	}
	gate, code, err := identity.NewGate(t.Context(), identitytest.Node{Service: svc}.Claim, log)
	if err != nil {
		t.Fatal(err)
	}
	pin, err := identity.ParsePIN("2468")
	if err != nil {
		t.Fatal(err)
	}
	var admin identity.Claimed
	err = gate.Run(t.Context(), identity.Caller{}, code, func(c identity.Claim) {
		admin, err = svc.ClaimController(t.Context(), c, "ada@example.test", "Ada", pin)
	})
	if err != nil {
		t.Fatal(err)
	}
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library:      lib,
		Catalogs:     map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Source:       fx.Source(),
		Artwork:      fx,
		Timings:      works.Timings{Fetch: 5 * time.Second},
		Changed:      discard[works.Changed]{},
		TypesChanged: discard[works.ContentTypesChanged]{},
		TypeChanged:  discard[works.TypeChanged]{},
		Log:          log,
	})
	if err != nil {
		t.Fatal(err)
	}
	targetsChanged, rootsChanged := &recorder[targets.Changed]{}, &recorder[roots.Changed]{}
	goals, err := targets.Open(t.Context(), targets.Config{
		Library: lib, Works: named{}, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx, "other": fx}, Changed: targetsChanged,
	})
	if err != nil {
		t.Fatal(err)
	}
	rootsCfg := rootstest.Config(t, lib, contenttypetest.ID)
	rootsCfg.Changed = rootsChanged
	homes, err := roots.Open(t.Context(), rootsCfg)
	if err != nil {
		t.Fatal(err)
	}
	composer := library.New(library.Config{Library: lib, Works: w, Targets: goals, Roots: homes})
	r := api.NewRouter(api.Services{Node: identitytest.Controller{Node: identitytest.Node{Service: svc}}, Gate: gate, Web: fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}, Log: log})
	worksapi.Routes(r, shelf{w, goals, composer})
	return &server{handler: r.Handler(), lib: lib, fx: fx, goals: goals, homes: homes, targetsChanged: targetsChanged, rootsChanged: rootsChanged, apiKey: admin.APIKey.Reveal(), session: admin.Cookie.Value, log: logged}
}

func (s *server) call(t *testing.T, method, path, body string) (status int, reply string) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	req.Header.Set("X-Api-Key", s.apiKey)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	s.handler.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func expect(t *testing.T, s *server, method, path, body string, status int, want string) {
	t.Helper()
	code, got := s.call(t, method, path, body)
	if code != status || got != want+"\n" {
		t.Fatalf("%s %s = %d %s, want %d %s", method, path, code, got, status, want)
	}
}

var (
	uuids = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	times = regexp.MustCompile(`"\d{4}-\d{2}-\d{2}T[0-9:.]+Z"`)
)

func masked(body string) string {
	seen := map[string]string{}
	body = uuids.ReplaceAllStringFunc(body, func(id string) string {
		if _, ok := seen[id]; !ok {
			seen[id] = fmt.Sprintf("id%d", len(seen)+1)
		}
		return seen[id]
	})
	return times.ReplaceAllString(body, `"T"`)
}

func record() contenttypetest.Record {
	return contenttypetest.Record{
		Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
			{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
		}},
		WorkFields:  []works.FieldValue{works.TextField("origin", "Grandma")},
		LevelFields: map[works.Key][]works.FieldValue{"b1": {works.DateField("packed", works.Date{Year: 2024, Month: 11, Day: 2})}},
		ItemFields:  map[works.Key][]works.FieldValue{"c1": {works.NumberField("count", 2), works.DurationField("runtime", 90*time.Second)}},
	}
}

func TestAddWorkAnswersTheTreeWithItsFields(t *testing.T) {
	s := serve(t)
	s.fx.Put("w1", record())
	code, body := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1","monitor":true}`)
	want := `{"created":true,"work":{"id":"id1","contentType":"fixture","key":"w1","label":"The Attic","sortLabel":"The Attic","addedAt":"T","refreshedAt":"T","goneUpstreamAt":null,` +
		`"fields":[{"key":"origin","kind":"text","text":"Grandma"}],"ratings":[],"artwork":[],` +
		`"levels":[{"id":"id2","parentId":null,"depth":1,"key":"b1","label":"Winter","position":1,"withdrawnAt":null,"fields":[{"key":"packed","kind":"date","date":"2024-11-02"}]}],` +
		`"items":[{"id":"id3","levelId":"id2","key":"c1","label":"Scarf","position":1,"withdrawnAt":null,"monitored":true,"fields":[{"key":"count","kind":"number","number":2},{"key":"runtime","kind":"duration","ms":90000}],"ratings":[],"artwork":[],` +
		`"availability":{"date":"2000-01-01","basis":""}}],"links":[],"profile":{"id":"id4","name":"fixture"},"root":null}}` + "\n"
	if code != http.StatusOK || masked(body) != want {
		t.Fatalf("addWork = %d\n%s\nwant\n%s", code, masked(body), want)
	}
	id := uuids.FindString(body)
	code, again := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`)
	if code != http.StatusOK || masked(again) != strings.Replace(want, `"created":true`, `"created":false`, 1) {
		t.Fatalf("addWork again = %d %s", code, masked(again))
	}
	code, list := s.call(t, http.MethodGet, "/api/v1/works?contentType=fixture", "")
	if want := `{"works":[{"id":"` + id + `","contentType":"fixture","key":"w1","label":"The Attic","sortLabel":"The Attic","addedAt":"T","refreshedAt":"T","goneUpstreamAt":null,"fields":[{"key":"origin","kind":"text","text":"Grandma"}],"artwork":[]}]}` + "\n"; code != http.StatusOK || times.ReplaceAllString(list, `"T"`) != want {
		t.Fatalf("listWorks = %d %s, want %s", code, list, want)
	}
	code, search := s.call(t, http.MethodGet, "/api/v1/content-types/fixture/catalog?query=attic", "")
	if want := `{"candidates":[{"key":"w1","label":"The Attic","original":"","overview":"","fields":[{"key":"origin","kind":"text","text":"Grandma"}],"posterUrl":null,"workId":"` + id + `"}]}` + "\n"; code != http.StatusOK || search != want {
		t.Fatalf("searchCatalog = %d %s, want %s", code, search, want)
	}
}

func TestPreviewCatalogRecordAnswersTheAddedTreeAndAddsNothing(t *testing.T) {
	s := serve(t)
	s.fx.Put("w1", record())
	tree := `{"id":"id1","contentType":"fixture","key":"w1","label":"The Attic","sortLabel":"The Attic","addedAt":"T","refreshedAt":"T","goneUpstreamAt":null,` +
		`"fields":[{"key":"origin","kind":"text","text":"Grandma"}],"ratings":[],"artwork":[],` +
		`"levels":[{"id":"id2","parentId":null,"depth":1,"key":"b1","label":"Winter","position":1,"withdrawnAt":null,"fields":[{"key":"packed","kind":"date","date":"2024-11-02"}]}],` +
		`"items":[{"id":"id3","levelId":"id2","key":"c1","label":"Scarf","position":1,"withdrawnAt":null,"monitored":false,"fields":[{"key":"count","kind":"number","number":2},{"key":"runtime","kind":"duration","ms":90000}],"ratings":[],"artwork":[],"availability":{"date":"2000-01-01","basis":""}}]}`
	code, body := s.call(t, http.MethodGet, "/api/v1/content-types/fixture/catalog/w1", "")
	if want := `{"inLibrary":false,"work":` + tree + "}\n"; code != http.StatusOK || masked(body) != want {
		t.Fatalf("previewCatalogRecord = %d\n%s\nwant\n%s", code, masked(body), want)
	}
	expect(t, s, http.MethodGet, "/api/v1/works", "", 200, `{"works":[]}`)
	expect(t, s, http.MethodGet, "/api/v1/content-types/fixture/catalog/w2", "", 404, `{"status":404,"code":"metadata_not_found","title":"The metadata source has no record with that key"}`)

	code, body = s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1","monitor":false}`)
	if code != http.StatusOK {
		t.Fatalf("addWork = %d %s", code, body)
	}
	id := uuids.FindString(body)
	code, body = s.call(t, http.MethodGet, "/api/v1/content-types/fixture/catalog/w1", "")
	if code != http.StatusOK || !strings.HasPrefix(body, `{"inLibrary":true,"work":{"id":"`+id+`"`) || masked(body) != `{"inLibrary":true,"work":`+tree+"}\n" {
		t.Fatalf("previewCatalogRecord once added = %d %s, want the Work %s", code, body, id)
	}
}

// fixtureDescriptors is the fixture Content Type's wire from its fields on.
const fixtureDescriptors = `"fields":[` +
	`{"key":"origin","label":"Origin","tier":"work","kind":"text","in":["facts","about"],"list":{"column":true,"sort":true,"filter":true},"with":null,"art":null},` +
	`{"key":"packed","label":"Packed","tier":"level","depth":1,"kind":"date","in":["facts"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"count","label":"Count","tier":"item","kind":"number","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"runtime","label":"Runtime","tier":"item","kind":"duration","in":["facts"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"tags","label":"Tags","tier":"work","kind":"texts","in":["about"],"list":{"column":true,"sort":false,"filter":true},"with":null,"art":null},` +
	`{"key":"makers","label":"Makers","tier":"work","kind":"people","in":["credits"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"page","label":"Page","tier":"work","kind":"link","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null}],` +
	`"artwork":[{"key":"photo","label":"Photo","tier":"work","shape":"poster"},{"key":"snap","label":"Snap","tier":"item","shape":"backdrop"}],` +
	`"source":"Fixture source","items":"many","art":{"poster":"photo","backdrop":null,"title":null},"ratings":["tmdb","imdb"]`

func TestListContentTypesAnswersASession(t *testing.T) {
	s := serve(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/content-types", http.NoBody)
	req.Header.Set("Cookie", identity.CookieName+"="+s.session)
	w := httptest.NewRecorder()
	s.handler.ServeHTTP(w, req)
	want := `{"contentTypes":[{"id":"fixture","name":"Fixture","icon":"shapes","enabled":true,"work":{"one":"Shelf","many":"Shelves"},"levels":[{"one":"Box","many":"Boxes"}],"item":{"one":"Thing","many":"Things"},` +
		fixtureDescriptors + `}]}` + "\n"
	if w.Code != http.StatusOK || w.Body.String() != want {
		t.Fatalf("listContentTypes = %d %s, want %s", w.Code, w.Body, want)
	}
	expect(t, s, http.MethodGet, "/api/v1/content-types", "", http.StatusForbidden, `{"status":403,"code":"needs_account","title":"Only a signed-in person can do this"}`)
}

func TestEachWorksErrorAnswersItsProblem(t *testing.T) {
	s := serve(t)
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"ghost","key":"w1"}`, 404, `{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`, 404, `{"status":404,"code":"metadata_not_found","title":"The metadata source has no record with that key"}`)
	s.fx.Fail("w1", fmt.Errorf("lookup: %w", works.ErrUnavailable))
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`, 502, `{"status":502,"code":"metadata_unavailable","title":"The metadata source didn't respond. Try again later."}`)
	s.fx.Fail("w1", nil)
	bad := record()
	bad.Tree.Children[0].Children = append(bad.Tree.Children[0].Children, works.Entry{Key: "c1", Label: "Again"})
	s.fx.Put("w1", bad)
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`, 502, `{"status":502,"code":"metadata_invalid","title":"The metadata source sent a record SlipStream can't use"}`)
	expect(t, s, http.MethodGet, "/api/v1/works?contentType=ghost", "", 404, `{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)
	expect(t, s, http.MethodGet, "/api/v1/works/"+storage.NewUUID().String(), "", 404, `{"status":404,"code":"work_not_found","title":"No Work has that id"}`)

	expect(t, s, http.MethodPut, "/api/v1/content-types/fixture/enabled", `{"enabled":false}`, 200,
		`{"id":"fixture","name":"Fixture","icon":"shapes","enabled":false,"work":{"one":"Shelf","many":"Shelves"},"levels":[{"one":"Box","many":"Boxes"}],"item":{"one":"Thing","many":"Things"},`+
			fixtureDescriptors+`}`)
	expect(t, s, http.MethodGet, "/api/v1/content-types/fixture/catalog?query=attic", "", 409, `{"status":409,"code":"content_type_disabled","title":"That Content Type is turned off"}`)
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`, 409, `{"status":409,"code":"content_type_disabled","title":"That Content Type is turned off"}`)
	if code, _ := s.call(t, http.MethodPut, "/api/v1/content-types/fixture/enabled", `{"enabled":true}`); code != 200 {
		t.Fatalf("turn on = %d", code)
	}

	s.fx.Put("w1", record())
	code, body := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`)
	if code != 200 {
		t.Fatalf("addWork = %d %s", code, body)
	}
	expect(t, s, http.MethodPut, "/api/v1/content-types/fixture/enabled", `{"enabled":false}`, 409, `{"status":409,"code":"content_type_in_use","title":"Remove its Works before turning this Content Type off"}`)
	s.fx.Fail("w2", &works.SourceError{Source: "TMDB", Words: "Invalid API key", Err: works.ErrRefused})
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w2"}`, 502, `{"status":502,"code":"metadata_refused","title":"The metadata source refused SlipStream's key","detail":"TMDB refused SlipStream's key"}`)
	s.fx.Fail("w2", &works.SourceError{Source: "TMDB", Words: "is rate limited until 2026-10-19T21:04:00Z", Err: works.ErrUnavailable})
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w2"}`, 502, `{"status":502,"code":"metadata_unavailable","title":"The metadata source didn't respond. Try again later.","detail":"TMDB: is rate limited until 2026-10-19T21:04:00Z"}`)
}

func TestRefreshByHandAnswersGoneUpstream409(t *testing.T) {
	s := serve(t)
	s.fx.Put("w1", record())
	code, body := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`)
	if code != 200 {
		t.Fatalf("addWork = %d %s", code, body)
	}
	work := "/api/v1/works/" + uuids.FindString(body)
	s.fx.Fail("w1", fmt.Errorf("lookup: %w", works.ErrNotFound))
	gone := `{"status":409,"code":"work_gone_upstream","title":"The metadata source no longer has this Work"}`
	expect(t, s, http.MethodPost, work+"/refresh", "", 409, gone)
	expect(t, s, http.MethodPost, work+"/refresh", "", 409, gone)
	code, body = s.call(t, http.MethodGet, work, "")
	if code != 200 || strings.Contains(body, `"goneUpstreamAt":null`) || !strings.Contains(body, `"goneUpstreamAt":"`) {
		t.Fatalf("getWork after the source lost it = %d %s, want goneUpstreamAt set", code, body)
	}
	s.fx.Fail("w1", nil)
	code, body = s.call(t, http.MethodPost, work+"/refresh", "")
	if code != 200 || !strings.Contains(body, `"goneUpstreamAt":null`) {
		t.Fatalf("refreshWork once the source has it again = %d %s, want goneUpstreamAt null", code, body)
	}
}

var ada = artwork.Ref{Source: "fixture", Shape: artwork.ShapePhoto, Path: "/ada.jpg"}

func TestFieldKindsTextsPeopleLinkRoundTripTheAPI(t *testing.T) {
	s := serve(t)
	r := record()
	r.WorkFields = append(r.WorkFields,
		works.TextsField("tags", []string{"wool", "winter"}),
		works.PeopleField("makers", []works.Person{{Name: "Ada", Role: "Knitter", Department: "Craft", Photo: ada}, {Name: "Grace"}}),
		works.LinkField("page", works.Link{Label: "Shop", URL: "https://example.test/attic"}),
	)
	votes := int64(1200)
	r.Tree.Ratings = []works.Rating{{Source: "tmdb", Value: 7.5, Votes: &votes}, {Source: "rottentomatoes", Value: 91}}
	s.fx.Put("w1", r)
	code, body := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`)
	if code != 200 {
		t.Fatalf("addWork = %d %s", code, body)
	}
	code, body = s.call(t, http.MethodGet, "/api/v1/works/"+uuids.FindString(body), "")
	want := `"fields":[{"key":"origin","kind":"text","text":"Grandma"},{"key":"tags","kind":"texts","texts":["wool","winter"]},` +
		`{"key":"makers","kind":"people","people":[{"name":"Ada","role":"Knitter","department":"Craft","photoUrl":"` + minter.URL(ada) + `"},{"name":"Grace","role":"","department":"","photoUrl":null}]},` +
		`{"key":"page","kind":"link","label":"Shop","url":"https://example.test/attic"}],` +
		`"ratings":[{"source":"rottentomatoes","label":"Rotten Tomatoes critics","value":91,"scale":100,"votes":null,"fetchedAt":"T"},` +
		`{"source":"tmdb","label":"TMDB","value":7.5,"scale":10,"votes":1200,"fetchedAt":"T"}],"artwork":[],"levels"`
	if code != 200 || !strings.Contains(masked(body), want) {
		t.Fatalf("getWork = %d\n%s\nwant it to hold\n%s", code, masked(body), want)
	}
}

func TestTheWorkAPIShowsDateAndBasis(t *testing.T) {
	s := serve(t)
	s.fx.Put("w1", record())
	code, body := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`)
	if code != 200 {
		t.Fatalf("addWork = %d %s", code, body)
	}
	var added struct {
		Work struct {
			ID    string `json:"id"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"work"`
	}
	if err := json.Unmarshal([]byte(body), &added); err != nil {
		t.Fatal(err)
	}
	item, err := storage.ParseUUID(added.Work.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	work := "/api/v1/works/" + added.Work.ID
	s.fx.SetTiming(works.ItemID{UUID: item}, targets.Timing{Released: release.Known(works.Date{Year: 2026, Month: 11, Day: 3}), Basis: "Digital release in United States"})
	for _, call := range []struct{ method, path string }{{http.MethodGet, work}, {http.MethodPost, work + "/refresh"}} {
		code, body = s.call(t, call.method, call.path, "")
		if want := `"availability":{"date":"2026-11-03","basis":"Digital release in United States"}`; code != 200 || !strings.Contains(body, want) {
			t.Fatalf("%s %s = %d %s, want it to hold %s", call.method, call.path, code, body, want)
		}
	}
	s.fx.SetTiming(works.ItemID{UUID: item}, targets.Timing{})
	if code, body = s.call(t, http.MethodGet, work, ""); code != 200 || !strings.Contains(body, `"availability":null`) {
		t.Fatalf("getWork with no date = %d %s, want availability null", code, body)
	}
}

func TestTypeSettingsRoundTripTheAPI(t *testing.T) {
	s := serve(t)
	settings := func(monitor string, depth int, isDefault bool) string {
		return `{"settings":[` + monitor + fmt.Sprintf(`,{"key":"fixture.depth","label":"Boxes deep","source":"database","applies":"live","isDefault":%t,"pendingRestart":false,`+
			`"kind":"count","value":%d,"running":%d,"default":3,"min":1,"max":9,"unit":{"one":"box","many":"boxes"}}]}`, isDefault, depth, depth)
	}
	on, off := addMonitorField(true, true), addMonitorField(false, false)
	path := "/api/v1/content-types/fixture/settings"
	expect(t, s, http.MethodGet, path, "", 200, settings(on, 3, true))
	expect(t, s, http.MethodPatch, path, `{"values":{"fixture.depth":5}}`, 200, settings(on, 5, false))
	expect(t, s, http.MethodPatch, path, `{"values":{"fixture.depth":12}}`, 422,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 1 and 9 boxes."}`)
	expect(t, s, http.MethodPatch, path, `{"values":{"fixture.width":2}}`, 422,
		`{"status":422,"code":"unknown_setting","title":"No setting has this key","detail":"No setting has the key fixture.width."}`)
	expect(t, s, http.MethodPatch, path, `{"values":{"fixture.add.monitor":false,"fixture.width":2}}`, 422,
		`{"status":422,"code":"unknown_setting","title":"No setting has this key","detail":"No setting has the key fixture.width."}`)
	expect(t, s, http.MethodGet, path, "", 200, settings(on, 5, false))
	expect(t, s, http.MethodPatch, path, `{"values":{"fixture.add.monitor":false,"fixture.depth":null}}`, 200, settings(off, 3, true))
	expect(t, s, http.MethodPatch, path, `{"values":{"fixture.add.monitor":null}}`, 200, settings(on, 3, true))
	expect(t, s, http.MethodGet, "/api/v1/content-types/ghost/settings", "", 404, `{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)
	expect(t, s, http.MethodPatch, "/api/v1/content-types/ghost/settings", `{"values":{}}`, 404, `{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)
}

// addMonitorField is fixture.add.monitor on the wire.
func addMonitorField(value, isDefault bool) string {
	return fmt.Sprintf(`{"key":"fixture.add.monitor","label":"New shelves are monitored","source":"database","applies":"live","isDefault":%t,"pendingRestart":false,"kind":"switch","value":%t,"running":%t,"default":true}`, isDefault, value, value)
}

func TestAPlainSourceErrorAnswersInternalAndLogsIt(t *testing.T) {
	s := serve(t)
	s.fx.Fail("w1", errors.New("disk on fire"))
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`, 500, `{"status":500,"code":"internal","title":"SlipStream had an internal error. This Node's log has the details."}`)
	if want := `level=ERROR msg="operation failed" method=POST path=/api/v1/works operation=addWork error="content type fixture: fetch w1: disk on fire"`; !strings.Contains(s.log.String(), want) {
		t.Fatalf("log = %s, want %s", s.log, want)
	}
}

func TestAddWorkWantsFromTheProfileItNamesOrTheTypesDefault(t *testing.T) {
	ctx := t.Context()
	s := serve(t)
	admin := identity.Admin{}
	profiles, err := s.goals.ListProfiles(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	fixture, other := profiles[0], profiles[1]
	small, err := s.goals.CreateProfile(ctx, admin, targets.ProfileSpec{Name: "Small", ContentType: contenttypetest.ID})
	if err != nil {
		t.Fatal(err)
	}
	s.fx.Put("w1", record())
	s.fx.Put("w2", record())
	add := func(key, profile string) string {
		return `{"contentType":"fixture","key":"` + key + `","profileId":"` + profile + `"}`
	}

	expect(t, s, http.MethodPost, "/api/v1/works", add("w1", other.ID.String()), 422,
		`{"status":422,"code":"profile_wrong_type","title":"The Profile is for another Content Type"}`)
	expect(t, s, http.MethodPost, "/api/v1/works", add("w1", storage.NewUUID().String()), 422,
		`{"status":422,"code":"profile_not_found","title":"No Profile has that id"}`)
	expect(t, s, http.MethodGet, "/api/v1/works", "", 200, `{"works":[]}`)

	wants := func(body string) targets.Goals {
		t.Helper()
		id, err := storage.ParseUUID(uuids.FindString(body))
		if err != nil {
			t.Fatal(err)
		}
		g, err := s.goals.Goals(ctx, admin, works.WorkID{UUID: id})
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	code, body := s.call(t, http.MethodPost, "/api/v1/works", add("w1", small.ID.String()))
	if g := wants(body); code != 200 || g.Profile.ID != small.ID {
		t.Fatalf("addWork with Small = %d, wants from %s; want Small", code, g.Profile.Name)
	}
	code, body = s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w2"}`)
	if g := wants(body); code != 200 || g.Profile.ID != fixture.ID {
		t.Fatalf("addWork with no Profile = %d, wants from %s; want the default %s", code, g.Profile.Name, fixture.Name)
	}
}

func TestArtworkAnswersAsURLsTheRouteOpens(t *testing.T) {
	s := serve(t)
	photo := artwork.Ref{Source: "fixture", Shape: artwork.ShapePoster, Path: "/attic.jpg"}
	snap := artwork.Ref{Source: "fixture", Shape: artwork.ShapeBackdrop, Path: "/scarf.jpg"}
	r := record()
	r.Tree.Artwork = []works.Art{{Kind: "photo", Ref: photo}, {Item: "c1", Kind: "snap", Ref: snap}}
	s.fx.Put("w1", r)
	opens := func(what, kind, url string, want artwork.Ref) {
		t.Helper()
		token, ok := strings.CutPrefix(url, artwork.Route)
		got, err := minter.Open(token)
		if !ok || err != nil || got != want {
			t.Errorf("%s's %s URL %q opens to %+v, %v; want %+v", what, kind, url, got, err, want)
		}
	}
	for _, at := range []struct{ op, method, path, body string }{
		{"previewCatalogRecord", http.MethodGet, "/api/v1/content-types/fixture/catalog/w1", ""},
		{"addWork", http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`},
	} {
		code, body := s.call(t, at.method, at.path, at.body)
		var tree struct {
			Work struct {
				Artwork []struct{ Kind, URL string }
				Items   []struct {
					Artwork []struct{ Kind, URL string }
				}
			}
		}
		if err := json.Unmarshal([]byte(body), &tree); code != http.StatusOK || err != nil || len(tree.Work.Artwork) != 1 || len(tree.Work.Items) != 1 || len(tree.Work.Items[0].Artwork) != 1 {
			t.Fatalf("%s = %d %s, %v; want one Work image and one Item image", at.op, code, body, err)
		}
		opens(at.op+"'s Work", tree.Work.Artwork[0].Kind, tree.Work.Artwork[0].URL, photo)
		opens(at.op+"'s Item", tree.Work.Items[0].Artwork[0].Kind, tree.Work.Items[0].Artwork[0].URL, snap)
	}

	code, list := s.call(t, http.MethodGet, "/api/v1/works", "")
	if want := `"artwork":[{"kind":"photo","url":"` + minter.URL(photo) + `"}]`; code != http.StatusOK || !strings.Contains(list, want) || strings.Contains(list, "scarf") {
		t.Fatalf("listWorks = %d %s, want the Work's own image %s and no Item's", code, list, want)
	}
}

// The literal routes beside /works/{workId} reach their own stubs.
func TestEachLibraryRouteReachesItsOwnHandler(t *testing.T) {
	s := serve(t)
	ids := `{"workIds":["` + storage.NewUUID().String() + `"]`
	stub := `{"status":501,"code":"not_implemented","title":"This route isn't built yet"}`
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/works/search?q=heat", ""},
		{http.MethodGet, "/api/v1/library?contentType=fixture", ""},
		{http.MethodPut, "/api/v1/works/monitored", ids + `,"on":true}`},
		{http.MethodPut, "/api/v1/works/profile?preview=true", ids + `,"profileId":"` + storage.NewUUID().String() + `"}`},
		{http.MethodPut, "/api/v1/works/root", ids + `,"rootId":"` + storage.NewUUID().String() + `"}`},
		{http.MethodPost, "/api/v1/works/refresh", ids + `}`},
		{http.MethodPost, "/api/v1/works/delete", ids + `,"files":"leave","preview":true}`},
	} {
		expect(t, s, c.method, c.path, c.body, http.StatusNotImplemented, stub)
	}
	expect(t, s, http.MethodGet, "/api/v1/library?contentType=fixture&workIds="+storage.NewUUID().String(), "", http.StatusUnprocessableEntity,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"query.workIds","message":"Use contentType or workIds, not both."}]}`)
}
