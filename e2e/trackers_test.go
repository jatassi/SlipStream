package e2e

import (
	"net"
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world"
)

type trackerView struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Privacy    string   `json:"privacy"`
	Hosts      []string `json:"hosts"`
	Keys       []string `json:"keys"`
	IndexerIDs []string `json:"indexerIds"`
}

// addProwlarr creates Fixture Prowlarr, a Torznab Indexer at the fixture
// site's Prowlarr host at priority 10, so it outranks Fixture Site. Its
// items name site.Source.
func (w *walked) addProwlarr(t *testing.T) string {
	t.Helper()
	i := slices.IndexFunc(w.m.Fronts, func(f world.Front) bool { return f.Target == "fixture-site" && f.Node == "seedbox" })
	if i < 0 {
		t.Fatal("the manifest lists no fixture-site front for seedbox")
	}
	u := must(url.Parse(w.m.Fronts[i].URL))
	u.Host = site.Host(site.Prowlarr)
	if p := must(url.Parse(w.m.Fronts[i].URL)).Port(); p != "" {
		u.Host = net.JoinHostPort(u.Host, p)
	}
	u.Path = site.TorznabPath
	body := `{"kind":"torznab","name":"Fixture Prowlarr","address":"` + u.String() + `","apiKey":"` + site.APIKey(w.s) +
		`","on":true,"automaticSearch":true,"priority":10}`
	reply := w.a.Admin("home", http.MethodPost, "/api/v1/indexers", body)
	if reply.Status != http.StatusCreated {
		t.Fatalf("createIndexer = %d %s, want 201", reply.Status, reply.Body)
	}
	reply.Status = http.StatusOK
	return decode(t, "createIndexer", reply, &struct {
		ID string `json:"id"`
	}{}).ID
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func (w *walked) trackers(t *testing.T) []trackerView {
	t.Helper()
	var list struct {
		Trackers []trackerView `json:"trackers"`
	}
	decode(t, "listTrackers", w.a.Admin("home", http.MethodGet, "/api/v1/trackers", ""), &list)
	return list.Trackers
}

type trackedGrab struct {
	TrackerID *string `json:"trackerId"`
	TorrentID *string `json:"torrentId"`
}

type trackedTorrent struct {
	TrackerID *string `json:"trackerId"`
	Trackers  []struct {
		URL string `json:"url"`
	} `json:"trackers"`
}

// trackedBy reads Grab id's Tracker and its Torrent's.
func (w *walked) trackedBy(t *testing.T, id string) (grab, torrent string, announce []string) {
	t.Helper()
	g := decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+id, ""), &trackedGrab{})
	if g.TrackerID == nil || g.TorrentID == nil {
		t.Fatalf("Grab %s has Tracker %v and Torrent %v, want both", id, g.TrackerID, g.TorrentID)
	}
	tr := decode(t, "getTorrent", w.a.Admin("home", http.MethodGet, "/api/v1/torrents/"+*g.TorrentID, ""), &trackedTorrent{})
	if tr.TrackerID == nil {
		t.Fatalf("Torrent %s names no Tracker", *g.TorrentID)
	}
	for _, a := range tr.Trackers {
		announce = append(announce, must(url.Parse(a.URL)).Hostname())
	}
	return *g.TrackerID, *tr.TrackerID, announce
}

// TestATorznabSourceBecomesATrackerAtItsFirstGrabAndMergesWithItsGrabAndTorrent
// adds a feed whose items name a source next to Fixture Site, whose feed
// names none. The source becomes a Tracker only when a Grab takes one of
// its Releases, gathers the private torrent's announce host, and a merge
// into Fixture Site's Tracker moves the Grab and the Torrent.
func TestATorznabSourceBecomesATrackerAtItsFirstGrabAndMergesWithItsGrabAndTorrent(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	prowlarr := w.addProwlarr(t)
	before := w.trackers(t)
	if len(before) != 1 || before[0].Name != "Fixture Site" || len(before[0].IndexerIDs) != 1 {
		t.Fatalf("before any Grab the Trackers are %+v, want only Fixture Site's", before)
	}
	fixture := before[0]

	s := w.searched(t, w.search(t))
	if *s.Outcome != "grabbed" || w.grabbedFrom(t, *s.GrabID) != prowlarr {
		t.Fatalf("the search ended %+v, want a Grab from Fixture Prowlarr %s", s, prowlarr)
	}
	w.ended(t, *s.GrabID)
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v, want imported", w.grab.Stage, w.grab.Failure)
	}
	grabbed, attached, announce := w.trackedBy(t, *s.GrabID)
	var source trackerView
	decode(t, "getTracker", w.a.Admin("home", http.MethodGet, "/api/v1/trackers/"+grabbed, ""), &source)
	if source.Name != site.Source || source.Privacy != "private" || !slices.Equal(source.Keys, []string{"prowlarr:" + prowlarr + ":1"}) ||
		!slices.Equal(source.IndexerIDs, []string{prowlarr}) || attached != grabbed || len(announce) != 1 || !slices.Equal(source.Hosts, announce) {
		t.Fatalf("the Grab reached %+v and its Torrent %s announcing to %v; want %s, private, keyed by the feed, holding that announce host, on both",
			source, attached, announce, site.Source)
	}

	merged := decode(t, "mergeTracker", w.a.Admin("home", http.MethodPost, "/api/v1/trackers/"+source.ID+"/merge", `{"into":"`+fixture.ID+`"}`), &trackerView{})
	if merged.ID != fixture.ID || merged.Name != "Fixture Site" || len(merged.Keys) != 2 || len(merged.IndexerIDs) != 2 || !slices.Equal(merged.Hosts, announce) {
		t.Fatalf("the merge kept %+v, want Fixture Site with both keys, both Indexers and the announce host", merged)
	}
	if grabbed, attached, _ = w.trackedBy(t, *s.GrabID); grabbed != fixture.ID || attached != fixture.ID {
		t.Fatalf("after the merge the Grab names Tracker %s and the Torrent %s, want Fixture Site's %s", grabbed, attached, fixture.ID)
	}
	if r := w.a.Admin("home", http.MethodGet, "/api/v1/trackers/"+source.ID, ""); r.Status != http.StatusNotFound {
		t.Fatalf("getTracker of the merged Tracker = %d %s, want 404", r.Status, r.Body)
	}
}
