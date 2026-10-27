package indexersapi_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
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
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/indexersapi"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type published struct{ ids []string }

func (p *published) Publish(_ context.Context, v indexers.Changed) {
	p.ids = append(p.ids, v.IndexerID)
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type registry struct{ x *indexers.Indexers }

func (r registry) Indexers() (*indexers.Indexers, bool) { return r.x, true }

// disconnected is a Node whose link is down.
type disconnected struct{}

func (disconnected) Run(context.Context, searching.Job) (searching.Poll, error) {
	return searching.Poll{}, link.ErrDisconnected
}

var (
	self   = role.NodeID{UUID: storage.NewUUID()}
	worker = role.NodeID{UUID: storage.NewUUID()}
)

type served struct {
	owner   *ownertest.Owner
	lib     *storage.DB[storage.Library]
	sealer  *secret.Sealer
	site    faketest.Served
	key     string
	changed *published
	asked   []role.NodeID
}

var fixtureScenario = scenario.Scenario{Name: "x", Seed: "indexersapi"}

// serve serves the operations over a fixture-site every Node reaches but
// down, whose link is disconnected. The roster's Controller is self.
func serve(t *testing.T, r indexerstest.Roster, down role.NodeID) *served {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), indexers.Schema())
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, self, "home"); err != nil {
			return err
		}
		return nodestest.Insert(ctx, tx, worker, "seedbox")
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := &served{lib: lib, sealer: secrettest.Sealer(t), key: site.APIKey(fixtureScenario), changed: &published{}}
	srv.site = faketest.Loopback(t, site.Fake, fixtureScenario)
	timings := searching.DefaultTimings()
	timings.Floor = 0
	home := searching.RemoteOf(searching.New(searching.Config{HTTP: srv.site.Client, Timings: timings, Log: slog.New(slog.DiscardHandler)}))
	x := indexers.New(indexers.Config{
		Library: lib,
		Sealer:  srv.sealer,
		Roster:  r,
		Searching: func(n role.NodeID) searching.Remote {
			srv.asked = append(srv.asked, n)
			if n == down {
				return disconnected{}
			}
			return home
		},
		Changed:  srv.changed,
		Status:   discard[indexers.StatusChanged]{},
		Trackers: discard[indexers.TrackerChanged]{},
		Timings:  indexers.DefaultTimings(),
		Zone:     time.UTC,
		Log:      slog.New(slog.DiscardHandler),
	})
	srv.owner = ownertest.Serve(t, lib, func(r *api.Router) { indexersapi.Routes(r, registry{x}) })
	return srv
}

func (s *served) address() string { return site.URL(s.site.Address) }

func (s *served) body(address, key string) string {
	return `{"kind":"torznab","name":"Fixture Site","address":` + strconv.Quote(address) + `,"apiKey":"` + key + `"}`
}

// create adds Fixture Site with the right key and is its id.
func (s *served) create(t *testing.T) string {
	t.Helper()
	code, reply := s.owner.Call(t, http.MethodPost, "/api/v1/indexers", s.body(s.address(), s.key))
	if code != http.StatusCreated {
		t.Fatalf("createIndexer = %d %s, want 201", code, reply)
	}
	return ownertest.FirstID(reply)
}

// plant saves an Indexer named name on the fixture-site, homed on self
// with key, without a Test.
func (s *served) plant(t *testing.T, name, key string) string {
	t.Helper()
	id := indexers.ID{UUID: storage.NewUUID()}
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return indexerstest.Torznab(ctx, tx, s.sealer, id, name, self, s.address(), key)
	})
	if err != nil {
		t.Fatal(err)
	}
	return id.String()
}

const modes = `"searchModes":[{"mode":"movie","params":["q","tmdbid","imdbid"]},{"mode":"search","params":["q"]}]`

// fresh is the wire of a just-created Fixture Site at revision with
// switches and priority, its home being id2 and its last success T.
func fresh(feedHost, switches string, priority, revision int, categories string) string {
	sentence := "Fixture Site is working."
	if strings.HasPrefix(switches, `"on":false`) {
		sentence = "Fixture Site is off."
	}
	return `{"id":"id1","name":"Fixture Site","kind":"torznab","homeNodeId":"id2",` + switches + `,"priority":` + strconv.Itoa(priority) +
		`,"categories":` + categories + `,"media":null,"allowNoSize":false,"searches":{"count":0,"per":"hour","used":0,"freeAt":null},"grabs":{"count":0,"per":"hour","used":0,"freeAt":null},` +
		`"status":{"state":"ok","cause":null,"text":null,"until":null,"since":null,"nodeId":null,"solverTried":false,"rung":0,"lastFailure":null,"successAt":"T"},` +
		`"sentence":"` + sentence + `","fix":null,"signsIn":false,"trackerIds":["id3"],` +
		`"waits":null,"session":null,"feedHost":"` + feedHost + `",` + modes + `,"revision":` + strconv.Itoa(revision) + `,"createdAt":"T"}`
}

const on = `"on":true,"automaticSearch":true,"rss":true,"allowFailover":false`

func TestAnIndexerIsCreatedOnItsHomeNodeAndListed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		holders []role.NodeID
		home    role.NodeID
	}{
		{"this Node holds Searching", []role.NodeID{self, worker}, self},
		{"only a Worker holds Searching", []role.NodeID{worker}, worker},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := serve(t, indexerstest.Roster{Controller: self, Holders: tc.holders}, role.NodeID{})
			host := site.Host("torznab")
			want := fresh(host, on, 25, 1, "null")
			s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", s.body(s.address(), s.key), http.StatusCreated, want)
			_, list := s.owner.Call(t, http.MethodGet, "/api/v1/indexers", "")
			id := ownertest.FirstID(list)
			if ownertest.Masked(list) != `{"indexers":[`+want+`]}`+"\n" || !strings.Contains(list, `"homeNodeId":"`+tc.home.String()+`"`) {
				t.Fatalf("listIndexers = %s, want %s homed on %s", list, want, tc.home)
			}
			if len(s.changed.ids) != 1 || s.changed.ids[0] != id {
				t.Fatalf("indexers.changed published %v, want [%s]", s.changed.ids, id)
			}
			if len(s.asked) != 1 || s.asked[0] != tc.home {
				t.Fatalf("asked %v for the Test, want the home Node %s", s.asked, tc.home)
			}
			s.owner.Expect(t, http.MethodGet, "/api/v1/indexers/"+id+"/queries", "", http.StatusOK,
				`{"queries":[{"id":"id1","use":"test","reservedAt":"T","startedAt":"T","completedAt":"T","outcome":"ok","releases":0}],"nextOffset":null}`)
		})
	}
}

func TestEachCreateErrorAnswersItsProblem(t *testing.T) {
	s := serve(t, indexerstest.Alone(self), role.NodeID{})
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", s.body(s.address(), "wrong"), http.StatusUnprocessableEntity,
		`{"status":422,"code":"indexer_test_failed","title":"The Indexer's Test failed","detail":"Fixture Site refused the API key with 'Invalid API Key'. Check the account, then test again."}`)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", s.body(s.site.Address.URL("/elsewhere"), s.key), http.StatusUnprocessableEntity,
		`{"status":422,"code":"indexer_test_failed","title":"The Indexer's Test failed","detail":"Fixture Site failed the Test with 'the Indexer responded with status 404'. Check the address and test again."}`)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", s.body("ftp://example.test/torznab", "k"), http.StatusUnprocessableEntity,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"body.address","message":"expected string to match pattern ^https?://[^/?#]+(/[^?#]*)?(\\?[^#]*)?$"}]}`)
	s.create(t)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", s.body(s.address(), s.key), http.StatusConflict,
		`{"status":409,"code":"indexer_name_taken","title":"Another Indexer has that name"}`)
	if len(s.changed.ids) != 1 {
		t.Fatalf("indexers.changed published %d times, want once for the one Indexer made", len(s.changed.ids))
	}

	none := serve(t, indexerstest.Roster{Controller: self}, role.NodeID{})
	none.owner.Expect(t, http.MethodPost, "/api/v1/indexers", none.body(none.address(), none.key), http.StatusConflict,
		`{"status":409,"code":"no_searching_node","title":"No Node holds Searching. Give one the Searching Role first."}`)
	elsewhere := `{"kind":"torznab","name":"Fixture Site","address":"https://site.test","apiKey":"k","homeNodeId":"` + worker.String() + `"}`
	none.owner.Expect(t, http.MethodPost, "/api/v1/indexers", elsewhere, http.StatusConflict,
		`{"status":409,"code":"node_not_searching","title":"That Node doesn't stand in Searching"}`)

	down := serve(t, indexerstest.Roster{Controller: self, Holders: []role.NodeID{worker}}, worker)
	down.owner.Expect(t, http.MethodPost, "/api/v1/indexers", down.body(down.address(), down.key), http.StatusBadGateway,
		`{"status":502,"code":"indexer_unavailable","title":"The Test couldn't run","detail":"SlipStream couldn't reach the Searching Node. Test again once it's back."}`)

	since := time.Date(2026, 10, 17, 21, 31, 0, 0, time.UTC)
	away := serve(t, indexerstest.Roster{Controller: self, Holders: []role.NodeID{self}, Absent: map[role.NodeID]nodes.Absence{self: {Since: since}}}, role.NodeID{})
	away.owner.Expect(t, http.MethodPost, "/api/v1/indexers", away.body(away.address(), away.key), http.StatusBadGateway,
		`{"status":502,"code":"indexer_unavailable","title":"The Test couldn't run","detail":"home has been offline since 21:31. Test again once it's back."}`)
	for _, x := range []*served{none, down, away} {
		if _, list := x.owner.Call(t, http.MethodGet, "/api/v1/indexers", ""); list != `{"indexers":[]}`+"\n" || len(x.changed.ids) != 0 {
			t.Fatalf("after a failed create the list is %s with %d events, want none", list, len(x.changed.ids))
		}
	}
}

func TestAPatchChangesOnlyWhatItSends(t *testing.T) {
	s := serve(t, indexerstest.Alone(self), role.NodeID{})
	id := s.create(t)
	path := "/api/v1/indexers/" + id
	host := site.Host("torznab")

	s.owner.Expect(t, http.MethodPatch, path, `{"on":false,"rss":false,"priority":10,"categories":[2000,5000]}`, http.StatusOK,
		fresh(host, `"on":false,"automaticSearch":true,"rss":false,"allowFailover":false`, 10, 2, "[2000,5000]"))
	s.owner.Expect(t, http.MethodPatch, path, `{"on":null,"rss":null,"priority":null,"categories":null}`, http.StatusOK,
		fresh(host, on, 25, 3, "null"))
	if len(s.asked) != 1 {
		t.Fatalf("edits that change no tested field asked %d Nodes, want only the create's Test", len(s.asked))
	}

	s.owner.Expect(t, http.MethodPatch, path, `{"apiKey":"wrong"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"indexer_test_failed","title":"The Indexer's Test failed","detail":"Fixture Site refused the API key with 'Invalid API Key'. Check the account, then test again."}`)
	s.owner.Expect(t, http.MethodGet, path, "", http.StatusOK, fresh(host, on, 25, 3, "null"))
	s.owner.Expect(t, http.MethodPatch, path, `{"apiKey":"`+s.key+`","searches":{"count":5,"per":"day"}}`, http.StatusOK,
		strings.Replace(fresh(host, on, 25, 4, "null"), `"searches":{"count":0,"per":"hour"`, `"searches":{"count":5,"per":"day"`, 1))

	other := s.plant(t, "Other Site", s.key)
	s.owner.Expect(t, http.MethodPatch, "/api/v1/indexers/"+other, `{"name":"Fixture Site"}`, http.StatusConflict,
		`{"status":409,"code":"indexer_name_taken","title":"Another Indexer has that name"}`)
	s.owner.Expect(t, http.MethodPatch, "/api/v1/indexers/"+storage.NewUUID().String(), `{"priority":5}`, http.StatusNotFound,
		`{"status":404,"code":"indexer_not_found","title":"No Indexer has that id"}`)
	if len(s.changed.ids) != 4 {
		t.Fatalf("indexers.changed published %d times, want the create and three saved edits", len(s.changed.ids))
	}
}

func TestDeleteRemovesTheIndexerOnce(t *testing.T) {
	s := serve(t, indexerstest.Alone(self), role.NodeID{})
	id := s.create(t)
	path := "/api/v1/indexers/" + id
	if code, reply := s.owner.Call(t, http.MethodDelete, path, ""); code != http.StatusNoContent || reply != "" {
		t.Fatalf("deleteIndexer = %d %q, want 204", code, reply)
	}
	gone := `{"status":404,"code":"indexer_not_found","title":"No Indexer has that id"}`
	s.owner.Expect(t, http.MethodGet, path, "", http.StatusNotFound, gone)
	s.owner.Expect(t, http.MethodDelete, path, "", http.StatusNotFound, gone)
	s.owner.Expect(t, http.MethodGet, path+"/queries", "", http.StatusNotFound, gone)
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexers", "", http.StatusOK, `{"indexers":[]}`)
	if len(s.changed.ids) != 2 || s.changed.ids[1] != id {
		t.Fatalf("indexers.changed published %v, want the create and the delete of %s", s.changed.ids, id)
	}
}

func TestATestReportsWhatItLearnedOrWhyItFailed(t *testing.T) {
	absent := map[role.NodeID]nodes.Absence{}
	s := serve(t, indexerstest.Roster{Controller: self, Holders: []role.NodeID{self}, Absent: absent}, role.NodeID{})
	id := s.create(t)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers/"+id+"/test", "", http.StatusOK, `{`+modes+`,"items":0,"sources":[]}`)

	stopped := s.plant(t, "Suspended Site", site.PlayKey(101))
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers/"+stopped+"/test", "", http.StatusUnprocessableEntity,
		`{"status":422,"code":"indexer_test_failed","title":"The Indexer's Test failed","detail":"Suspended Site refused the API key with 'Account suspended'. Check the account, then test again."}`)
	_, v := s.owner.Call(t, http.MethodGet, "/api/v1/indexers/"+stopped, "")
	if !strings.Contains(v, `"status":{"state":"stopped","cause":"account_refused","text":"Account suspended"`) ||
		!strings.Contains(v, `"sentence":"Stopped because Suspended Site refused the API key with 'Account suspended'.","fix":"Check the account, then test the Indexer."`) ||
		!strings.Contains(v, `"waits":{"kind":"stopped"`) {
		t.Fatalf("after a failed Test the Indexer is %s, want stopped for account_refused", v)
	}

	absent[self] = nodes.Absence{Since: time.Date(2026, 10, 17, 21, 31, 0, 0, time.UTC)}
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers/"+id+"/test", "", http.StatusBadGateway,
		`{"status":502,"code":"indexer_unavailable","title":"The Test couldn't run","detail":"home has been offline since 21:31. Test again once it's back."}`)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers/"+storage.NewUUID().String()+"/test", "", http.StatusNotFound,
		`{"status":404,"code":"indexer_not_found","title":"No Indexer has that id"}`)
}

func TestADraftIsTestedWithoutSaving(t *testing.T) {
	s := serve(t, indexerstest.Alone(self), role.NodeID{})
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers/test", s.body(s.address(), s.key), http.StatusOK, `{`+modes+`,"items":0,"sources":[]}`)
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexers", "", http.StatusOK, `{"indexers":[]}`)
	if len(s.changed.ids) != 0 {
		t.Fatalf("a draft's Test published %v, want nothing", s.changed.ids)
	}
}

func TestAnIndexerSaysItWaitsForItsHomeUntilTheGrace(t *testing.T) {
	absent := map[role.NodeID]nodes.Absence{}
	s := serve(t, indexerstest.Roster{Controller: self, Holders: []role.NodeID{self, worker}, Absent: absent}, role.NodeID{})
	s.create(t)
	since := time.Date(2026, 10, 17, 21, 31, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		absence nodes.Absence
		waits   string
	}{
		{"offline within the grace", nodes.Absence{Since: since, Standing: nodes.Offline},
			`{"kind":"waiting","releases":0,"until":null,"since":"2026-10-17T21:31:00Z","nodeId":"` + self.String() + `","lapsed":false,"cause":null,"text":null}`},
		{"lapsed, so a feed searches elsewhere", nodes.Absence{Since: since, Standing: nodes.LapsedNode, Lapsed: true}, "null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			absent[self] = tc.absence
			if _, list := s.owner.Call(t, http.MethodGet, "/api/v1/indexers", ""); !strings.Contains(list, `"waits":`+tc.waits+`,`) {
				t.Fatalf("listIndexers = %s, want waits %s", list, tc.waits)
			}
		})
	}
}

func TestTheSettingsChangeAllOrNone(t *testing.T) {
	s := serve(t, indexerstest.Alone(self), role.NodeID{})
	field := func(value int, isDefault bool) string {
		v := strconv.Itoa(value)
		return `{"settings":[{"key":"indexers.longBackoff","label":"Warn when every request has failed for","source":"database","applies":"live",` +
			`"isDefault":` + strconv.FormatBool(isDefault) + `,"pendingRestart":false,"kind":"duration","value":` + v + `,"running":` + v +
			`,"default":21600,"min":3600,"max":604800,"step":3600}]}`
	}
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexer-settings", "", http.StatusOK, field(21600, true))
	s.owner.Expect(t, http.MethodPatch, "/api/v1/indexer-settings", `{"values":{"indexers.longBackoff":7200}}`, http.StatusOK, field(7200, false))
	code, _ := s.owner.Call(t, http.MethodPatch, "/api/v1/indexer-settings", `{"values":{"indexers.longBackoff":60}}`)
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("a long backoff under an hour answered %d, want 422", code)
	}
	code, _ = s.owner.Call(t, http.MethodPatch, "/api/v1/indexer-settings", `{"values":{"indexers.longBackoff":3600,"indexers.other":1}}`)
	if code == http.StatusOK {
		t.Fatal("a patch with an unknown key was applied")
	}
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexer-settings", "", http.StatusOK, field(7200, false))
	s.owner.Expect(t, http.MethodPatch, "/api/v1/indexer-settings", `{"values":{"indexers.longBackoff":null}}`, http.StatusOK, field(21600, true))
}

func TestTrackersListTheIndexersThatReachThem(t *testing.T) {
	s := serve(t, indexerstest.Alone(self), role.NodeID{})
	s.owner.Expect(t, http.MethodGet, "/api/v1/trackers", "", http.StatusOK, `{"trackers":[]}`)
	feed := s.plant(t, "Prowlarr", s.key)
	torrentLeech := s.plant(t, "TorrentLeech", s.key)
	s.reach(t, map[string][]string{"TorrentLeech": {feed, torrentLeech}, "Aither": {feed}, "Orphan": nil})
	_, list := s.owner.Call(t, http.MethodGet, "/api/v1/trackers", "")
	want := `{"trackers":[` +
		`{"id":"Aither","name":"Aither","privacy":"private","hosts":[],"keys":[],"indexerIds":["Prowlarr"]},` +
		`{"id":"Orphan","name":"Orphan","privacy":"private","hosts":[],"keys":[],"indexerIds":[]},` +
		`{"id":"TorrentLeech","name":"TorrentLeech","privacy":"private","hosts":[],"keys":[],"indexerIds":["Prowlarr","TorrentLeech"]}]}` + "\n"
	var trackers struct {
		Trackers []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"trackers"`
	}
	if err := json.Unmarshal([]byte(list), &trackers); err != nil {
		t.Fatal(err)
	}
	names := []string{feed, "Prowlarr", torrentLeech, "TorrentLeech"}
	for _, tr := range trackers.Trackers {
		names = append(names, tr.ID, tr.Name)
	}
	if list = strings.NewReplacer(names...).Replace(list); list != want {
		t.Fatalf("listTrackers = %s, want %s", list, want)
	}
	var v struct {
		TrackerIDs []string `json:"trackerIds"`
	}
	_, body := s.owner.Call(t, http.MethodGet, "/api/v1/indexers/"+feed, "")
	if err := json.Unmarshal([]byte(body), &v); err != nil || len(v.TrackerIDs) != 2 {
		t.Fatalf("getIndexer = %s, want the two Trackers it reaches", body)
	}
}

// reach adds each Tracker, reached by the Indexers with the given ids.
func (s *served) reach(t *testing.T, trackers map[string][]string) {
	t.Helper()
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for name, indexers := range trackers {
			id := storage.NewUUID()
			if _, err := tx.ExecContext(ctx, `INSERT INTO indexers_tracker (id, name, privacy, created_at) VALUES (?, ?, 'private', 0)`, id, name); err != nil {
				return err
			}
			for _, raw := range indexers {
				indexer, err := storage.ParseUUID(raw)
				if err != nil {
					return err
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO indexers_reach (indexer_id, tracker_id) VALUES (?, ?)`, indexer, id); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
