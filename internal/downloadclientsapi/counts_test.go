package downloadclientsapi_test

import (
	"net/http"
	"strings"
	"testing"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
)

func TestTorrentCountsAnswerEachDownloadingNodeInNodeOrder(t *testing.T) {
	w := serveWorld(t)
	seedbox := w.downloader(t, "seedbox")
	w.hold(t, "Orchard", &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Done: 250})
	w.hold(t, "Harbor", &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Stalled: true})
	w.hold(t, "Meadow", &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_FAILED})
	w.hold(t, "Lantern", &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_SEEDING, Complete: true})
	w.hold(t, "Willow", queuedReport())

	status, reply := w.o.Call(t, http.MethodGet, "/api/v1/torrents/counts", "")
	want := `{"nodes":[{"nodeId":"` + w.self.String() + `","torrents":5,"downloading":1,"seeding":1,"failures":1,"warnings":1},` +
		`{"nodeId":"` + seedbox.String() + `","torrents":0,"downloading":0,"seeding":0,"failures":0,"warnings":0}]}`
	if status != http.StatusOK || strings.TrimSpace(reply) != want {
		t.Fatalf("GET /torrents/counts = %d %s\nwant 200 %s", status, reply, want)
	}
}
