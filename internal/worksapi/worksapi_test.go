package worksapi_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
	"github.com/jatassi/slipstream/internal/worksapi"
)

type quiet struct{}

func (quiet) Publish(context.Context, storage.UUID, identity.SessionsChanged) {}

func (quiet) EndSession(storage.UUID) {}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type shelf struct{ w *works.Works }

func (s shelf) Works() (*works.Works, bool) { return s.w, true }

type server struct {
	handler http.Handler
	fx      *contenttypetest.Fixture
	apiKey  string
	session string
	log     *bytes.Buffer
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
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), works.Schema())
	keys, err := secret.Open(t.Context(), lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := identity.New(t.Context(), lib, keys.PINPepper(), quiet{}, quiet{})
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
		Timings:      works.Timings{Fetch: 5 * time.Second},
		Changed:      discard[works.WorkChanged]{},
		TypesChanged: discard[works.ContentTypesChanged]{},
		Log:          log,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{Node: identitytest.Controller{Node: identitytest.Node{Service: svc}}, Gate: gate, Web: fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}, Log: log})
	worksapi.Routes(r, shelf{w})
	return &server{handler: r.Handler(), fx: fx, apiKey: admin.APIKey, session: admin.Cookie.Value, log: logged}
}

func (s *server) call(t *testing.T, method, path, body string) (int, string) {
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

// masked names each id by its order of first appearance and hides times.
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
	code, body := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`)
	want := `{"created":true,"work":{"id":"id1","contentType":"fixture","key":"w1","label":"The Attic","sortLabel":"The Attic","addedAt":"T","refreshedAt":"T",` +
		`"fields":[{"key":"origin","kind":"text","text":"Grandma"}],` +
		`"levels":[{"id":"id2","parentId":null,"depth":1,"key":"b1","label":"Winter","position":1,"withdrawnAt":null,"fields":[{"key":"packed","kind":"date","date":"2024-11-02"}]}],` +
		`"items":[{"id":"id3","levelId":"id2","key":"c1","label":"Scarf","position":1,"withdrawnAt":null,"fields":[{"key":"count","kind":"number","number":2},{"key":"runtime","kind":"duration","ms":90000}]}]}}` + "\n"
	if code != http.StatusOK || masked(body) != want {
		t.Fatalf("addWork = %d\n%s\nwant\n%s", code, masked(body), want)
	}
	id := uuids.FindString(body)
	code, again := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`)
	if code != http.StatusOK || masked(again) != strings.Replace(want, `"created":true`, `"created":false`, 1) {
		t.Fatalf("addWork again = %d %s", code, masked(again))
	}
	code, list := s.call(t, http.MethodGet, "/api/v1/works?contentType=fixture", "")
	if want := `{"works":[{"id":"` + id + `","contentType":"fixture","key":"w1","label":"The Attic","sortLabel":"The Attic","addedAt":"T","refreshedAt":"T"}]}` + "\n"; code != http.StatusOK || times.ReplaceAllString(list, `"T"`) != want {
		t.Fatalf("listWorks = %d %s, want %s", code, list, want)
	}
	code, search := s.call(t, http.MethodGet, "/api/v1/content-types/fixture/catalog?query=attic", "")
	if want := `{"candidates":[{"key":"w1","label":"The Attic","fields":[{"key":"origin","kind":"text","text":"Grandma"}],"workId":"` + id + `"}]}` + "\n"; code != http.StatusOK || search != want {
		t.Fatalf("searchCatalog = %d %s, want %s", code, search, want)
	}
}

func TestListContentTypesAnswersASession(t *testing.T) {
	s := serve(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/content-types", http.NoBody)
	req.AddCookie(&http.Cookie{Name: identity.CookieName, Value: s.session})
	w := httptest.NewRecorder()
	s.handler.ServeHTTP(w, req)
	want := `{"contentTypes":[{"id":"fixture","name":"Fixture","enabled":true,"work":{"one":"Shelf","many":"Shelves"},"levels":[{"one":"Box","many":"Boxes"}],"item":{"one":"Thing","many":"Things"},` +
		`"fields":[{"key":"origin","label":"Origin","tier":"work","kind":"text"},{"key":"packed","label":"Packed","tier":"level","depth":1,"kind":"date"},` +
		`{"key":"count","label":"Count","tier":"item","kind":"number"},{"key":"runtime","label":"Runtime","tier":"item","kind":"duration"}]}]}` + "\n"
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
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`, 502, `{"status":502,"code":"metadata_unavailable","title":"The metadata source didn't answer. Try again later."}`)
	s.fx.Fail("w1", nil)
	bad := record()
	bad.Tree.Children[0].Children = append(bad.Tree.Children[0].Children, works.Entry{Key: "c1", Label: "Again"})
	s.fx.Put("w1", bad)
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`, 502, `{"status":502,"code":"metadata_invalid","title":"The metadata source sent a record SlipStream can't use"}`)
	expect(t, s, http.MethodGet, "/api/v1/works?contentType=ghost", "", 404, `{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)
	expect(t, s, http.MethodGet, "/api/v1/works/"+storage.NewUUID().String(), "", 404, `{"status":404,"code":"work_not_found","title":"No Work has that id"}`)

	expect(t, s, http.MethodPut, "/api/v1/content-types/fixture/enabled", `{"enabled":false}`, 200,
		`{"id":"fixture","name":"Fixture","enabled":false,"work":{"one":"Shelf","many":"Shelves"},"levels":[{"one":"Box","many":"Boxes"}],"item":{"one":"Thing","many":"Things"},`+
			`"fields":[{"key":"origin","label":"Origin","tier":"work","kind":"text"},{"key":"packed","label":"Packed","tier":"level","depth":1,"kind":"date"},`+
			`{"key":"count","label":"Count","tier":"item","kind":"number"},{"key":"runtime","label":"Runtime","tier":"item","kind":"duration"}]}`)
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
	s.fx.Fail("w1", fmt.Errorf("lookup: %w", works.ErrNotFound))
	expect(t, s, http.MethodPost, "/api/v1/works/"+uuids.FindString(body)+"/refresh", "", 409, `{"status":409,"code":"work_gone_upstream","title":"The metadata source no longer has this Work"}`)
}

func TestAPlainSourceErrorAnswersInternalAndLogsIt(t *testing.T) {
	s := serve(t)
	s.fx.Fail("w1", errors.New("disk on fire"))
	expect(t, s, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`, 500, `{"status":500,"code":"internal","title":"Something went wrong"}`)
	if want := `level=ERROR msg="operation failed" method=POST path=/api/v1/works operation=addWork error="content type fixture: fetch w1: disk on fire"`; !strings.Contains(s.log.String(), want) {
		t.Fatalf("log = %s, want %s", s.log, want)
	}
}
