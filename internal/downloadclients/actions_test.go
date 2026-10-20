package downloadclients_test

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// roster is the Nodes as a test sets them: in step or not, written off or
// not, serving Downloading or not, with a change closing changed.
type roster struct {
	mu         sync.Mutex
	away       map[role.NodeID]bool
	writtenOff map[role.NodeID]bool
	serving    []role.NodeID
	changed    chan struct{}
}

func (r *roster) Serving(name role.Name) []role.NodeID {
	r.mu.Lock()
	defer r.mu.Unlock()
	if name != role.Downloading {
		return nil
	}
	return slices.Clone(r.serving)
}

func (r *roster) InStep(node role.NodeID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.away[node]
}

func (r *roster) WrittenOff(name role.Name, node role.NodeID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return name == role.Downloading && r.writtenOff[node]
}

func (r *roster) Changed() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.changed == nil {
		r.changed = make(chan struct{})
	}
	return r.changed
}

func (r *roster) change(f func()) {
	r.Changed()
	r.mu.Lock()
	defer r.mu.Unlock()
	f()
	close(r.changed)
	r.changed = nil
}

func (r *roster) writeOff(node role.NodeID) {
	r.change(func() {
		if r.writtenOff == nil {
			r.writtenOff = map[role.NodeID]bool{}
		}
		r.writtenOff[node] = true
	})
}

func (r *roster) leave(node role.NodeID) {
	r.change(func() {
		if r.away == nil {
			r.away = map[role.NodeID]bool{}
		}
		r.away[node] = true
	})
}

// uses is the Grabs of each Torrent as a test sets them.
type uses map[dc.ID][]dc.Use

func (u uses) Of(_ context.Context, _ storage.Reader[storage.Library], ids []dc.ID) (map[dc.ID][]dc.Use, error) {
	out := map[dc.ID][]dc.Use{}
	for _, id := range ids {
		out[id] = u[id]
	}
	return out, nil
}

// client is a Node's torrent client as the Controller calls it: it
// answers Act with its refusals and Frees with its counts, and records
// what it was asked.
type client struct {
	downloadingv1connect.UnimplementedDownloadingServiceHandler
	refuse map[dc.ID]*dlv1.ActResult
	freed  map[dc.ID]*dlv1.Freed

	mu    sync.Mutex
	asked []dlv1.Act
}

func (c *client) Act(_ context.Context, req *dlv1.ActRequest) (*dlv1.ActResponse, error) {
	c.mu.Lock()
	c.asked = append(c.asked, req.GetAct())
	c.mu.Unlock()
	res := &dlv1.ActResponse{}
	for _, raw := range req.GetIds() {
		r := &dlv1.ActResult{Id: raw}
		if refused, ok := c.refuse[idFrom(raw)]; ok {
			r = refused
		}
		res.Results = append(res.Results, r)
	}
	return res, nil
}

func (c *client) Frees(_ context.Context, req *dlv1.FreesRequest) (*dlv1.FreesResponse, error) {
	res := &dlv1.FreesResponse{}
	for _, raw := range req.GetIds() {
		f := &dlv1.Freed{Id: raw}
		if got, ok := c.freed[idFrom(raw)]; ok {
			f = got
		}
		res.Freed = append(res.Freed, f)
	}
	return res, nil
}

func idFrom(raw []byte) dc.ID {
	u, _ := storage.UUIDFromBytes(raw)
	return dc.ID{UUID: u}
}

// heard is what the Clients told its readers.
type heard struct {
	mu      sync.Mutex
	touched []role.NodeID
	changed []string
}

func (h *heard) Publish(_ context.Context, c dc.Changed) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.changed = append(h.changed, c.IDs...)
}

func (h *heard) touch(ids ...role.NodeID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.touched = append(h.touched, ids...)
}

func (h *heard) touchedNodes() []role.NodeID {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.touched)
}

func (h *heard) changedIDs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.changed)
}

func newClients(lib library, r *roster, u uses, node *client) (*dc.Clients, *heard) {
	h := &heard{}
	if node == nil {
		node = &client{}
	}
	return dc.New(dc.Config{
		Library: lib, Touch: h.touch, Changed: h, Roster: r, Uses: u, Log: slog.New(slog.DiscardHandler),
		Remote: func(role.NodeID) downloadingv1connect.DownloadingServiceHandler { return node },
		Live:   dc.LiveConfig{Watched: func() []role.NodeID { return nil }, Event: &admin{}, Sample: liveSample},
	}), h
}

// runClients runs a Clients until the test ends.
func runClients(t *testing.T, lib library, r *roster, u uses, node *client) *heard {
	t.Helper()
	c, h := newClients(lib, r, u, node)
	ctx, stop := context.WithCancel(context.WithoutCancel(t.Context()))
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = c.Run(ctx)
	}()
	t.Cleanup(func() {
		stop()
		<-done
	})
	synctest.Wait()
	return h
}

func act(t *testing.T, c *dc.Clients, a dc.Action) ([]dc.Result, *dc.Preview) {
	t.Helper()
	results, preview, err := c.Act(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	return results, preview
}

func heldOn(t *testing.T, lib library, node role.NodeID) nodes.Held {
	t.Helper()
	var held nodes.Held
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		held, err = dc.DownloadingOwner.Held(ctx, r, node)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return held
}

func refusals(results []dc.Result) []*dc.Refusal {
	out := make([]*dc.Refusal, len(results))
	for i, r := range results {
		out[i] = r.Refusal
	}
	return out
}

func wantedOf(t *testing.T, lib library, id dc.ID) string {
	t.Helper()
	var w string
	if err := lib.QueryRowContext(t.Context(), "SELECT wanted FROM downloadclients_torrent WHERE id = ?", id).Scan(&w); err != nil {
		t.Fatal(err)
	}
	return w
}

// shelf is a Node holding three Torrents of the Orchard, the Attic and
// the Harbor, the last one complete.
type shelf struct {
	lib                    library
	box                    role.NodeID
	orchard, attic, harbor dc.ID
}

func newShelf(t *testing.T) shelf {
	t.Helper()
	lib := open(t)
	box := addNode(t, lib, "seedbox")
	add := func(name string) dc.ID {
		m, raw := dotTorrent(t, name)
		id, _ := mustAttach(t, lib, dc.Want{Node: box, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
		return id
	}
	s := shelf{lib: lib, box: box, orchard: add("Orchard"), attic: add("Attic"), harbor: add("Harbor")}
	apply(t, lib, box, s.harbor, completed())
	return s
}

var (
	orchardWork = works.WorkID{UUID: storage.NewUUID()}
	harborWork  = works.WorkID{UUID: storage.NewUUID()}
)

// TestPauseAndResumeSetTheWantedStateForTheIntent pauses two Torrents,
// one of them twice, then resumes one: each sets the wanted state the
// Node's intent carries, touches the Node and says what changed, whether
// the Node is in step or not; a removed Torrent refuses.
func TestPauseAndResumeSetTheWantedStateForTheIntent(t *testing.T) {
	s := newShelf(t)
	r := &roster{}
	r.leave(s.box)
	c, h := newClients(s.lib, r, nil, nil)
	removeTorrent(t, s.lib, s.harbor, dc.ByAdmin)
	missing := dc.ID{UUID: storage.NewUUID()}

	results, _ := act(t, c, dc.Action{Kind: dc.Pause, IDs: []dc.ID{s.orchard, s.attic, s.harbor, missing}})
	if got := refusals(results); got[0] != nil || got[1] != nil || got[2].Code != dc.RefusedRemoved || got[3].Code != dc.RefusedNotFound {
		t.Fatalf("pausing answered %+v, want the two live ones paused, the removed and the missing ones refused", got)
	}
	if wantedOf(t, s.lib, s.orchard) != "paused" || wantedOf(t, s.lib, s.attic) != "paused" {
		t.Fatalf("pausing left the Torrents %s and %s, want both paused", wantedOf(t, s.lib, s.orchard), wantedOf(t, s.lib, s.attic))
	}
	if got := h.touchedNodes(); len(got) == 0 || got[0] != s.box {
		t.Errorf("pausing touched %v, want the offline seedbox's intent all the same", got)
	}
	if got := h.changedIDs(); !slices.Equal(got, slices.Sorted(slices.Values([]string{s.orchard.String(), s.attic.String()}))) {
		t.Errorf("pausing said %v changed, want the two paused Torrents", got)
	}

	act(t, c, dc.Action{Kind: dc.Pause, IDs: []dc.ID{s.orchard}})
	act(t, c, dc.Action{Kind: dc.Resume, IDs: []dc.ID{s.attic}})
	if wantedOf(t, s.lib, s.orchard) != "paused" || wantedOf(t, s.lib, s.attic) != "started" {
		t.Errorf("after a second pause and a resume the Torrents are %s and %s, want paused and started", wantedOf(t, s.lib, s.orchard), wantedOf(t, s.lib, s.attic))
	}
}

// TestRemoveIsRefusedDuringImport removes the Orchard while one of its
// Grabs imports, and the Harbor whose Grab downloaded and imported it:
// the first is refused naming the Work, and the second leaves the intent
// for the discards with its download copy to delete.
func TestRemoveIsRefusedDuringImport(t *testing.T) {
	s := newShelf(t)
	u := uses{
		s.orchard: {{Grab: storage.NewUUID(), Work: orchardWork, Stage: dc.UseImporting}, {Grab: storage.NewUUID(), Work: orchardWork, Stage: dc.UseFailed}},
		s.harbor:  {{Grab: storage.NewUUID(), Work: harborWork, Stage: dc.UseImported}},
	}
	c, _ := newClients(s.lib, &roster{}, u, nil)

	for _, dryRun := range []bool{true, false} {
		results, _ := act(t, c, dc.Action{Kind: dc.Remove, IDs: []dc.ID{s.orchard, s.harbor}, DeleteFiles: true, DryRun: dryRun})
		if got := refusals(results); got[0] == nil || *got[0] != (dc.Refusal{Code: dc.RefusedImporting, Work: orchardWork}) || got[1] != nil {
			t.Fatalf("removing, dry run %v, answered %+v, want the Orchard refused while its Work imports and the Harbor removed", dryRun, got)
		}
	}
	in := intentWithDiscards(t, s.lib, s.box)
	if len(in.GetTorrents()) != 2 || len(in.GetDiscards()) != 1 || !in.GetDiscards()[0].GetDeleteFiles() || idFrom(in.GetDiscards()[0].GetId()) != s.harbor {
		t.Fatalf("seedbox's intent is %v, want the Orchard and the Attic held and the Harbor discarded with its files", in)
	}
	if got := outcomeOf(t, s.lib, s.harbor).Outcome; got != (dc.Gone{Reason: dc.ByAdmin}) {
		t.Errorf("the removed Harbor reads %#v, want Gone by the Admin", got)
	}
	results, _ := act(t, c, dc.Action{Kind: dc.Remove, IDs: []dc.ID{s.harbor}})
	if got := refusals(results); got[0] == nil || got[0].Code != dc.RefusedRemoved {
		t.Errorf("removing the Harbor again answered %+v, want it refused as removed", got)
	}
}

func intentWithDiscards(t *testing.T, lib library, node role.NodeID) *dlv1.Intent {
	t.Helper()
	var in *dlv1.Intent
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		in, err = dc.Intent(ctx, r, node)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return in
}

// TestARemovePreviewNamesWhatItCancelsStopsAndFrees previews removing all
// three Torrents: it cancels the Orchard's downloading Grab, stops the
// complete Harbor seeding for its Work, frees what the Node counts and
// changes nothing; with the Node away the bytes are unknown.
func TestARemovePreviewNamesWhatItCancelsStopsAndFrees(t *testing.T) {
	s := newShelf(t)
	downloading := dc.Use{Grab: storage.NewUUID(), Work: orchardWork, Release: "Orchard.2026.1080p", Stage: dc.UseDownloading}
	u := uses{s.orchard: {downloading}, s.harbor: {{Grab: storage.NewUUID(), Work: harborWork, Stage: dc.UseImported}}}
	node := &client{freed: map[dc.ID]*dlv1.Freed{
		s.orchard: {Id: s.orchard.Bytes(), Bytes: 600},
		s.harbor:  {Id: s.harbor.Bytes(), Bytes: 24, Linked: true},
	}}
	r := &roster{}
	c, _ := newClients(s.lib, r, u, node)

	_, preview := act(t, c, dc.Action{Kind: dc.Remove, IDs: []dc.ID{s.orchard, s.attic, s.harbor}, DryRun: true})
	if preview == nil || !slices.Equal(preview.Cancels, []dc.Use{downloading}) || !slices.Equal(preview.StopSeeding, []works.WorkID{harborWork}) ||
		preview.Frees == nil || *preview.Frees != 624 || !preview.Linked {
		t.Fatalf("the preview is %+v, want the Orchard's Grab canceled, the Harbor's Work no longer seeded, 624 bytes freed and a library file", preview)
	}
	if got := intentOf(t, s.lib, s.box); len(got) != 3 {
		t.Errorf("after a preview seedbox's intent holds %v, want all three", got)
	}

	r.leave(s.box)
	if _, preview := act(t, c, dc.Action{Kind: dc.Remove, IDs: []dc.ID{s.orchard}, DryRun: true}); preview.Frees != nil {
		t.Errorf("with seedbox away the preview frees %d bytes, want unknown", *preview.Frees)
	}
	if _, _, err := c.Act(t.Context(), dc.Action{Kind: dc.Pause, IDs: []dc.ID{s.orchard}, DryRun: true}); err == nil {
		t.Errorf("a dry run of pause answered, want ErrNoPreview")
	}
}

// TestMarkFailedRemovesOnlyAStalledDownloadWithItsFiles marks a stalled
// and a running download failed: the stalled one is removed with its
// download copy for mark-failed, the other refused.
func TestMarkFailedRemovesOnlyAStalledDownloadWithItsFiles(t *testing.T) {
	s := newShelf(t)
	apply(t, s.lib, s.box, s.orchard, &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Stalled: true})
	apply(t, s.lib, s.box, s.attic, &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING})
	c, _ := newClients(s.lib, &roster{}, nil, nil)

	results, _ := act(t, c, dc.Action{Kind: dc.MarkFailed, IDs: []dc.ID{s.orchard, s.attic}})
	if got := refusals(results); got[0] != nil || got[1] == nil || got[1].Code != dc.RefusedNotStalled {
		t.Fatalf("marking failed answered %+v, want the stalled one done and the other refused", got)
	}
	if got := outcomeOf(t, s.lib, s.orchard).Outcome; got != (dc.Gone{Reason: dc.ByMarkFailed}) {
		t.Errorf("the marked download reads %#v, want Gone by mark-failed", got)
	}
	if d := intentWithDiscards(t, s.lib, s.box).GetDiscards(); len(d) != 1 || !d[0].GetDeleteFiles() {
		t.Errorf("seedbox's discards are %v, want the marked download with its files", d)
	}
}

// TestAnnounceAndRecheckAskEachTorrentsNode asks seedbox to announce two
// Torrents, one inside a tracker's window, and then to recheck while it
// is away: the Node's answers come back per Torrent, and an away Node's
// Torrents are refused as offline without a call.
func TestAnnounceAndRecheckAskEachTorrentsNode(t *testing.T) {
	s := newShelf(t)
	acceptsAt := reportedAt.Add(20 * time.Minute)
	node := &client{refuse: map[dc.ID]*dlv1.ActResult{
		s.attic: {Id: s.attic.Bytes(), Refused: dlv1.Refusal_REFUSAL_WINDOW, AcceptsAtUnixMs: acceptsAt.UnixMilli()},
	}}
	r := &roster{}
	c, _ := newClients(s.lib, r, nil, node)

	results, _ := act(t, c, dc.Action{Kind: dc.Announce, IDs: []dc.ID{s.orchard, s.attic}})
	if got := refusals(results); got[0] != nil || got[1] == nil || *got[1] != (dc.Refusal{Code: dc.RefusedWindow, AcceptsAt: acceptsAt}) {
		t.Fatalf("announcing answered %+v, want the Orchard announced and the Attic refused until %v", got, acceptsAt)
	}

	r.leave(s.box)
	results, _ = act(t, c, dc.Action{Kind: dc.Recheck, IDs: []dc.ID{s.orchard}})
	if got := refusals(results); got[0] == nil || got[0].Code != dc.RefusedOffline {
		t.Fatalf("rechecking on an away Node answered %+v, want offline", got)
	}
	if !slices.Equal(node.asked, []dlv1.Act{dlv1.Act_ACT_ANNOUNCE}) {
		t.Errorf("the Node was asked %v, want the announce alone", node.asked)
	}
}

// TestARemovingTorrentHoldsItsNodeUntilParted is a drain's view of a
// removal: a Removing Torrent still counts as held work until the Node
// confirms it, so the Node keeps Downloading long enough to delete its
// files and send stopped.
func TestARemovingTorrentHoldsItsNodeUntilParted(t *testing.T) {
	s := newShelf(t)
	for _, id := range []dc.ID{s.orchard, s.attic, s.harbor} {
		removeTorrent(t, s.lib, id, dc.ByAdmin)
	}
	if held := heldOn(t, s.lib, s.box); held.Units != 3 || held.Endless {
		t.Fatalf("with every Torrent Removing seedbox holds %+v, want 3 it will finish", held)
	}
	for _, id := range []dc.ID{s.orchard, s.attic, s.harbor} {
		partTorrent(t, s.lib, id)
	}
	if held := heldOn(t, s.lib, s.box); held.Units != 0 {
		t.Errorf("once every removal parted seedbox holds %+v, want nothing", held)
	}
}
