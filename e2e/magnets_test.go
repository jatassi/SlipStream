package e2e

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/dht"
	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type magnetRelease struct {
	name, key, placed string
}

var (
	trackerlessOrchard = magnetRelease{"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW", "9000001", "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv"}
	harborMagnet       = magnetRelease{"Harbor.Lights.2019.1080p.WEB-DL.H264-FKW", "9000002", "movies/Harbor Lights (2019)/Harbor Lights (2019) - 1080p.mkv"}
	meadowRedirect     = magnetRelease{"Salt.Meadow.2020.1080p.WEB-DL.H264-FKW", "9000003", "movies/Salt Meadow (2020)/Salt Meadow (2020) - 1080p.mkv"}
	privateHollow      = magnetRelease{"Cedar.Hollow.2018.1080p.WEB-DL.H264-FKW", "9000004", ""}
	privateFalls       = magnetRelease{"Juniper.Falls.2022.1080p.WEB-DL.H264-FKW", "9000005", "movies/Juniper Falls (2022)/Juniper Falls (2022) - 1080p.mkv"}
)

func grabbed(t *testing.T, a *worldtest.API, rel magnetRelease) string {
	t.Helper()
	work := decode(t, "addWork", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+rel.key+`","monitor":true}`), &added{})
	var tr tree
	if err := json.Unmarshal(work.Work, &tr); err != nil {
		t.Fatal(err)
	}
	if r := a.Admin("home", http.MethodPost, "/api/v1/works/"+tr.ID+"/search", ""); r.Status != http.StatusAccepted {
		t.Fatalf("searchWork %s = %d %s, want 202", rel.name, r.Status, r.Body)
	}
	return tr.ID
}

func endedGrab(t *testing.T, a *worldtest.API, work string) grabView {
	t.Helper()
	var g grabView
	worldtest.Until(t, "the Grab of "+work+" is imported or failed", func() (bool, error) {
		var list struct {
			Grabs []struct {
				grabView
				WorkID string `json:"workId"`
			} `json:"grabs"`
		}
		decode(t, "listGrabs", a.Admin("home", http.MethodGet, "/api/v1/grabs", ""), &list)
		for _, row := range list.Grabs {
			if row.WorkID == work {
				g = row.grabView
			}
		}
		return g.Stage == "imported" || g.Stage == "failed", nil
	})
	return g
}

func infoHashOf(t *testing.T, m world.Manifest, name string) string {
	t.Helper()
	for _, r := range m.Releases {
		if r.Name == name {
			return r.InfoHash
		}
	}
	t.Fatalf("the manifest lists no release %s", name)
	return ""
}

func TestMagnetLinksGetTheirMetadataFromTheSwarm(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "magnets")
	m := run.Manifest()
	a := api(t, run)
	if r := a.Admin("home", http.MethodPatch, "/api/v1/download-clients/thresholds", `{"metadataWait":60}`); r.Status != http.StatusOK {
		t.Fatalf("setDownloadClientThresholds = %d %s, want 200", r.Status, r.Body)
	}
	all := []magnetRelease{trackerlessOrchard, harborMagnet, meadowRedirect, privateHollow, privateFalls}
	works := map[string]string{}
	for _, rel := range all {
		works[rel.name] = grabbed(t, a, rel)
	}

	for _, rel := range []magnetRelease{trackerlessOrchard, harborMagnet, meadowRedirect, privateFalls} {
		g := endedGrab(t, a, works[rel.name])
		if g.Stage != "imported" || g.InfoHash != infoHashOf(t, m, rel.name) {
			t.Fatalf("the Grab of %s ended %s with %+v after %v, want %s imported", rel.name, g.Stage, g.Failure, g.stages(), g.InfoHash)
		}
		if _, err := world.MatchRelease(m, filepath.Join(m.MediaDir, filepath.FromSlash(rel.placed)), rel.name); err != nil {
			t.Fatalf("%s: %v", rel.name, err)
		}
	}
	g := endedGrab(t, a, works[privateHollow.name])
	if g.Stage != "failed" || g.Failure == nil || g.Failure.Code != "no_metadata" || !strings.HasPrefix(g.Failure.Message, "No peer sent the metadata in 1 min") {
		t.Fatalf("the Grab of the private magnet link ended %s with %+v after %v, want failed no_metadata", g.Stage, g.Failure, g.stages())
	}

	orchard := infoHashOf(t, m, trackerlessOrchard.name)
	looked := map[string]bool{}
	for _, rec := range selected(t, run, events.Query{Kind: dht.Lookups.Kind, Node: "home", Match: map[string]string{"method": "get_peers"}}) {
		var q dht.Query
		if err := json.Unmarshal(rec.Data, &q); err != nil {
			t.Fatal(err)
		}
		looked[q.InfoHash] = true
	}
	if !looked[orchard] {
		t.Errorf("fake-dht saw no get_peers for the trackerless release %s, only %v", orchard, looked)
	}
	for _, private := range []magnetRelease{privateHollow, privateFalls} {
		h := infoHashOf(t, m, private.name)
		if looked[h] || len(selected(t, run, events.Query{Kind: dht.Lookups.Kind, Match: map[string]string{"infoHash": h}})) > 0 {
			t.Errorf("fake-dht saw the private release %s's info hash %s", private.name, h)
		}
	}
	if n := len(selected(t, run, events.Query{Kind: tracker.Announces.Kind, Node: "home", Match: map[string]string{"infoHash": orchard}})); n > 0 {
		t.Errorf("the tracker saw %d announces of the trackerless release from home, want none", n)
	}
}
