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
		id, _, err = downloadclients.Attach(ctx, tx, downloadclients.Want{Node: w.self, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: func(role.NodeID) bool { return false }}, storage.Now())
		if err != nil || report == nil {
			return err
		}
		report.Id = id.Bytes()
		c := downloadclients.New(downloadclients.Config{Touch: func(...role.NodeID) {}, Changed: torrentsChanged{}})
		return c.Apply(ctx, tx, w.self, &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{report}}}})
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

func seeding() *dlv1.TorrentReport {
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
	harbor := w.hold(t, "Harbor", seeding())
	work := works.WorkID{UUID: storage.NewUUID()}
	w.uses[orchard] = []downloadclients.Use{{Grab: storage.NewUUID(), Work: work, Edition: "any", Release: orchardRelease, Stage: downloadclients.UseDownloading}}

	counts := func(node string) string {
		return `"counts":{"byState":[{"state":"removing","torrents":0},{"state":"failed","torrents":0},{"state":"tracker-error","torrents":1},` +
			`{"state":"paused","torrents":0},{"state":"checking","torrents":0},{"state":"queued","torrents":1},{"state":"stalled","torrents":0},` +
			`{"state":"downloading","torrents":0},{"state":"seeding","torrents":0}],` +
			`"byTab":[{"tab":"downloading","torrents":1},{"tab":"seeding","torrents":0},{"tab":"problems","torrents":1},{"tab":"all","torrents":2}],` +
			`"byNode":[{"nodeId":"` + node + `","torrents":2}]}}`
	}
	orchardRow := `{"id":"id1","nodeId":"id2","name":"Orchard","contentType":"fixture","workId":"id3","edition":"any","quality":"webdl-1080p",` +
		`"state":"queued","hold":"slot","queuePosition":1,"severity":"none","tabs":["downloading","all"],"size":1000,"done":250,"uploaded":0,` +
		`"ratio":0,"seedingSeconds":0,"tracker":{"host":"beacon.test","status":"not_contacted"},"addedAt":"T"}`
	w.o.Expect(t, http.MethodGet, "/api/v1/torrents?tab=downloading", "", http.StatusOK, `{"rows":[`+orchardRow+`],`+counts("id2"))

	harborRow := `{"id":"id1","nodeId":"id2","name":"Harbor","contentType":"fixture","state":"tracker-error","severity":"failure",` +
		`"tabs":["problems","all"],"size":1000,"done":1000,"uploaded":0,"ratio":0,"seedingSeconds":0,` +
		`"tracker":{"host":"beacon.test","status":"unregistered","message":"Unregistered torrent","retryAt":"T"},` +
		`"fault":{"kind":"unregistered","at":"T","sentence":"The release's tracker reports that the torrent is no longer registered."},"addedAt":"T"}`
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

func TestATorrentReadsLiveFromItsNodeAndRefusesAnUnknownID(t *testing.T) {
	w := serveWorld(t)
	harbor := w.hold(t, "Harbor", seeding())
	answered := reportedAt.Add(-time.Minute)
	w.node.inspected = map[downloadclients.ID]*dlv1.InspectResponse{harbor: {
		Files:            []*dlv1.FileProgress{{Index: 0, Done: 1000}},
		Trackers:         []*dlv1.TrackerDetail{{Index: 1, Status: dlv1.TrackerStatus_TRACKER_STATUS_WORKING, AnsweredAtUnixMs: answered.UnixMilli(), MinIntervalSeconds: 1800}},
		AnnounceAtUnixMs: answered.Add(30 * time.Minute).UnixMilli(),
	}}
	file, item := 0, works.ItemID{UUID: storage.NewUUID()}
	w.uses[harbor] = []downloadclients.Use{{Grab: storage.NewUUID(), Work: works.WorkID{UUID: storage.NewUUID()}, Item: item, Release: "Harbor.2019.720p.HDTV", Stage: downloadclients.UseImported, File: &file}}

	_, reply := w.o.Call(t, http.MethodGet, "/api/v1/torrents/"+harbor.String(), "")
	want := `{"id":"id1","nodeId":"id2","name":"Harbor","contentType":"fixture","workId":"id3","quality":"hdtv-720p","state":"tracker-error","severity":"failure",` +
		`"tabs":["problems","all"],"size":1000,"done":1000,"uploaded":0,"ratio":0,"seedingSeconds":0,` +
		`"tracker":{"host":"beacon.test","status":"unregistered","message":"Unregistered torrent","retryAt":"T"},` +
		`"fault":{"kind":"unregistered","at":"T","sentence":"The release's tracker reports that the torrent is no longer registered."},"addedAt":"T",` +
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

func TestActionsAnswerForEachTorrent(t *testing.T) {
	w := serveWorld(t)
	orchard := w.hold(t, "Orchard", queuedReport())
	harbor := w.hold(t, "Harbor", seeding())
	unknown := storage.NewUUID().String()
	work := works.WorkID{UUID: storage.NewUUID()}
	w.uses[orchard] = []downloadclients.Use{{Grab: storage.NewUUID(), Work: work, Release: orchardRelease, Stage: downloadclients.UseDownloading}}
	w.node.acceptsAt, w.node.freed = reportedAt.Add(20*time.Minute), 900

	w.o.Expect(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"pause","ids":["`+orchard.String()+`","`+unknown+`"]}`, http.StatusOK,
		`{"results":[{"id":"id1","done":true},{"id":"id2","done":false,"refusal":{"code":"not_found"}}]}`)
	w.o.Expect(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"announce","ids":["`+harbor.String()+`"]}`, http.StatusOK,
		`{"results":[{"id":"id1","done":false,"refusal":{"code":"window","acceptsAt":"T"}}]}`)
	w.o.Expect(t, http.MethodPost, "/api/v1/torrents/actions", `{"action":"remove","ids":["`+orchard.String()+`"],"deleteFiles":true,"dryRun":true}`, http.StatusOK,
		`{"results":[{"id":"id1","done":true}],"preview":{"cancels":[{"grabId":"id2","workId":"id3","release":"`+orchardRelease+`"}],"stopSeeding":[],"frees":900,"linked":false}}`)
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

func TestDownloadClientsListEachDownloadingNodesCardAndTheThresholds(t *testing.T) {
	w := serveWorld(t)
	w.hold(t, "Harbor", seeding())
	w.facts[w.self] = &dlv1.Facts{
		Libtorrent: "2.0.11.0", PeerIdPrefix: "-LT20B0-", UserAgent: "libtorrent/2.0.11.0", Listen: "0.0.0.0:6881", Bound: "192.0.2.4:6881",
		IncomingAtUnixMs: reportedAt.UnixMilli(),
	}
	thresholds := `"thresholds":{"faultStall":{"key":"downloading.faultStall","label":"A download with no data at all is bad after","source":"database","applies":"live",` +
		`"isDefault":true,"pendingRestart":false,"kind":"duration","value":43200,"running":43200,"default":43200,"min":3600,"max":2592000,"step":3600},` +
		`"unregisteredGap":{"key":"downloading.unregisteredGap","label":"Unregistered twice, this far apart","source":"database","applies":"live",` +
		`"isDefault":true,"pendingRestart":false,"kind":"duration","value":1800,"running":1800,"default":1800,"min":60,"max":604800,"step":60}}`
	w.o.Expect(t, http.MethodGet, "/api/v1/download-clients", "", http.StatusOK,
		`{"cards":[{"nodeId":"id1","nodeName":"home","inStep":true,"client":{"libtorrent":"2.0.11.0","peerIdPrefix":"-LT20B0-","userAgent":"libtorrent/2.0.11.0",`+
			`"listen":"0.0.0.0:6881","bound":"192.0.2.4:6881","reach":"reachable","reachedAt":"T"},`+
			`"limits":{"downloadLimit":0,"uploadLimit":0,"activeDownloads":3,"activeChecks":1,"connections":8000,"utp":true,"stalledAfterSeconds":600},`+
			`"totals":{"uploaded":0,"downloaded":0,"seedingSeconds":0,"activeSeconds":0},"torrents":1}],`+thresholds+`}`)

	_, reply := w.o.Call(t, http.MethodPatch, "/api/v1/download-clients/thresholds", `{"unregisteredGap":3600}`)
	if want := `"value":3600`; !strings.Contains(reply, want) {
		t.Errorf("setDownloadClientThresholds = %s, want the gap at an hour", reply)
	}
	if !slices.Equal(w.heard.touched, []role.NodeID{w.self}) || len(w.heard.cards) != 1 || !slices.Equal(w.heard.cards[0], []string{w.self.String()}) {
		t.Errorf("a threshold change touched %v and told %v, want the downloading Node's intent and download-clients.changed for it", w.heard.touched, w.heard.cards)
	}
	w.o.Expect(t, http.MethodPatch, "/api/v1/download-clients/thresholds", `{"faultStall":60}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 1 h and 30 days."}`)

	delete(w.facts, w.self)
	w.roster.away[w.self] = true
	_, reply = w.o.Call(t, http.MethodGet, "/api/v1/download-clients", "")
	if !strings.Contains(reply, `"inStep":false,"limits"`) {
		t.Errorf("listDownloadClients with the Node out of step = %s, want its card with no client", reply)
	}
}
