package indexersapi_test

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexersapi"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/search/v1/searchv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type published struct{ ids []string }

func (p *published) Publish(_ context.Context, v indexers.Changed) {
	p.ids = append(p.ids, v.IndexerID)
}

type registry struct{ x *indexers.Indexers }

func (r registry) Indexers() (*indexers.Indexers, bool) { return r.x, true }

type roster struct {
	self    role.NodeID
	holders []role.NodeID
	away    map[role.NodeID]nodes.Absence
}

func (r roster) Self() role.NodeID { return r.self }

// Pick is prefer when it holds Searching, else the first holder.
func (r roster) Pick(_ role.Name, prefer role.NodeID) (role.NodeID, bool) {
	if slices.Contains(r.holders, prefer) {
		return prefer, true
	}
	if len(r.holders) == 0 {
		return role.NodeID{}, false
	}
	return r.holders[0], true
}

// Route waits for home while it is away within the grace, else picks.
func (r roster) Route(rn role.Name, home role.NodeID) (role.NodeID, error) {
	if a, ok := r.Away(rn, home); ok && !a.Lapsed {
		return role.NodeID{}, &nodes.Unrouted{Why: "home is " + a.Standing.String()}
	}
	to, ok := r.Pick(rn, home)
	if !ok {
		return role.NodeID{}, &nodes.Unrouted{Why: "no Node searches"}
	}
	return to, nil
}

func (r roster) Away(_ role.Name, node role.NodeID) (nodes.Absence, bool) {
	a, ok := r.away[node]
	return a, ok
}

// disconnected is a home Node whose link is down.
type disconnected struct {
	searchv1connect.UnimplementedSearchingServiceHandler
}

func (disconnected) Caps(context.Context, *searchv1.CapsRequest) (*searchv1.CapsResponse, error) {
	return nil, link.ErrDisconnected
}

var (
	self   = role.NodeID{UUID: storage.NewUUID()}
	worker = role.NodeID{UUID: storage.NewUUID()}
)

type served struct {
	owner   *ownertest.Owner
	site    faketest.Served
	key     string
	changed *published
	asked   []role.NodeID
}

// serve serves the operations over a fixture-site every holder reaches but
// down, whose link is disconnected.
func serve(t *testing.T, holders []role.NodeID, down role.NodeID) *served {
	t.Helper()
	return serveOn(t, fixtureSite(t), storagetest.Library(t, secret.Schema(), identity.Schema(), indexers.Schema()), roster{self: self, holders: holders}, down)
}

var fixtureScenario = scenario.Scenario{Name: "x", Seed: "indexersapi"}

func fixtureSite(t *testing.T) faketest.Served {
	t.Helper()
	return faketest.Loopback(t, site.Fake, fixtureScenario)
}

func serveOn(t *testing.T, fixture faketest.Served, lib *storage.DB[storage.Library], r roster, down role.NodeID) *served {
	t.Helper()
	srv := &served{key: site.APIKey(fixtureScenario), changed: &published{}, site: fixture}
	home := indexers.NewHome(srv.site.Client, indexers.Timings{Request: 5 * time.Second})
	x := indexers.New(indexers.Config{
		Library: lib,
		Sealer:  secrettest.Sealer(t),
		Roster:  r,
		Searching: func(n role.NodeID) searchv1connect.SearchingServiceHandler {
			srv.asked = append(srv.asked, n)
			if n == down {
				return disconnected{}
			}
			return home
		},
		Changed: srv.changed,
	})
	srv.owner = ownertest.Serve(t, lib, func(r *api.Router) { indexersapi.Routes(r, registry{x}) })
	return srv
}

func (s *served) body(path, key string) string {
	return `{"name":"Fixture Site","url":` + strconv.Quote(s.site.Address.URL(path)) + `,"apiKey":"` + key + `"}`
}

func TestAnIndexerIsCreatedOnItsHomeNodeAndListed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		holders []role.NodeID
		home    role.NodeID
	}{
		{"this Node holds Searching", []role.NodeID{worker, self}, self},
		{"a Worker holds Searching", []role.NodeID{worker}, worker},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := serve(t, tc.holders, role.NodeID{})
			url := s.site.Address.URL(site.TorznabPath)
			modes := `"searchModes":[{"mode":"movie","params":["q","tmdbid","imdbid"]},{"mode":"search","params":["q"]}]`
			want := `{"id":"id1","name":"Fixture Site","protocol":"torznab","url":"` + url + `","homeNodeId":"id2","apiKeySet":true,` + modes + `,"waiting":null,"createdAt":"T"}`
			s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", s.body(site.TorznabPath, s.key), http.StatusCreated, want)
			_, list := s.owner.Call(t, http.MethodGet, "/api/v1/indexers", "")
			id := ownertest.FirstID(list)
			if want := `{"indexers":[{"id":"` + id + `","name":"Fixture Site","protocol":"torznab","url":"` + url + `","homeNodeId":"` + tc.home.String() + `","apiKeySet":true,` + modes + `,"waiting":null,"createdAt":"T"}]}` + "\n"; ownertest.Masked(list) != ownertest.Masked(want) || !strings.Contains(list, `"homeNodeId":"`+tc.home.String()+`"`) {
				t.Fatalf("listIndexers = %s, want %s", list, want)
			}
			if len(s.changed.ids) != 1 || s.changed.ids[0] != id {
				t.Fatalf("indexers.changed published %v, want [%s]", s.changed.ids, id)
			}
			if len(s.asked) != 1 || s.asked[0] != tc.home {
				t.Fatalf("asked %v for caps, want the home Node %s", s.asked, tc.home)
			}
		})
	}
}

func TestEachIndexersErrorAnswersItsProblem(t *testing.T) {
	s := serve(t, []role.NodeID{self}, role.NodeID{})
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", s.body(site.TorznabPath, "wrong"), http.StatusUnprocessableEntity,
		`{"status":422,"code":"indexer_refused","title":"The Indexer refused the request","detail":"error 100: Invalid API Key"}`)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", s.body("/elsewhere", s.key), http.StatusBadGateway,
		`{"status":502,"code":"indexer_unavailable","title":"The Indexer didn't respond","detail":"The Indexer responded with status 404."}`)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", `{"name":"Fixture Site","url":"ftp://example.test/torznab","apiKey":"k"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"body.url","message":"expected string to match pattern ^https?://[^/?#]+(/[^?#]*)?$"}]}`)
	if code, reply := s.owner.Call(t, http.MethodPost, "/api/v1/indexers", s.body(site.TorznabPath, s.key)); code != http.StatusCreated {
		t.Fatalf("createIndexer = %d %s, want 201", code, reply)
	}
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", s.body(site.TorznabPath, s.key), http.StatusConflict,
		`{"status":409,"code":"indexer_name_taken","title":"Another Indexer has that name"}`)
	if len(s.changed.ids) != 1 {
		t.Fatalf("indexers.changed published %d times, want once for the one Indexer made", len(s.changed.ids))
	}

	none := serve(t, nil, role.NodeID{})
	none.owner.Expect(t, http.MethodPost, "/api/v1/indexers", none.body(site.TorznabPath, none.key), http.StatusConflict,
		`{"status":409,"code":"no_searching_node","title":"No Node holds Searching. Give one the Searching Role first."}`)

	down := serve(t, []role.NodeID{worker}, worker)
	down.owner.Expect(t, http.MethodPost, "/api/v1/indexers", down.body(site.TorznabPath, down.key), http.StatusBadGateway,
		`{"status":502,"code":"indexer_unavailable","title":"The Indexer didn't respond","detail":"No live link session to the peer."}`)
	for _, x := range []*served{none, down} {
		if _, list := x.owner.Call(t, http.MethodGet, "/api/v1/indexers", ""); list != `{"indexers":[]}`+"\n" || len(x.changed.ids) != 0 {
			t.Fatalf("after a failed create the list is %s with %d events, want none", list, len(x.changed.ids))
		}
	}
}

func TestAKeyTheSealingMigrationDroppedIsEnteredAgain(t *testing.T) {
	ctx := t.Context()
	fixture := fixtureSite(t)
	url := fixture.Address.URL(site.TorznabPath)
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, schemas, func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil }, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	before := open(indexers.Schema().Through(1))
	id := storage.NewUUID()
	err := before.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', ?, 'plain key', ?, '{}', 0)`, id, url, self)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := before.Close(); err != nil {
		t.Fatal(err)
	}
	lib := open(secret.Schema(), identity.Schema(), indexers.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	s := serveOn(t, fixture, lib, roster{self: self, holders: []role.NodeID{self}}, role.NodeID{})
	row := func(set bool, modes string) string {
		return `{"id":"id1","name":"Fixture Site","protocol":"torznab","url":"` + url + `","homeNodeId":"id2","apiKeySet":` + strconv.FormatBool(set) + `,"searchModes":[` + modes + `],"waiting":null,"createdAt":"T"}`
	}
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexers", "", http.StatusOK, `{"indexers":[`+row(false, "")+`]}`)
	s.owner.Expect(t, http.MethodPut, "/api/v1/indexers/"+storage.NewUUID().String()+"/api-key", `{"apiKey":"k"}`, http.StatusNotFound,
		`{"status":404,"code":"indexer_not_found","title":"No Indexer has that id"}`)
	put := "/api/v1/indexers/" + id.String() + "/api-key"
	s.owner.Expect(t, http.MethodPut, put, `{"apiKey":"wrong"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"indexer_refused","title":"The Indexer refused the request","detail":"error 100: Invalid API Key"}`)
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexers", "", http.StatusOK, `{"indexers":[`+row(false, "")+`]}`)
	modes := `{"mode":"movie","params":["q","tmdbid","imdbid"]},{"mode":"search","params":["q"]}`
	s.owner.Expect(t, http.MethodPut, put, `{"apiKey":"`+s.key+`"}`, http.StatusOK, row(true, modes))
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexers", "", http.StatusOK, `{"indexers":[`+row(true, modes)+`]}`)
	if len(s.changed.ids) != 1 || s.changed.ids[0] != id.String() {
		t.Fatalf("indexers.changed published %v, want [%s]", s.changed.ids, id)
	}
}

func TestAnIndexerSaysItWaitsForItsHomeUntilTheGrace(t *testing.T) {
	away := map[role.NodeID]nodes.Absence{}
	s := serveOn(t, fixtureSite(t), storagetest.Library(t, secret.Schema(), identity.Schema(), indexers.Schema()),
		roster{self: self, holders: []role.NodeID{self, worker}, away: away}, role.NodeID{})
	if code, reply := s.owner.Call(t, http.MethodPost, "/api/v1/indexers", s.body(site.TorznabPath, s.key)); code != http.StatusCreated {
		t.Fatalf("createIndexer = %d %s, want 201", code, reply)
	}
	since := time.Date(2026, 10, 17, 21, 31, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		absence nodes.Absence
		waiting string
	}{
		{"offline within the grace", nodes.Absence{Since: since, Standing: nodes.Offline}, `{"standing":"offline","since":"2026-10-17T21:31:00Z"}`},
		{"reconnecting", nodes.Absence{Since: since, Standing: nodes.Reconnecting}, `{"standing":"reconnecting","since":"2026-10-17T21:31:00Z"}`},
		{"lapsed", nodes.Absence{Since: since, Standing: nodes.LapsedNode, Lapsed: true}, "null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			away[self] = tc.absence
			if _, list := s.owner.Call(t, http.MethodGet, "/api/v1/indexers", ""); !strings.Contains(list, `"waiting":`+tc.waiting+`,`) {
				t.Fatalf("listIndexers = %s, want waiting %s", list, tc.waiting)
			}
		})
	}
}
