package trackersapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/trackersapi"
	"github.com/jatassi/slipstream/internal/works"
)

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type registry struct {
	indexers *indexers.Indexers
	trackers *trackers.Trackers
}

func (r registry) Indexers() (*indexers.Indexers, bool) { return r.indexers, true }

func (r registry) Trackers() (*trackers.Trackers, bool) { return r.trackers, true }

// roster is every Node in step, and drains nothing.
type roster struct{}

func (roster) InStep(role.NodeID) bool                { return true }
func (roster) WrittenOff(role.Name, role.NodeID) bool { return false }
func (roster) Changed() <-chan struct{}               { return nil }
func (roster) Serving(role.Name) []role.NodeID        { return nil }
func (roster) Draining(role.Name, role.NodeID) bool   { return false }
func (roster) Holds(role.Name, role.NodeID) bool      { return true }

// holding is the roster with the Nodes assigned Downloading as a test
// sets them.
type holding struct {
	roster
	nodes []role.NodeID
}

func (h *holding) Holding(role.Name) []role.NodeID { return h.nodes }

// late is the Trackers' seeder once they are made, after the clients.
type late struct{ x **trackers.Trackers }

func (l late) Judge(ctx context.Context, r storage.Reader[storage.Library], rows []downloadclients.Row) (map[downloadclients.ID]downloadclients.Judged, error) {
	return (*l.x).Seeder().Judge(ctx, r, rows)
}

func (l late) EndlessUnits(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) (int64, error) {
	return (*l.x).Seeder().EndlessUnits(ctx, r, node)
}

type served struct {
	owner   *ownertest.Owner
	lib     *storage.DB[storage.Library]
	holding *holding
}

func serve(t *testing.T) *served {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(),
		downloadclients.Schema(), grabs.Schema(), indexers.Schema(), trackers.Schema())
	r := registry{
		indexers: indexers.New(indexers.Config{
			Library: lib, Trackers: discard[indexers.TrackerChanged]{},
			Repoints: []indexers.Repoint{grabs.RepointTracker, downloadclients.RepointTracker, trackers.RepointTracker},
		}),
	}
	h := &holding{}
	clients := downloadclients.New(downloadclients.Config{Library: lib, Roster: roster{}, Uses: grabs.TorrentUses{}, Seeding: late{&r.trackers}})
	r.trackers = trackers.New(trackers.Config{
		Library: lib, Clients: clients, Grabs: grabs.TrackerFacts{}, Roster: roster{},
		Soon: func() {}, Reconciling: trackers.NotReconciling, Changed: discard[trackers.SeedingChanged]{}, TrackersChanged: discard[indexers.TrackerChanged]{},
		Routing: trackers.Routing{Roster: h}, Rerouted: func() {},
	})
	return &served{owner: ownertest.Serve(t, lib, func(router *api.Router) { trackersapi.Routes(router, r) }), lib: lib, holding: h}
}

// tracker adds a Tracker named name reached by no Indexer.
func (s *served) tracker(t *testing.T, name string, privacy indexers.Privacy) string {
	t.Helper()
	id := storage.NewUUID()
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO indexers_tracker (id, name, privacy, created_at) VALUES (?, ?, ?, 0)`, id, name, string(privacy))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id.String()
}

func TestTheTrackerOperationsAnswerAsBeforeTheMove(t *testing.T) {
	s := serve(t)
	s.owner.Expect(t, http.MethodGet, "/api/v1/trackers", "", http.StatusOK, `{"trackers":[]}`)
	ids := map[string]string{}
	for _, name := range []string{"TorrentLeech", "aither", "Orphan"} {
		ids[name] = s.tracker(t, name, indexers.Private)
	}
	_, list := s.owner.Call(t, http.MethodGet, "/api/v1/trackers", "")
	item := func(name string) string {
		return `{"id":"` + ids[name] + `","name":"` + name + `","privacy":"private","hosts":[],"keys":[],"indexerIds":[],"rule":null,"goal":null,` +
			`"counts":{"torrents":0,"seeding":0,"owing":0,"atRisk":0,"goalMet":0}}`
	}
	if want := `{"trackers":[` + item("aither") + `,` + item("Orphan") + `,` + item("TorrentLeech") + `]}` + "\n"; list != want {
		t.Fatalf("listTrackers = %s, want %s", list, want)
	}
	s.owner.Expect(t, http.MethodPatch, "/api/v1/trackers/"+ids["aither"], `{"name":"torrentleech"}`, http.StatusConflict,
		`{"status":409,"code":"tracker_name_taken","title":"Another Tracker has that name"}`)
	s.owner.Expect(t, http.MethodPost, "/api/v1/trackers/"+ids["aither"]+"/merge", `{"into":"`+ids["aither"]+`"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"tracker_merge_self","title":"A Tracker can't be merged into itself"}`)
	s.owner.Expect(t, http.MethodPost, "/api/v1/trackers/"+ids["aither"]+"/merge", `{"into":"`+storage.NewUUID().String()+`"}`, http.StatusNotFound,
		`{"status":404,"code":"tracker_not_found","title":"That Tracker no longer exists. It may have been removed or merged into another Tracker."}`)
	if code, got := s.owner.Call(t, http.MethodPatch, "/api/v1/trackers/"+ids["Orphan"], `{"name":"Lonely"}`); code != http.StatusOK || !strings.Contains(got, `"name":"Lonely"`) {
		t.Fatalf("renameTracker = %d %s, want 200 with the new name", code, got)
	}
	if code, got := s.owner.Call(t, http.MethodPost, "/api/v1/trackers/"+ids["aither"]+"/merge", `{"into":"`+ids["TorrentLeech"]+`"}`); code != http.StatusOK ||
		!strings.Contains(got, `"name":"TorrentLeech"`) {
		t.Fatalf("mergeTracker = %d %s, want 200 with TorrentLeech", code, got)
	}
	s.owner.Expect(t, http.MethodGet, "/api/v1/trackers/"+ids["aither"], "", http.StatusNotFound,
		`{"status":404,"code":"tracker_not_found","title":"That Tracker no longer exists. It may have been removed or merged into another Tracker."}`)
}

func TestAPublicTrackerRefusesARule(t *testing.T) {
	s := serve(t)
	nyaa := s.tracker(t, "Nyaa", indexers.Public)
	refused := `{"status":409,"code":"tracker_public","title":"This Tracker is public","detail":"Nyaa is public, so its torrents follow your seeding goal alone."}`
	s.owner.Expect(t, http.MethodPut, "/api/v1/trackers/"+nyaa+"/rule", `{"clears":"time","ratio":null,"seedSeconds":3600}`, http.StatusConflict, refused)
	s.owner.Expect(t, http.MethodDelete, "/api/v1/trackers/"+nyaa+"/rule", "", http.StatusConflict, refused)
	if code, got := s.owner.Call(t, http.MethodPut, "/api/v1/trackers/"+nyaa+"/goal", `{"combine":"forever","ratio":null,"seedSeconds":null,"action":null}`); code != http.StatusOK ||
		!strings.Contains(got, `"rule":null,"goal":{"combine":"forever","ratio":null,"seedSeconds":null,"action":null}`) {
		t.Fatalf("setTrackerGoal on a public Tracker = %d %s, want 200 with its goal", code, got)
	}
}

func TestSeedingValuesOutOfRangeAreRefused(t *testing.T) {
	s := serve(t)
	seedpool := s.tracker(t, "seedpool", indexers.Private)
	refused := func(field, detail string) string {
		return `{"status":422,"code":"seeding_value_refused","title":"A seeding value is out of range","detail":"` + detail +
			`","errors":[{"location":"body.` + field + `","message":"` + detail + `"}]}`
	}
	for _, c := range []struct{ path, body, field, detail string }{
		{"/rule", `{"clears":"ratio","ratio":1.234,"seedSeconds":null}`, "ratio", "Use a ratio of 0.01 or more with at most two decimals."},
		{"/rule", `{"clears":"ratio","ratio":0,"seedSeconds":null}`, "ratio", "Use a ratio of 0.01 or more with at most two decimals."},
		{"/rule", `{"clears":"time","ratio":1,"seedSeconds":3600}`, "clears", "Give a seeding time and no ratio for a rule that clears by time."},
		{"/rule", `{"clears":"both","ratio":1,"seedSeconds":null}`, "clears", "Give a ratio and a seeding time for a rule that clears by both."},
		{"/rule", `{"clears":"time","ratio":null,"seedSeconds":0}`, "seedSeconds", "Use a seeding time between 1 second and 3650 days."},
		{"/goal", `{"combine":"forever","ratio":null,"seedSeconds":null,"action":"pause"}`, "combine", "Leave out the ratio, seeding time and action of a goal that seeds forever."},
		{"/goal", `{"combine":"first","ratio":2,"seedSeconds":3600,"action":null}`, "action", "Choose what happens to a torrent once the goal is met."},
		{"/goal", `{"combine":"both","ratio":2.001,"seedSeconds":3600,"action":"remove"}`, "ratio", "Use a ratio of 0.01 or more with at most two decimals."},
		{"/goal", `{"combine":"both","ratio":null,"seedSeconds":null,"action":"remove"}`, "ratio", "Give the goal a ratio, a seeding time or both."},
	} {
		s.owner.Expect(t, http.MethodPut, "/api/v1/trackers/"+seedpool+c.path, c.body, http.StatusUnprocessableEntity, refused(c.field, c.detail))
	}
	s.owner.Expect(t, http.MethodPut, "/api/v1/trackers/default-goal", `{"combine":"first","ratio":null,"seedSeconds":null,"action":"remove"}`,
		http.StatusUnprocessableEntity, refused("ratio", "Give the goal a ratio, a seeding time or both."))
}

func TestATwoDecimalRatioRoundTrips(t *testing.T) {
	s := serve(t)
	seedpool := s.tracker(t, "seedpool", indexers.Private)
	ruled := `"rule":{"clears":"either","ratio":1.25,"seedSeconds":432000,"source":"admin"}`
	if code, got := s.owner.Call(t, http.MethodPut, "/api/v1/trackers/"+seedpool+"/rule", `{"clears":"either","ratio":1.25,"seedSeconds":432000}`); code != http.StatusOK ||
		!strings.Contains(got, ruled+`,"goal":null}`) {
		t.Fatalf("setTrackerRule = %d %s, want 200 with %s", code, got, ruled)
	}
	own := `{"combine":"both","ratio":0.07,"seedSeconds":86400,"action":"remove-delete"}`
	if code, got := s.owner.Call(t, http.MethodPut, "/api/v1/trackers/"+seedpool+"/goal", own); code != http.StatusOK || !strings.Contains(got, ruled+`,"goal":`+own+`}`) {
		t.Fatalf("setTrackerGoal = %d %s, want 200 with %s", code, got, own)
	}
	if _, got := s.owner.Call(t, http.MethodGet, "/api/v1/trackers/"+seedpool, ""); !strings.Contains(got, ruled+`,"goal":`+own+`,"stats":`) {
		t.Fatalf("getTracker = %s, want the rule and goal it was given", got)
	}
	if _, got := s.owner.Call(t, http.MethodDelete, "/api/v1/trackers/"+seedpool+"/rule", ""); !strings.Contains(got, `"rule":null`) {
		t.Fatalf("clearTrackerRule = %s, want no rule, since no Release stated one", got)
	}

	initial := `{"combine":"first","ratio":2,"seedSeconds":1209600,"action":"remove-delete"}`
	s.owner.Expect(t, http.MethodGet, "/api/v1/trackers/default-goal", "", http.StatusOK, initial)
	changed := `{"combine":"first","ratio":3.33,"seedSeconds":600,"action":"pause"}`
	s.owner.Expect(t, http.MethodPut, "/api/v1/trackers/default-goal", changed, http.StatusOK, changed)
	s.owner.Expect(t, http.MethodGet, "/api/v1/trackers/default-goal", "", http.StatusOK, changed)
	if _, got := s.owner.Call(t, http.MethodPut, "/api/v1/trackers/"+seedpool+"/goal", changed); !strings.Contains(got, `"goal":null`) {
		t.Fatalf("setTrackerGoal to the default = %s, want it following the default", got)
	}
}

func TestAGoalWithOnePartRoundTrips(t *testing.T) {
	s := serve(t)
	seedpool := s.tracker(t, "seedpool", indexers.Private)
	for _, own := range []string{
		`{"combine":"both","ratio":1.5,"seedSeconds":null,"action":"pause"}`,
		`{"combine":"first","ratio":null,"seedSeconds":86400,"action":"remove"}`,
	} {
		if code, got := s.owner.Call(t, http.MethodPut, "/api/v1/trackers/"+seedpool+"/goal", own); code != http.StatusOK || !strings.Contains(got, `"goal":`+own+`}`) {
			t.Fatalf("setTrackerGoal = %d %s, want 200 with %s", code, got, own)
		}
		if _, got := s.owner.Call(t, http.MethodGet, "/api/v1/trackers/"+seedpool, ""); !strings.Contains(got, `"goal":`+own+`,"stats":`) {
			t.Fatalf("getTracker = %s, want the goal it was given", got)
		}
		s.owner.Expect(t, http.MethodPut, "/api/v1/trackers/default-goal", own, http.StatusOK, own)
		s.owner.Expect(t, http.MethodGet, "/api/v1/trackers/default-goal", "", http.StatusOK, own)
	}
}

func TestTheSeedingSettingsReadAndChangeByKey(t *testing.T) {
	s := serve(t)
	if code, got := s.owner.Call(t, http.MethodPatch, "/api/v1/trackers/settings", `{"values":{"trackers.margin":1800,"trackers.warnAfter":null}}`); code != http.StatusOK ||
		!strings.Contains(got, `"key":"trackers.margin"`) || !strings.Contains(got, `"value":1800`) || !strings.Contains(got, `"key":"trackers.warnAfter"`) {
		t.Fatalf("updateSeedingSettings = %d %s, want both settings with the margin at 1800", code, got)
	}
	if code, got := s.owner.Call(t, http.MethodPatch, "/api/v1/trackers/settings", `{"values":{"trackers.goal":1}}`); code != http.StatusUnprocessableEntity ||
		!strings.Contains(got, `"code":"unknown_setting"`) {
		t.Fatalf("updateSeedingSettings with an unknown key = %d %s, want unknown_setting", code, got)
	}
}

func (s *served) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, sql, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTheListCountsATrackersTorrentsAndItsPageSumsThem(t *testing.T) {
	s := serve(t)
	seedpool := s.tracker(t, "seedpool", indexers.Private)
	tracker, err := storage.ParseUUID(seedpool)
	if err != nil {
		t.Fatal(err)
	}
	node, work, item := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	s.exec(t, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'seedbox', 0)`, node)
	s.exec(t, `INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, work)
	s.exec(t, `INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, item, work)
	now := storage.Now()
	held := func(name, wanted, report string, up, down int64) storage.UUID {
		t.Helper()
		id := storage.NewUUID()
		s.exec(t, `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, tracker_id,
			wanted, added_order, added_at, uploaded, downloaded, report, reported_at)
			VALUES (?, ?, randomblob(20), x'00', ?, 4294967296, 1, '[]', 'fixture', ?, ?,
			(SELECT coalesce(max(added_order), 0) + 1 FROM downloadclients_torrent), ?, ?, ?, ?, ?)`,
			id, node, name, tracker, wanted, now, up, down, report, now)
		return id
	}
	paused := `{"at":"2026-11-08T00:00:00Z","engine":"seeding","hold":"wanted","complete":true,"done":0}`
	orchard := held("Orchard", "paused", paused, 1<<30, 4<<30)
	gone := held("Harbor", "started", paused, 8<<30, 4<<30)
	s.exec(t, `UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = 'admin', parted_at = ?, freed = 0 WHERE id = ?`, now, now, gone)
	s.exec(t, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, tracker_id, node_id, release_title, release_size, release_magnet,
		stage, created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, ?, 'Orchard.1080p', 1, 'magnet:?xt=urn:btih:00', 'fetching', ?, ?)`,
		storage.NewUUID(), work, item, tracker, node, now, now)

	if _, got := s.owner.Call(t, http.MethodGet, "/api/v1/trackers", ""); !strings.Contains(got, `"counts":{"torrents":1,"seeding":1,"owing":1,"atRisk":1,"goalMet":0}`) {
		t.Errorf("listTrackers = %s, want seedpool counting its paused Orchard, which owes it and is at risk", got)
	}
	want := `"stats":{"uploaded":9663676416,"downloaded":8589934592,"ratio":1.12,"torrents":1,"seedingBytes":4294967296,"grabs30d":1,"owing":1}`
	_, page := s.owner.Call(t, http.MethodGet, "/api/v1/trackers/"+seedpool, "")
	if !strings.Contains(page, want) {
		t.Errorf("getTracker = %s, want %s: removed Harbor's upload counts, and only live Orchard seeds and owes", page, want)
	}
	attention := `"attention":[{"level":"warn","text":"No torrents are seeding. 1 still owes seedpool, and it isn't seeding."},` +
		`{"level":"info","text":"seedpool's Releases don't state a seeding rule. Set one so its seeding goal can run."}]`
	if !strings.Contains(page, attention) {
		t.Errorf("getTracker = %s, want %s", page, attention)
	}
	_, owing := s.owner.Call(t, http.MethodGet, "/api/v1/trackers/"+seedpool+"/windows?state=owing", "")
	if !strings.Contains(owing, `"name":"Orchard","state":"owing","torrents":["`+orchard.String()+`"],"rule":null,`) || !strings.Contains(owing, `"owed":null,"owes":null,"atRisk":"yes","quietSince":`) ||
		!strings.HasSuffix(owing, `"nextOffset":null}`+"\n") {
		t.Errorf("listTrackerWindows owing = %s, want Orchard's window alone with its live Torrent, owing with no rule known and at risk", owing)
	}
	s.owner.Expect(t, http.MethodGet, "/api/v1/trackers/"+seedpool+"/windows?state=met", "", http.StatusOK, `{"windows":[],"nextOffset":null}`)
	s.owner.Expect(t, http.MethodGet, "/api/v1/torrents/goals/preview", "", http.StatusOK, `{"acts":[],"kept":[],"waits":[]}`)
}
