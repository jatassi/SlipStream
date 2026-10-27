package downloadclients_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// inspector is a Node's client that answers Inspect for the Torrents it
// holds.
type inspector struct {
	downloadingv1connect.UnimplementedDownloadingServiceHandler
	held map[dc.ID]*dlv1.InspectResponse
}

func (i inspector) Inspect(_ context.Context, req *dlv1.InspectRequest) (*dlv1.InspectResponse, error) {
	if res, ok := i.held[idFrom(req.GetId())]; ok {
		return res, nil
	}
	return nil, errors.New("this Node holds no such torrent")
}

// listed is a Controller holding Torrents on seedbox, in step, and attic,
// out of step since its last report.
type listed struct {
	lib     library
	r       *roster
	u       uses
	seedbox role.NodeID
	attic   role.NodeID
	work    works.WorkID
	held    map[string]dc.ID
	node    inspector
}

func newListed(t *testing.T) *listed {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), health.Schema(), roots.Schema(), dc.Schema(), indexers.Schema())
	l := &listed{lib: lib, u: uses{}, held: map[string]dc.ID{}, work: works.WorkID{UUID: storage.NewUUID()}, node: inspector{held: map[dc.ID]*dlv1.InspectResponse{}}}
	l.seedbox, l.attic = addNode(t, lib, "seedbox"), addNode(t, lib, "attic")
	l.r = &roster{serving: []role.NodeID{l.seedbox, l.attic}, away: map[role.NodeID]bool{l.attic: true}}
	exec(t, lib, "INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'movie', '1', 'The Quiet Orchard', 'quiet orchard', 0, 0)", l.work)
	return l
}

func (l *listed) add(t *testing.T, node role.NodeID, name string, r *dlv1.TorrentReport) dc.ID {
	t.Helper()
	m, raw := dotTorrent(t, name)
	id, _ := mustAttach(t, l.lib, dc.Want{Node: node, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	if r != nil {
		apply(t, l.lib, node, id, r)
	}
	l.held[name] = id
	return id
}

func (l *listed) clients() *dc.Clients {
	return dc.New(dc.Config{
		Library: l.lib, Touch: func(...role.NodeID) {}, Changed: published{}, CardsChanged: &told{}, Roster: l.r, Uses: l.u,
		Remote: func(role.NodeID) downloadingv1connect.DownloadingServiceHandler { return l.node },
		Facts:  func(role.NodeID) (*dlv1.Facts, bool) { return nil, false },
		Live:   dc.LiveConfig{Watched: func() []role.NodeID { return nil }, Event: &admin{}, Sample: liveSample},
		Log:    slog.New(slog.DiscardHandler),
	})
}

func queued(trackers ...*dlv1.TrackerLine) *dlv1.TorrentReport {
	return &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Hold: dlv1.Hold_HOLD_SLOT, Trackers: trackers}
}

// shown is what a test reads of a row.
type shown struct {
	State     dc.State
	InLine    int
	Pending   dc.Wanted
	Tabs      dc.Tab
	Tracker   dc.TrackerCell
	Fault     dc.FaultKind
	LastKnown time.Time
}

func shownOf(r dc.Row) shown {
	s := shown{State: r.State, InLine: r.QueuePosition, Pending: r.Pending, Tabs: r.Tabs, LastKnown: r.LastKnown}
	if r.Tracker != nil {
		s.Tracker = *r.Tracker
	}
	if r.Fault != nil {
		s.Fault = r.Fault.Kind
	}
	return s
}

func list(t *testing.T, c *dc.Clients, f dc.Filter) dc.Listing {
	t.Helper()
	got, err := c.List(t.Context(), f)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func names(l *listed, rows []dc.Row) []string {
	var out []string
	for _, r := range rows {
		for name, id := range l.held {
			if id == r.ID {
				out = append(out, name)
			}
		}
	}
	return out
}

func TestTheListDerivesEachTorrentsWordsAndCountsThemAll(t *testing.T) {
	l := newListed(t)
	working := &dlv1.TrackerLine{Index: 1, Status: dlv1.TrackerStatus_TRACKER_STATUS_WORKING}
	unregistered := &dlv1.TrackerLine{Index: 0, Status: dlv1.TrackerStatus_TRACKER_STATUS_UNREGISTERED, Message: "Unregistered torrent", RetryAtUnixMs: reportedAt.Add(time.Hour).UnixMilli()}
	l.add(t, l.seedbox, "Orchard", queued(working))
	l.add(t, l.seedbox, "Attic", queued())
	paused := l.add(t, l.seedbox, "Pantry", queued())
	l.add(t, l.seedbox, "Cellar", queued())
	seeding := completed()
	seeding.Trackers = []*dlv1.TrackerLine{unregistered, working}
	seeding.Fault = &dlv1.Fault{Kind: dlv1.FaultKind_FAULT_KIND_UNREGISTERED, AtUnixMs: reportedAt.UnixMilli()}
	l.add(t, l.seedbox, "Harbor", seeding)
	gone := l.add(t, l.seedbox, "Shed", queued())
	removeTorrent(t, l.lib, gone, dc.ByAdmin)
	l.add(t, l.attic, "Loft", queued(&dlv1.TrackerLine{Index: 0, Status: dlv1.TrackerStatus_TRACKER_STATUS_ERROR, Message: "timed out"}))
	l.add(t, l.attic, "Barn", nil)
	l.u[l.held["Harbor"]] = []dc.Use{{Grab: storage.NewUUID(), Work: l.work, Release: "Harbor.Lights.2019.1080p.WEB-DL", Stage: dc.UseImported}}
	c := l.clients()
	act(t, c, dc.Action{Kind: dc.Pause, IDs: []dc.ID{paused}})

	all := list(t, c, dc.Filter{})
	notContacted := dc.TrackerCell{Host: "tracker.test", Status: dc.TrackerNotContacted}
	want := map[string]shown{
		"Orchard": {State: dc.StateQueued, InLine: 1, Tabs: dc.TabDownloading | dc.TabAll, Tracker: dc.TrackerCell{Host: "backup.test", Status: dc.TrackerWorking}},
		"Attic":   {State: dc.StateQueued, InLine: 2, Tabs: dc.TabDownloading | dc.TabAll, Tracker: notContacted},
		"Pantry":  {State: dc.StatePaused, Pending: dc.Paused, Tabs: dc.TabDownloading | dc.TabAll, Tracker: notContacted},
		"Cellar":  {State: dc.StateQueued, InLine: 3, Tabs: dc.TabDownloading | dc.TabAll, Tracker: notContacted},
		"Harbor":  {State: dc.StateSeeding, Tabs: dc.TabSeeding | dc.TabAll, Tracker: dc.TrackerCell{Host: "backup.test", Status: dc.TrackerWorking}, Fault: dc.FaultUnregistered},
		"Shed":    {State: dc.StateRemoving, Tabs: dc.TabAll, Tracker: notContacted},
		"Loft": {State: dc.StateTrackerErr, Tabs: dc.TabProblems | dc.TabAll, LastKnown: reportedAt,
			Tracker: dc.TrackerCell{Host: "tracker.test", Status: dc.TrackerError, Message: "timed out"}},
	}
	got := map[string]shown{}
	for _, r := range all.Rows {
		got[names(l, []dc.Row{r})[0]] = shownOf(r)
	}
	barn := got["Barn"]
	delete(got, "Barn")
	for name, w := range want {
		if g := got[name]; g != w {
			t.Errorf("%s = %+v, want %+v", name, g, w)
		}
	}
	if barn.State != dc.StateQueued || barn.LastKnown.IsZero() || barn.InLine != 0 {
		t.Errorf("Barn, never reported on attic while it is out of step, = %+v, want queued, in no line, last known when added", barn)
	}

	wantStates := map[dc.State]int{dc.StateQueued: 4, dc.StatePaused: 1, dc.StateSeeding: 1, dc.StateRemoving: 1, dc.StateTrackerErr: 1}
	wantTabs := map[dc.Tab]int{dc.TabDownloading: 5, dc.TabSeeding: 1, dc.TabProblems: 1, dc.TabAll: 8}
	wantNodes := map[role.NodeID]int{l.seedbox: 6, l.attic: 2}
	for _, f := range []dc.Filter{{}, {Tab: dc.TabSeeding}, {IDs: []dc.ID{paused}}} {
		got := list(t, c, f).Counts
		if !equalMaps(got.States, wantStates) || !equalMaps(got.Tabs, wantTabs) || !equalMaps(got.Nodes, wantNodes) {
			t.Errorf("counts under %+v = %+v, want every Torrent counted whatever the filter", f, got)
		}
	}

	for _, c2 := range []struct {
		filter dc.Filter
		want   []string
	}{
		{dc.Filter{Tab: dc.TabSeeding}, []string{"Harbor"}},
		{dc.Filter{Tab: dc.TabProblems}, []string{"Loft"}},
		{dc.Filter{Node: l.attic}, []string{"Loft", "Barn"}},
		{dc.Filter{IDs: []dc.ID{l.held["Cellar"], l.held["Orchard"]}}, []string{"Orchard", "Cellar"}},
		{dc.Filter{Search: "CELLAR"}, []string{"Cellar"}},
		{dc.Filter{Search: "lights.2019"}, []string{"Harbor"}},
		{dc.Filter{Search: "quiet orchard"}, []string{"Harbor"}},
		{dc.Filter{Type: contenttypetest.ID, Tab: dc.TabSeeding}, []string{"Harbor"}},
		{dc.Filter{Type: "series"}, nil},
	} {
		if got := names(l, list(t, c, c2.filter).Rows); !slices.Equal(got, c2.want) {
			t.Errorf("listed under %+v: %v, want %v", c2.filter, got, c2.want)
		}
	}
}

func equalMaps[K comparable](a, b map[K]int) bool {
	for k, v := range b {
		if a[k] != v {
			return false
		}
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestTheInspectorReadsTheNodeLiveAndTheLastReportOtherwise(t *testing.T) {
	l := newListed(t)
	answered := reportedAt.Add(-time.Minute)
	report := queued(&dlv1.TrackerLine{Index: 1, Status: dlv1.TrackerStatus_TRACKER_STATUS_ERROR, Message: "timed out", RetryAtUnixMs: reportedAt.Add(time.Hour).UnixMilli()})
	id := l.add(t, l.seedbox, "Orchard", report)
	file := 1
	l.u[id] = []dc.Use{{Grab: storage.NewUUID(), Work: l.work, Item: works.ItemID{UUID: storage.NewUUID()}, Stage: dc.UseImported, File: &file}}
	l.node.held[id] = &dlv1.InspectResponse{
		Files:            []*dlv1.FileProgress{{Index: 0, Done: 400}, {Index: 1, Done: 24}},
		Trackers:         []*dlv1.TrackerDetail{{Index: 0, Status: dlv1.TrackerStatus_TRACKER_STATUS_WORKING, AnsweredAtUnixMs: answered.UnixMilli(), NextAnnounceUnixMs: answered.Add(30 * time.Minute).UnixMilli(), MinIntervalSeconds: 1800, AcceptsAtUnixMs: answered.Add(30 * time.Minute).UnixMilli()}},
		AnnounceAtUnixMs: answered.Add(30 * time.Minute).UnixMilli(),
	}
	c := l.clients()

	d, err := c.Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Live || !d.AnnounceAt.Equal(answered.Add(30*time.Minute)) || !d.Private || d.PieceSize != 16384 {
		t.Errorf("detail = live %t, announce at %s, private %t, piece size %d; want the Node's word, private, 16 KiB pieces", d.Live, d.AnnounceAt, d.Private, d.PieceSize)
	}
	wantTrackers := []dc.TrackerDetail{
		{URL: "https://tracker.test/announce/•••", Tier: 0, Status: dc.TrackerWorking, AnsweredAt: answered, NextAnnounce: answered.Add(30 * time.Minute), MinInterval: 30 * time.Minute, AcceptsAt: answered.Add(30 * time.Minute)},
		{URL: "udp://backup.test:6969/announce", Tier: 1, Status: dc.TrackerError, Message: "timed out", RetryAt: reportedAt.Add(time.Hour)},
	}
	if !slices.EqualFunc(d.Trackers, wantTrackers, sameTracker) {
		t.Errorf("trackers = %+v, want %+v", d.Trackers, wantTrackers)
	}
	if len(d.Files) != 2 || d.Files[0].Path != "Orchard/Orchard.mkv" || *d.Files[0].Done != 400 || !d.Files[0].Item.IsZero() ||
		d.Files[1].Path != "Orchard/Sample/sample.mkv" || *d.Files[1].Done != 24 || d.Files[1].Item != l.u[id][0].Item {
		t.Errorf("files = %+v, want both with the Node's progress, the second with the Item its Grab imported", d.Files)
	}

	l.r.leave(l.seedbox)
	d, err = c.Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Live || !d.LastKnown.Equal(reportedAt) || d.Files[0].Done != nil || d.Trackers[0].Status != dc.TrackerNotContacted || d.Trackers[1].Status != dc.TrackerError {
		t.Errorf("detail while seedbox is out of step = %+v, want its last report: progress unknown, the second tracker failing", d)
	}

	removeTorrent(t, l.lib, id, dc.ByAdmin)
	partTorrent(t, l.lib, id)
	if _, err := c.Get(t.Context(), id); !errors.Is(err, dc.ErrNotFound) {
		t.Errorf("Get of a parted Torrent = %v, want ErrNotFound", err)
	}
	if _, err := c.Get(t.Context(), dc.ID{UUID: storage.NewUUID()}); !errors.Is(err, dc.ErrNotFound) {
		t.Errorf("Get of an unknown id = %v, want ErrNotFound", err)
	}
}

func sameTracker(a, b dc.TrackerDetail) bool {
	return a.URL == b.URL && a.Tier == b.Tier && a.Status == b.Status && a.Message == b.Message && a.MinInterval == b.MinInterval &&
		a.RetryAt.Equal(b.RetryAt) && a.AnsweredAt.Equal(b.AnsweredAt) && a.NextAnnounce.Equal(b.NextAnnounce) && a.AcceptsAt.Equal(b.AcceptsAt)
}

func TestACardReadsItsNodesClientFactsSettingsAndTotals(t *testing.T) {
	l := newListed(t)
	reached := reportedAt.Add(-time.Hour)
	facts := map[role.NodeID]*dlv1.Facts{l.seedbox: {
		Libtorrent: "2.0.11", PeerIdPrefix: "-LT20B0-", UserAgent: "libtorrent/2.0.11", Listen: "0.0.0.0:6881", Bound: "192.0.2.4:6881",
		IncomingAtUnixMs: reached.UnixMilli(),
	}}
	harbor := l.add(t, l.seedbox, "Harbor", completed())
	l.add(t, l.seedbox, "Orchard", queued())
	exec(t, l.lib, "UPDATE downloadclients_torrent SET uploaded = 3000, downloaded = 1024, seeding_seconds = 600 WHERE id = ?", harbor)
	gone := l.add(t, l.seedbox, "Shed", queued())
	exec(t, l.lib, "UPDATE downloadclients_torrent SET uploaded = 99999 WHERE id = ?", gone)
	removeTorrent(t, l.lib, gone, dc.ByAdmin)
	cards := &told{}
	c := dc.New(dc.Config{
		Library: l.lib, Touch: func(...role.NodeID) {}, Changed: published{}, CardsChanged: cards, Roster: l.r,
		Facts: func(n role.NodeID) (*dlv1.Facts, bool) { f, ok := facts[n]; return f, ok },
	})
	limit := int64(1 << 20)
	if _, err := c.SetNodeSettings(t.Context(), l.seedbox, dc.NodeChange{DownloadLimit: parsed(dc.DownloadLimit.Parse(&limit))}); err != nil {
		t.Fatal(err)
	}

	got, err := c.Cards(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Node != l.seedbox || got[1].Node != l.attic {
		t.Fatalf("cards = %+v, want seedbox's then attic's", got)
	}
	box := got[0]
	if !box.InStep || box.Facts == nil || *box.Facts != (dc.ClientFacts{Libtorrent: "2.0.11", PeerIDPrefix: "-LT20B0-", UserAgent: "libtorrent/2.0.11", Listen: "0.0.0.0:6881", Bound: "192.0.2.4:6881", ReachedAt: reached}) ||
		box.Facts.Reach() != dc.Reachable || box.Settings.DownloadLimit != limit || box.Torrents != 2 ||
		box.Totals.Uploaded != 3000 || box.Totals.Downloaded != 1024 || box.Totals.Seeding != 10*time.Minute {
		t.Errorf("seedbox's card = %+v (facts %+v), want its facts, reachable, its limit and its two live Torrents' totals", box, box.Facts)
	}
	if attic := got[1]; attic.InStep || attic.Facts != nil || attic.Torrents != 0 {
		t.Errorf("attic's card = %+v, want out of step with no facts", attic)
	}
	if (dc.ClientFacts{}).Reach() != dc.Unproven {
		t.Error("facts with no peer connected in read reachable, want unproven")
	}

	c.Arrived(l.attic, string(role.Storage))
	c.Arrived(l.attic, string(role.Downloading))
	if want := [][]string{{l.seedbox.String()}, {l.attic.String()}}; !slices.EqualFunc(cards.nodes, want, slices.Equal) {
		t.Errorf("download-clients.changed named %v, want seedbox for its settings, then attic for its Downloading facts alone", cards.nodes)
	}
}
