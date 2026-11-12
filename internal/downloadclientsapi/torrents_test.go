package downloadclientsapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
	"github.com/jatassi/slipstream/internal/works"
)

// node is the Node's client: it answers Inspect for the Torrents a test
// set, refuses every action until acceptsAt, and frees freed bytes for
// each Torrent.
type node struct {
	downloadingv1connect.UnimplementedDownloadingServiceHandler
	inspected map[downloadclients.ID]*dlv1.InspectResponse
	acceptsAt time.Time
	freed     int64
	// tests are the Tests the Node ran, and answer is how it answers the
	// next, a pass of Transmission 4.1.1 while nil.
	tests  []test
	answer *dlv1.TestClientResponse
}

func (n *node) Inspect(_ context.Context, req *dlv1.InspectRequest) (*dlv1.InspectResponse, error) {
	u, _ := storage.UUIDFromBytes(req.GetId())
	if res, ok := n.inspected[downloadclients.ID{UUID: u}]; ok {
		return res, nil
	}
	return nil, errors.New("not held")
}

func (n *node) Act(_ context.Context, req *dlv1.ActRequest) (*dlv1.ActResponse, error) {
	res := &dlv1.ActResponse{}
	for _, id := range req.GetIds() {
		res.Results = append(res.Results, &dlv1.ActResult{Id: id, Refused: dlv1.Refusal_REFUSAL_WINDOW, AcceptsAtUnixMs: n.acceptsAt.UnixMilli()})
	}
	return res, nil
}

func (n *node) Frees(_ context.Context, req *dlv1.FreesRequest) (*dlv1.FreesResponse, error) {
	res := &dlv1.FreesResponse{}
	for _, id := range req.GetIds() {
		res.Freed = append(res.Freed, &dlv1.Freed{Id: id, Bytes: n.freed})
	}
	return res, nil
}

const orchardRelease = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW"

var reportedAt = time.Date(2026, 10, 19, 21, 0, 0, 0, time.UTC)

// hold attaches a Torrent named name on the world's Node and applies
// report, when there is one.
func (w *world) hold(t *testing.T, name string, report *dlv1.TorrentReport) downloadclients.ID {
	t.Helper()
	return w.holdIn(t, name, downloadclients.Target{Node: w.self}, report)
}

// holdIn attaches a Torrent named name in client to and applies report,
// when there is one.
func (w *world) holdIn(t *testing.T, name string, to downloadclients.Target, report *dlv1.TorrentReport) downloadclients.ID {
	t.Helper()
	raw, err := bencode.Encode(map[string]any{
		"announce-list": []any{[]any{"https://beacon.test/announce/0123456789abcdef0123456789abcdef"}, []any{"udp://backup.test:6969/announce"}},
		"info": map[string]any{
			"name": name, "piece length": int64(16384), "pieces": "aaaaaaaaaaaaaaaaaaaa", "private": int64(1),
			"files": []any{map[string]any{"length": int64(1000), "path": []any{name + ".mkv"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	var id downloadclients.ID
	err = w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		id, _, err = downloadclients.Attach(ctx, tx, downloadclients.Want{Target: to, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: func(role.NodeID) bool { return false }}, storage.Now())
		if err != nil || report == nil {
			return err
		}
		report.Id = id.Bytes()
		c := downloadclients.New(downloadclients.Config{Touch: func(...role.NodeID) {}, Changed: torrentsChanged{}})
		return c.Apply(ctx, tx, to.Node, &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{report}}}})
	})
	if err != nil {
		t.Fatal(err)
	}
	w.hashes[id] = m.InfoHash.String()
	return id
}

func decodeReply(t *testing.T, w *world, path string, v any) {
	t.Helper()
	status, reply := w.o.Call(t, http.MethodGet, path, "")
	if err := json.Unmarshal([]byte(reply), v); status != http.StatusOK || err != nil {
		t.Fatalf("GET %s = %d %s, want 200", path, status, reply)
	}
}

func maskedReply(reply string) string { return strings.TrimSuffix(ownertest.Masked(reply), "\n") }

func unregisteredSeeding() *dlv1.TorrentReport {
	return &dlv1.TorrentReport{
		AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_SEEDING, Complete: true,
		Completion: &dlv1.Completion{Files: []*dlv1.File{{Index: 0, Path: "Harbor/Harbor.mkv", Size: 1000}}},
		Trackers: []*dlv1.TrackerLine{
			{Index: 0, Status: dlv1.TrackerStatus_TRACKER_STATUS_UNREGISTERED, Message: "Unregistered torrent", RetryAtUnixMs: reportedAt.Add(time.Hour).UnixMilli()},
		},
		Fault: &dlv1.Fault{Kind: dlv1.FaultKind_FAULT_KIND_UNREGISTERED, AtUnixMs: reportedAt.UnixMilli()},
	}
}

func queuedReport() *dlv1.TorrentReport {
	return &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Hold: dlv1.Hold_HOLD_SLOT, Done: 250}
}

func TestTheTorrentsListAnswersEachRowsWordsAndCountsEveryTorrent(t *testing.T) {
	w := serveWorld(t)
	orchard := w.hold(t, "Orchard", queuedReport())
	harbor := w.hold(t, "Harbor", unregisteredSeeding())
	work := works.WorkID{UUID: storage.NewUUID()}
	w.uses[orchard] = []downloadclients.Use{{Grab: storage.NewUUID(), Work: work, Edition: "any", Release: orchardRelease, Stage: downloadclients.UseDownloading}}

	counts := func(node string) string {
		return `"counts":{"byState":[{"state":"removing","torrents":0},{"state":"failed","torrents":0},{"state":"tracker-error","torrents":1},` +
			`{"state":"paused","torrents":0},{"state":"checking","torrents":0},{"state":"queued","torrents":1},{"state":"stalled","torrents":0},` +
			`{"state":"downloading","torrents":0},{"state":"seeding","torrents":0}],` +
			`"byTab":[{"tab":"downloading","torrents":1},{"tab":"seeding","torrents":0},{"tab":"problems","torrents":1},{"tab":"goal-met","torrents":0},{"tab":"all","torrents":2}],` +
			`"byNode":[{"nodeId":"` + node + `","torrents":2}]}}`
	}
	orchardRow := `{"id":"id1","nodeId":"id2","deleteFiles":true,"name":"Orchard","contentType":"fixture","workId":"id3","edition":"any","quality":"webdl-1080p",` +
		`"state":"queued","hold":"slot","queuePosition":1,"severity":"none","tabs":["downloading","all"],"size":1000,"done":250,"uploaded":0,` +
		`"ratio":0,"seedingSeconds":0,"announce":{"host":"beacon.test","status":"not_contacted"},` +
		unjudged + `,"retryable":false,"held":true,"addedAt":"T"}`
	w.o.Expect(t, http.MethodGet, "/api/v1/torrents?tab=downloading", "", http.StatusOK, `{"rows":[`+orchardRow+`],`+counts("id2"))

	harborRow := `{"id":"id1","nodeId":"id2","deleteFiles":true,"name":"Harbor","contentType":"fixture","state":"tracker-error","severity":"failure",` +
		`"tabs":["problems","all"],"size":1000,"done":1000,"uploaded":0,"ratio":0,"seedingSeconds":0,` +
		`"announce":{"host":"beacon.test","status":"unregistered","message":"Unregistered torrent","retryAt":"T"},` + unjudged + `,` +
		`"fault":{"kind":"unregistered","at":"T","sentence":"An announce URL responded that the torrent is no longer registered."},"retryable":false,"held":false,"addedAt":"T"}`
	w.o.Expect(t, http.MethodGet, "/api/v1/torrents?ids="+harbor.String(), "", http.StatusOK, `{"rows":[`+harborRow+`],`+counts("id2"))

	var got struct {
		Rows []struct {
			ID string `json:"id"`
		} `json:"rows"`
	}
	decodeReply(t, w, "/api/v1/torrents?ids="+orchard.String()+","+harbor.String(), &got)
	if len(got.Rows) != 2 || got.Rows[0].ID != orchard.String() || got.Rows[1].ID != harbor.String() {
		t.Errorf("listTorrents for both ids = %+v, want both rows in added order", got.Rows)
	}
	w.o.Expect(t, http.MethodGet, "/api/v1/torrents?q=quiet.orchard&tab=all", "", http.StatusOK, `{"rows":[`+orchardRow+`],`+counts("id2"))
	w.o.Expect(t, http.MethodGet, "/api/v1/torrents?nodeId="+storage.NewUUID().String(), "", http.StatusOK, `{"rows":[],`+counts("id1"))
}

// unjudged is the seeding of a private torrent with no Tracker that has
// downloaded nothing, on the initial default goal.
const unjudged = `"tracker":null,"seeding":{"ratio":{"value":0,"goal":2},"time":{"value":0,"goal":1209600},` +
	`"state":"none","atRisk":"no","goalMet":false,"combine":"first","action":"remove-delete","endless":"none","clears":"none"}`

func TestATorrentReadsLiveFromItsNodeAndRefusesAnUnknownID(t *testing.T) {
	w := serveWorld(t)
	harbor := w.hold(t, "Harbor", unregisteredSeeding())
	answered := reportedAt.Add(-time.Minute)
	w.node.inspected = map[downloadclients.ID]*dlv1.InspectResponse{harbor: {
		Files:            []*dlv1.FileProgress{{Index: 0, Done: 1000}},
		Trackers:         []*dlv1.TrackerDetail{{Index: 1, Status: dlv1.TrackerStatus_TRACKER_STATUS_WORKING, AnsweredAtUnixMs: answered.UnixMilli(), MinIntervalSeconds: 1800}},
		AnnounceAtUnixMs: answered.Add(30 * time.Minute).UnixMilli(),
	}}
	file, item := 0, works.ItemID{UUID: storage.NewUUID()}
	w.uses[harbor] = []downloadclients.Use{{Grab: storage.NewUUID(), Work: works.WorkID{UUID: storage.NewUUID()}, Item: item, Release: "Harbor.2019.720p.HDTV", Stage: downloadclients.UseImported, File: &file}}

	_, reply := w.o.Call(t, http.MethodGet, "/api/v1/torrents/"+harbor.String(), "")
	want := `{"id":"id1","nodeId":"id2","deleteFiles":true,"name":"Harbor","contentType":"fixture","workId":"id3","quality":"hdtv-720p","state":"tracker-error","severity":"failure",` +
		`"tabs":["problems","all"],"size":1000,"done":1000,"uploaded":0,"ratio":0,"seedingSeconds":0,` +
		`"announce":{"host":"beacon.test","status":"unregistered","message":"Unregistered torrent","retryAt":"T"},` + unjudged + `,` +
		`"fault":{"kind":"unregistered","at":"T","sentence":"An announce URL responded that the torrent is no longer registered."},"retryable":false,"held":false,"addedAt":"T",` +
		`"downloaded":0,"activeSeconds":0,"infoHash":"` + w.hashes[harbor] + `","pieceSize":16384,"private":true,` +
		`"trackers":[{"url":"https://beacon.test/announce/•••","tier":0,"status":"unregistered","message":"Unregistered torrent","retryAt":"T"},` +
		`{"url":"udp://backup.test:6969/announce","tier":1,"status":"working","answeredAt":"T","minIntervalSeconds":1800}],` +
		`"files":[{"index":0,"path":"Harbor/Harbor.mkv","size":1000,"done":1000,"itemId":"id4"}],` +
		`"grabs":[{"id":"id5","workId":"id3","itemId":"id4","release":"Harbor.2019.720p.HDTV","stage":"imported"}],"live":true,"announceAt":"T"}`
	if got := maskedReply(reply); got != want {
		t.Errorf("getTorrent =\n%s\nwant\n%s", got, want)
	}
	w.o.Expect(t, http.MethodGet, "/api/v1/torrents/"+storage.NewUUID().String(), "", http.StatusNotFound,
		`{"status":404,"code":"torrent_not_found","title":"No Torrent has this id"}`)
}

func TestATrackerLineCountsTheSwarmOnlyWhenItsResponseDid(t *testing.T) {
	w := serveWorld(t)
	harbor := w.hold(t, "Harbor", unregisteredSeeding())
	working := dlv1.TrackerStatus_TRACKER_STATUS_WORKING
	w.node.inspected = map[downloadclients.ID]*dlv1.InspectResponse{harbor: {Trackers: []*dlv1.TrackerDetail{
		{Index: 0, Status: working},
		{Index: 1, Status: working, Seeders: new(int32(142)), Leechers: new(int32(0))},
	}}}
	var got struct {
		Trackers []map[string]any `json:"trackers"`
	}
	decodeReply(t, w, "/api/v1/torrents/"+harbor.String(), &got)
	if len(got.Trackers) != 2 {
		t.Fatalf("getTorrent lists trackers %v, want Harbor's two", got.Trackers)
	}
	if _, ok := got.Trackers[0]["seeders"]; ok {
		t.Errorf("the line whose response carried no counts reads %v, want no seeders", got.Trackers[0])
	}
	if _, ok := got.Trackers[0]["leechers"]; ok {
		t.Errorf("the line whose response carried no counts reads %v, want no leechers", got.Trackers[0])
	}
	if s, l := got.Trackers[1]["seeders"], got.Trackers[1]["leechers"]; s != float64(142) || l != float64(0) {
		t.Errorf("the line whose response counted 142 seeders and 0 leechers reads %v", got.Trackers[1])
	}
}

func TestActionsAnswerForEachTorrent(t *testing.T) {
	w := serveWorld(t)
	orchard := w.hold(t, "Orchard", queuedReport())
	harbor := w.hold(t, "Harbor", unregisteredSeeding())
	unknown := storage.NewUUID().String()
	work := works.WorkID{UUID: storage.NewUUID()}
	w.uses[orchard] = []downloadclients.Use{{Grab: storage.NewUUID(), Work: work, Release: orchardRelease, Stage: downloadclients.UseDownloading}}
	w.node.acceptsAt, w.node.freed = reportedAt.Add(20*time.Minute), 900

	w.o.Expect(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"pause","ids":["`+orchard.String()+`","`+unknown+`"]}`, http.StatusOK,
		`{"results":[{"id":"id1","done":true},{"id":"id2","done":false,"refusal":{"code":"not_found"}}]}`)
	w.o.Expect(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"announce","ids":["`+harbor.String()+`"]}`, http.StatusOK,
		`{"results":[{"id":"id1","done":false,"refusal":{"code":"window","acceptsAt":"T"}}]}`)
	w.o.Expect(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"remove","ids":["`+orchard.String()+`"],"deleteFiles":true,"dryRun":true}`, http.StatusOK,
		`{"results":[{"id":"id1","done":true}],"preview":{"cancels":[{"grabId":"id2","workId":"id3","release":"`+orchardRelease+`"}],"stopSeeding":[],"frees":900,"linked":false,"hitAndRun":[]}}`)
	w.o.Expect(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"pause","ids":["`+orchard.String()+`"],"dryRun":true}`, http.StatusBadRequest,
		`{"status":400,"code":"no_preview","title":"Only remove, mark failed and recheck can be previewed"}`)

	w.roster.away[w.self] = true
	w.o.Expect(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"recheck","ids":["`+harbor.String()+`"]}`, http.StatusOK,
		`{"results":[{"id":"id1","done":false,"refusal":{"code":"offline"}}]}`)
	var got struct {
		Rows []struct {
			State     string  `json:"state"`
			Pending   string  `json:"pending"`
			LastKnown *string `json:"lastKnown"`
		} `json:"rows"`
	}
	decodeReply(t, w, "/api/v1/torrents?ids="+orchard.String(), &got)
	if len(got.Rows) != 1 || got.Rows[0].State != "paused" || got.Rows[0].Pending != "pause" || got.Rows[0].LastKnown == nil {
		t.Errorf("the paused Torrent on its offline Node = %+v, want paused, the pause pending and its last known time", got.Rows)
	}
}

// TestMarkFailedBlocklistsAHeldTorrentInTheNameOfTheAdminActing marks
// failed a Torrent a downloading Grab holds as Ada's session and another
// through the API key: each entry names who acted. A Torrent no Grab holds
// is refused as not held, and one a Grab imports from as importing.
func TestMarkFailedBlocklistsAHeldTorrentInTheNameOfTheAdminActing(t *testing.T) {
	w := serveWorld(t)
	orchard := w.hold(t, "Orchard", queuedReport())
	attic := w.hold(t, "Attic", queuedReport())
	harbor := w.hold(t, "Harbor", queuedReport())
	cove := w.hold(t, "Cove", queuedReport())
	work := works.WorkID{UUID: storage.NewUUID()}
	w.uses[orchard] = []downloadclients.Use{{Grab: storage.NewUUID(), Work: work, Release: orchardRelease, Stage: downloadclients.UseDownloading}}
	w.uses[attic] = []downloadclients.Use{{Grab: storage.NewUUID(), Work: work, Release: "Attic.2020.1080p.WEB", Stage: downloadclients.UseDownloading}}
	w.uses[cove] = []downloadclients.Use{{Grab: storage.NewUUID(), Work: work, Release: "Cove.2021.1080p.WEB", Stage: downloadclients.UseImporting}}

	status, reply := w.o.CallSignedIn(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"mark-failed","ids":["`+orchard.String()+`","`+harbor.String()+`","`+cove.String()+`"]}`)
	if want := `{"results":[{"id":"id1","done":true},{"id":"id2","done":false,"refusal":{"code":"not_held"}},{"id":"id3","done":false,"refusal":{"code":"importing","workId":"id4"}}]}`; status != http.StatusOK || strings.TrimSpace(ownertest.Masked(reply)) != want {
		t.Fatalf("Ada marking failed = %d %s, want 200 %s", status, ownertest.Masked(reply), want)
	}
	w.o.Expect(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"mark-failed","ids":["`+attic.String()+`"]}`, http.StatusOK,
		`{"results":[{"id":"id1","done":true}]}`)
	if want := []string{"Ada", "the API key"}; !slices.Equal(w.blocklist.by, want) {
		t.Errorf("the Blocklist entries name %q, want %q", w.blocklist.by, want)
	}
}

func TestDownloadClientsListEachDownloadingNodesCardAndTheThresholds(t *testing.T) {
	w := serveWorld(t)
	w.hold(t, "Harbor", unregisteredSeeding())
	w.facts[w.self] = &dlv1.Facts{
		Libtorrent: "2.0.11.0", PeerIdPrefix: "-LT20B0-", UserAgent: "libtorrent/2.0.11.0", Listen: "0.0.0.0:6881", Bound: "192.0.2.4:6881",
		IncomingAtUnixMs: reportedAt.UnixMilli(),
	}
	thresholds := `"thresholds":{"faultStall":{"key":"downloading.faultStall","label":"A download with no data at all is bad after","source":"database","applies":"live",` +
		`"isDefault":true,"pendingRestart":false,"kind":"duration","value":43200,"running":43200,"default":43200,"min":3600,"max":2592000,"step":3600},` +
		`"unregisteredGap":{"key":"downloading.unregisteredGap","label":"Unregistered twice, this far apart","source":"database","applies":"live",` +
		`"isDefault":true,"pendingRestart":false,"kind":"duration","value":1800,"running":1800,"default":1800,"min":60,"max":604800,"step":60},` +
		`"metadataWait":{"key":"downloading.metadataWait","label":"A magnet link with no metadata fails after","source":"database","applies":"live",` +
		`"isDefault":true,"pendingRestart":false,"kind":"duration","value":1800,"running":1800,"default":1800,"min":60,"max":604800,"step":60},` +
		`"clientErrorFail":{"key":"downloading.clientErrorFail","label":"A client error fails its Grab after","source":"database","applies":"live",` +
		`"isDefault":true,"pendingRestart":false,"kind":"duration","value":3600,"running":3600,"default":3600,"min":300,"max":604800,"step":60}}`
	acts := `"actions":["pause","resume","recheck","announce"]`
	kinds := `"kinds":[` +
		`{"kind":"qbittorrent","name":"qBittorrent","path":"","username":"optional","password":"optional","apiKey":true,"daemon":false,"label":"any","floor":"4.6",` + acts + `,"deletesData":true},` +
		`{"kind":"transmission","name":"Transmission","path":"/transmission/rpc","username":"optional","password":"optional","apiKey":false,"daemon":false,"label":"no_comma","floor":"4.0",` + acts + `,"deletesData":true},` +
		`{"kind":"deluge","name":"Deluge","path":"/json","username":"none","password":"required","apiKey":false,"daemon":true,"label":"lower_word","floor":"2.1",` + acts + `,"deletesData":true},` +
		`{"kind":"rtorrent","name":"rTorrent","path":"/RPC2","username":"optional","password":"optional","apiKey":false,"daemon":false,"label":"any","floor":"0.9.8",` + acts + `,"deletesData":false}]`
	w.o.Expect(t, http.MethodGet, "/api/v1/download-clients", "", http.StatusOK,
		`{"cards":[{"nodeId":"id1","nodeName":"home","grabsTo":null,"externals":[],"inStep":true,"client":{"libtorrent":"2.0.11.0","peerIdPrefix":"-LT20B0-","userAgent":"libtorrent/2.0.11.0",`+
			`"listen":"0.0.0.0:6881","bound":"192.0.2.4:6881","reach":"reachable","reachedAt":"T"},`+
			`"limits":{"downloadLimit":0,"uploadLimit":0,"activeDownloads":3,"activeChecks":1,"connections":8000,"utp":true,"dht":true,"stalledAfterSeconds":600},`+
			`"totals":{"uploaded":0,"downloaded":0,"seedingSeconds":0,"activeSeconds":0},"torrents":1,"trackers":[]}],"orphans":[],`+kinds+`,`+thresholds+`}`)

	_, reply := w.o.Call(t, http.MethodPatch, "/api/v1/download-clients/thresholds", `{"unregisteredGap":3600,"metadataWait":300}`)
	if want := `"value":3600`; !strings.Contains(reply, want) || !strings.Contains(reply, `"value":300,`) {
		t.Errorf("setDownloadClientThresholds = %s, want the gap at an hour and the metadata wait at 5 minutes", reply)
	}
	if !slices.Equal(w.heard.touched, []role.NodeID{w.self}) || len(w.heard.cards) != 1 || !slices.Equal(w.heard.cards[0], []string{w.self.String()}) {
		t.Errorf("a threshold change touched %v and told %v, want the downloading Node's intent and download-clients.changed for it", w.heard.touched, w.heard.cards)
	}
	w.o.Expect(t, http.MethodPatch, "/api/v1/download-clients/thresholds", `{"faultStall":60}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 1 h and 30 days."}`)
	if _, reply := w.o.Call(t, http.MethodPatch, "/api/v1/download-clients/thresholds", `{"clientErrorFail":600}`); !strings.Contains(reply, `"value":600,`) {
		t.Errorf("setDownloadClientThresholds = %s, want the client error to fail its Grab after 10 minutes", reply)
	}
	if _, reply := w.o.Call(t, http.MethodGet, "/api/v1/download-clients", ""); !strings.Contains(reply, `"clientErrorFail":{"key":"downloading.clientErrorFail","label":"A client error fails its Grab after","source":"database","applies":"live","isDefault":false`) {
		t.Errorf("listDownloadClients = %s, want the stored client error setting", reply)
	}
	w.o.Expect(t, http.MethodPatch, "/api/v1/download-clients/thresholds", `{"clientErrorFail":60}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 5 min and 7 days."}`)

	delete(w.facts, w.self)
	w.roster.away[w.self] = true
	_, reply = w.o.Call(t, http.MethodGet, "/api/v1/download-clients", "")
	if !strings.Contains(reply, `"inStep":false,"limits"`) {
		t.Errorf("listDownloadClients with the Node out of step = %s, want its card with no client", reply)
	}
}

func TestARowShowsItsTrackerAndItsSeedingAndListsByTrackerAndGoalMet(t *testing.T) {
	w := serveWorld(t)
	tracker := func(name, privacy string) storage.UUID {
		id := storage.NewUUID()
		w.exec(t, `INSERT INTO indexers_tracker (id, name, privacy, created_at) VALUES (?, ?, ?, 0)`, id, name, privacy)
		return id
	}
	seedpool, nyaa := tracker("seedpool", "private"), tracker("Nyaa", "public")
	complete := &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_SEEDING, Complete: true}
	orchard, harbor := w.hold(t, "Orchard", complete), w.hold(t, "Harbor", complete)
	w.exec(t, `UPDATE downloadclients_torrent SET tracker_id = ?, uploaded = 250, downloaded = 1000 WHERE id = ?`, seedpool, orchard)
	w.exec(t, `UPDATE downloadclients_torrent SET tracker_id = ?, uploaded = 3000, downloaded = 1000 WHERE id = ?`, nyaa, harbor)
	w.uses[orchard] = []downloadclients.Use{{Grab: storage.NewUUID(), Release: orchardRelease, Stage: downloadclients.UseImported, Floor: seeding.RatioRule{Ratio: 100}}}

	type row struct {
		ID      string          `json:"id"`
		Tracker json.RawMessage `json:"tracker"`
		Seeding json.RawMessage `json:"seeding"`
	}
	var got struct {
		Rows []row `json:"rows"`
	}
	decodeReply(t, w, "/api/v1/torrents?tracker="+seedpool.String(), &got)
	want := row{
		ID: orchard.String(), Tracker: json.RawMessage(`{"id":"` + seedpool.String() + `","name":"seedpool"}`),
		Seeding: json.RawMessage(`{"ratio":{"value":0.25,"rule":1,"goal":2},"time":{"value":0,"goal":1209600},"state":"owing",` +
			`"owes":{"seconds":0,"bytes":750},"atRisk":"no","goalMet":false,"combine":"first","action":"remove-delete","endless":"none","clears":"ratio"}`),
	}
	if len(got.Rows) != 1 || string(got.Rows[0].Tracker) != string(want.Tracker) || string(got.Rows[0].Seeding) != string(want.Seeding) {
		t.Errorf("seedpool's torrents = %+v, want only Orchard owing seedpool 750 bytes of upload: %s %s", got.Rows, want.Tracker, want.Seeding)
	}

	decodeReply(t, w, "/api/v1/torrents?tab=goal-met", &got)
	if len(got.Rows) != 1 || got.Rows[0].ID != harbor.String() {
		t.Errorf("the Goal met tab = %+v, want public Harbor past ratio 2", got.Rows)
	}
}

func TestRemovingTheLastCopyOfATorrentThatOwesPreviewsAHitAndRun(t *testing.T) {
	w := serveWorld(t)
	seedpool, nyaa := storage.NewUUID(), storage.NewUUID()
	w.exec(t, `INSERT INTO indexers_tracker (id, name, privacy, created_at) VALUES (?, 'seedpool', 'private', 0), (?, 'Nyaa', 'public', 0)`, seedpool, nyaa)
	complete := &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_SEEDING, Complete: true}
	harbor, orchard, quay := w.hold(t, "Harbor", complete), w.hold(t, "Orchard", complete), w.hold(t, "Quay", complete)
	w.exec(t, `UPDATE downloadclients_torrent SET tracker_id = ?, uploaded = 3000, downloaded = 1000 WHERE id = ?`, nyaa, harbor)
	w.exec(t, `UPDATE downloadclients_torrent SET tracker_id = ?, uploaded = 250, downloaded = 1000 WHERE id = ?`, seedpool, orchard)
	w.exec(t, `UPDATE downloadclients_torrent SET tracker_id = ?, downloaded = 1000 WHERE id = ?`, seedpool, quay)
	w.uses[orchard] = []downloadclients.Use{{Grab: storage.NewUUID(), Release: orchardRelease, Stage: downloadclients.UseImported, Floor: seeding.RatioRule{Ratio: 100}}}
	ids := `["` + harbor.String() + `","` + orchard.String() + `","` + quay.String() + `"]`

	w.o.Expect(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"remove","ids":`+ids+`,"dryRun":true}`, http.StatusOK,
		`{"results":[{"id":"id1","done":true},{"id":"id2","done":true},{"id":"id3","done":true}],"preview":{"cancels":[],"stopSeeding":[],"frees":0,"linked":false,`+
			`"hitAndRun":[{"torrentId":"id2","tracker":{"id":"id4","name":"seedpool"},"owes":{"seconds":0,"bytes":750},"clears":"ratio"},{"torrentId":"id3","tracker":{"id":"id4","name":"seedpool"},"clears":"none"}]}}`)

	var got struct {
		Seeding struct {
			Endless string `json:"endless"`
		} `json:"seeding"`
	}
	decodeReply(t, w, "/api/v1/torrents/"+quay.String(), &got)
	if got.Seeding.Endless != "no-rule" {
		t.Errorf("Quay, owing seedpool under no known rule, seeds with no end because %q, want no-rule", got.Seeding.Endless)
	}
}

func (w *world) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, sql, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
