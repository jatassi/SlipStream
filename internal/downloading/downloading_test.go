package downloading_test

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // BitTorrent v1 piece hashes are SHA-1 by spec (BEP 3)
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"math/rand/v2"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

const trackerURL = "https://tracker.example/announce"

const pieceLength = 16384

// size is every test torrent's size: three pieces, the last one short.
const size = 40000

var timings = downloading.DefaultTimings()

// file is a single-file torrent the tests download.
type file struct {
	name    string
	payload []byte
	meta    []byte
	hash    bencode.InfoHash
}

func newFile(t *testing.T, name string) file {
	t.Helper()
	return makeFile(t, name, map[string]any{})
}

// newPrivateFile is a torrent of a private tracker, which the announce
// policy holds to the tracker's window.
func newPrivateFile(t *testing.T, name string) file {
	t.Helper()
	return makeFile(t, name, map[string]any{"private": 1})
}

func makeFile(t *testing.T, name string, info map[string]any) file {
	t.Helper()
	payload := make([]byte, size)
	_, _ = rand.NewChaCha8([32]byte{name[0], name[1]}).Read(payload)
	var pieces []byte
	for p := payload; len(p) > 0; p = p[min(pieceLength, len(p)):] {
		sum := sha1.Sum(p[:min(pieceLength, len(p))]) //nolint:gosec // BitTorrent v1 piece hashes are SHA-1 by spec (BEP 3)
		pieces = append(pieces, sum[:]...)
	}
	info["length"], info["name"], info["piece length"], info["pieces"] = size, name, pieceLength, pieces
	meta, err := bencode.Encode(map[string]any{
		"announce": trackerURL,
		"info":     info,
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	return file{name: name, payload: payload, meta: meta, hash: m.InfoHash}
}

// holder is the Role's holder half as a test drives it: it yields the
// snapshots sent on intents and records the facts and reports.
type holder struct {
	intents chan role.Snapshot[*dlv1.Intent]

	mu      sync.Mutex
	facts   []*dlv1.Facts
	reports []*dlv1.Report
}

func (h *holder) Intent(ctx context.Context) iter.Seq[role.Snapshot[*dlv1.Intent]] {
	return func(yield func(role.Snapshot[*dlv1.Intent]) bool) {
		for {
			select {
			case s := <-h.intents:
				if !yield(s) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

func (h *holder) SetFacts(f *dlv1.Facts) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.facts = append(h.facts, f)
}

func (h *holder) Report(_ context.Context, _ *storage.Tx[storage.Local], r *dlv1.Report) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reports = append(h.reports, r)
	return nil
}

func (h *holder) reported() []*dlv1.Report {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.reports)
}

// reportsOf is every report of the torrent id, oldest first.
func (h *holder) reportsOf(id storage.UUID) []*dlv1.TorrentReport {
	var out []*dlv1.TorrentReport
	for _, r := range h.reported() {
		for _, tr := range r.GetTorrents().GetReports() {
			if bytes.Equal(tr.GetId(), id.Bytes()) {
				out = append(out, tr)
			}
		}
	}
	return out
}

func (h *holder) latest(t *testing.T, id storage.UUID) *dlv1.TorrentReport {
	t.Helper()
	reps := h.reportsOf(id)
	if len(reps) == 0 {
		t.Fatalf("no report of torrent %s", id)
	}
	return reps[len(reps)-1]
}

func (h *holder) parted() []*dlv1.Parted {
	var out []*dlv1.Parted
	for _, r := range h.reported() {
		if p := r.GetParted(); p != nil {
			out = append(out, p)
		}
	}
	return out
}

func (h *holder) totals() [][]*dlv1.TotalsRow {
	var out [][]*dlv1.TotalsRow
	for _, r := range h.reported() {
		if tot := r.GetTotals(); tot != nil {
			out = append(out, tot.GetRows())
		}
	}
	return out
}

func (h *holder) lastFacts() *dlv1.Facts {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.facts[len(h.facts)-1]
}

// control is the Controller as the client fetches from it.
type control struct {
	mu      sync.Mutex
	answers map[storage.UUID]*dlv1.TorrentResponse
	asked   []storage.UUID
}

func (c *control) Torrent(_ context.Context, req *dlv1.TorrentRequest) (*dlv1.TorrentResponse, error) {
	id, err := storage.UUIDFromBytes(req.GetId())
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.asked = append(c.asked, id)
	res, ok := c.answers[id]
	if !ok {
		return nil, rpc.Errorf(rpc.CodeNotFound, "no torrent %s", id)
	}
	return proto.CloneOf(res), nil
}

func (c *control) answer(id storage.UUID, res *dlv1.TorrentResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.answers[id] = res
}

func (c *control) fetched() []storage.UUID {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.asked)
}

// checks is an engine that counts the checks it is asked for, keeps the
// limits it runs with, and calls removing as it is asked to remove a
// torrent.
type checks struct {
	engine.Engine
	mu       sync.Mutex
	n        int
	limits   engine.Limits
	removing func()
}

func (c *checks) Remove(h engine.Handle, deleteFiles bool) error {
	c.mu.Lock()
	removing := c.removing
	c.mu.Unlock()
	if removing != nil {
		removing()
	}
	return c.Engine.Remove(h, deleteFiles)
}

func (c *checks) onRemove(f func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removing = f
}

func (c *checks) Configure(l engine.Limits) error {
	c.mu.Lock()
	c.limits = l
	c.mu.Unlock()
	return c.Engine.Configure(l)
}

func (c *checks) running() engine.Limits {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.limits
}

func (c *checks) Recheck(h engine.Handle) error {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return c.Engine.Recheck(h)
}

func (c *checks) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// node is one Node across its client's restarts: the World its engines
// open in, its local database, its downloads folder and its Controller.
type node struct {
	t     *testing.T
	world *enginetest.World
	local *storage.DB[storage.Local]
	dir   string
	ctl   *control
	// free is what the downloads folder's volume has left.
	free atomic.Int64
}

func newNode(t *testing.T, rules enginetest.TrackerRules) *node {
	t.Helper()
	w := enginetest.NewWorld()
	w.Tracker(trackerURL, rules)
	return &node{
		t: t, world: w, local: storagetest.Local(t, downloading.Schema()), dir: t.TempDir(),
		ctl: &control{answers: map[storage.UUID]*dlv1.TorrentResponse{}},
	}
}

var listen = netip.MustParseAddrPort("0.0.0.0:6881")

const volumeTotal = 1 << 40

// volume reads path itself, as statfs does, so a missing folder has none.
func (n *node) volume(_ context.Context, path string) (storing.Volume, error) {
	if _, err := os.Stat(path); err != nil {
		return storing.Volume{}, err
	}
	return storing.Volume{Mount: "/data", Name: "/data", Total: volumeTotal, Free: n.free.Load(), SampledAt: time.Now()}, nil
}

// hold has the Controller answer for f as a new Movies torrent.
func (n *node) hold(f file) storage.UUID {
	id := storage.NewUUID()
	n.ctl.answer(id, &dlv1.TorrentResponse{Metainfo: f.meta, ContentType: "Movies"})
	return id
}

type client struct {
	*downloading.Client
	h      *holder
	engine *checks
	stop   func()
}

// start runs a client on a new engine until stop saves its resume data
// and closes it.
func (n *node) start() *client {
	n.t.Helper()
	e, err := n.world.Open(engine.Config{ListenInterface: listen})
	if err != nil {
		n.t.Fatal(err)
	}
	eng := &checks{Engine: e}
	c := downloading.NewClient(downloading.Config{
		Engine: eng, Dir: n.dir, Listen: listen, Timings: timings, RetryMin: time.Second, RetryMax: time.Minute,
		Rand: rand.New(rand.NewPCG(1, 2)), ProvesPort: downloading.PublicPeer(false), Log: slog.New(slog.DiscardHandler), //nolint:gosec // seeded on purpose so tests replay
		Volume: n.volume,
	})
	h := &holder{intents: make(chan role.Snapshot[*dlv1.Intent])}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { _ = c.Loop(ctx) })
	wg.Go(func() {
		if err := c.Run(n.local, n.ctl)(ctx, h); err != nil {
			n.t.Errorf("run: %v", err)
		}
	})
	stop := sync.OnceFunc(func() {
		if err := c.Shutdown(n.local)(context.Background()); err != nil {
			n.t.Errorf("shutdown: %v", err)
		}
		cancel()
		_ = e.Close()
		wg.Wait()
	})
	n.t.Cleanup(stop)
	synctest.Wait()
	return &client{Client: c, h: h, engine: eng, stop: stop}
}

func (c *client) send(s role.Snapshot[*dlv1.Intent]) {
	c.h.intents <- s
	synctest.Wait()
}

var settings = &dlv1.Settings{ActiveDownloads: 3, ActiveChecks: 1, StalledAfterSeconds: 600}

// thresholds are the Controller's defaults.
var thresholds = &dlv1.Thresholds{FaultStallSeconds: 12 * 3600, UnregisteredGapSeconds: 30 * 60}

func live(ts ...*dlv1.Torrent) role.Snapshot[*dlv1.Intent] {
	return role.Snapshot[*dlv1.Intent]{Intent: &dlv1.Intent{Torrents: ts, Settings: settings, Thresholds: thresholds}, Standing: role.Live}
}

func (f file) started(id storage.UUID, order int64) *dlv1.Torrent {
	return &dlv1.Torrent{Id: id.Bytes(), InfoHash: f.hash[:], Wanted: dlv1.Wanted_WANTED_STARTED, Order: order}
}

func (f file) paused(id storage.UUID, order int64) *dlv1.Torrent {
	return &dlv1.Torrent{Id: id.Bytes(), InfoHash: f.hash[:], Wanted: dlv1.Wanted_WANTED_PAUSED, Order: order}
}

func startedAnnounces(w *enginetest.World, h bencode.InfoHash, since time.Time) []time.Time {
	var out []time.Time
	for _, a := range w.Announces(trackerURL) {
		if a.InfoHash == h && a.Event == "started" && !a.At.Before(since) {
			out = append(out, a.At)
		}
	}
	return out
}

// onceEach is the save folders and the completions reps carry.
func onceEach(reps []*dlv1.TorrentReport) (folders []string, completions []*dlv1.Completion) {
	for _, r := range reps {
		if r.GetSaveFolder() != "" {
			folders = append(folders, r.GetSaveFolder())
		}
		if r.GetCompletion() != nil {
			completions = append(completions, r.GetCompletion())
		}
	}
	return folders, completions
}

func TestANewTorrentIsFetchedCheckedAndSeedsInItsContentTypesFolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		c := n.start()
		c.send(live(f.started(id, 1)))

		folder := filepath.Join(n.dir, "Movies", f.hash.String())
		path, err := c.Path(f.hash, 0)
		if want := filepath.Join(folder, f.name); err != nil || path != want {
			t.Fatalf("Path = %q, %v; want %q", path, err, want)
		}
		if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, f.payload) {
			t.Fatalf("the download holds %d bytes, %v; want its %d", len(got), err, len(f.payload))
		}
		if got := c.engine.count(); got != 1 {
			t.Errorf("the client checked %d times, want once for a torrent it holds no data for", got)
		}
		if last := c.h.latest(t, id); last.GetEngine() != dlv1.EngineState_ENGINE_STATE_SEEDING || !last.GetComplete() {
			t.Errorf("the last report is %v, want complete and seeding", last)
		}
		folders, completions := onceEach(c.h.reportsOf(id))
		if !slices.Equal(folders, []string{folder}) {
			t.Errorf("reported save folders %q, want %q once", folders, folder)
		}
		want := &dlv1.Completion{Files: []*dlv1.File{{Index: 0, Path: f.name, Size: size}}}
		if len(completions) != 1 || !proto.Equal(completions[0], want) {
			t.Errorf("reported completions %v, want %v once", completions, want)
		}
	})
}

func TestAPausedTorrentHoldsUntilItIsWanted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		c := n.start()
		c.send(live(f.paused(id, 1)))

		if r := c.h.latest(t, id); r.GetHold() != dlv1.Hold_HOLD_WANTED || r.GetComplete() {
			t.Fatalf("a paused torrent reports %v, want held as wanted and incomplete", r)
		}
		if _, err := c.Path(f.hash, 0); !errors.Is(err, downloading.ErrNotPresent) {
			t.Fatalf("Path of a paused torrent = %v, want ErrNotPresent", err)
		}

		c.send(live(f.started(id, 1)))
		if r := c.h.latest(t, id); r.GetHold() != dlv1.Hold_HOLD_UNSPECIFIED || !r.GetComplete() {
			t.Fatalf("a started torrent reports %v, want complete with no hold", r)
		}
	})
}

// TestADownloadWaitsForAFreeSlot runs two downloads with one slot: the
// one ready first takes it, since their fetches race, and the other waits
// until it finishes.
func TestADownloadWaitsForAFreeSlot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		files := []file{newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv")}
		ids := []storage.UUID{n.hold(files[0]), n.hold(files[1])}
		for _, f := range files {
			n.world.Payload(f.hash, f.payload)
			n.world.Transfer(f.hash, 1000, 0)
		}
		c := n.start()
		one := live(files[0].started(ids[0], 1), files[1].started(ids[1], 2))
		one.Intent.Settings = &dlv1.Settings{ActiveDownloads: 1, ActiveChecks: 1, StalledAfterSeconds: 600}
		c.send(one)

		running, waiting := 0, 1
		if c.h.latest(t, ids[0]).GetHold() == dlv1.Hold_HOLD_SLOT {
			running, waiting = 1, 0
		}
		if r := c.h.latest(t, ids[running]); r.GetHold() != dlv1.Hold_HOLD_UNSPECIFIED || r.GetEngine() != dlv1.EngineState_ENGINE_STATE_DOWNLOADING {
			t.Fatalf("one download reports %v, want it downloading", r)
		}
		if r := c.h.latest(t, ids[waiting]); r.GetHold() != dlv1.Hold_HOLD_SLOT {
			t.Fatalf("the other download reports %v while the first runs, want held for a slot", r)
		}
		time.Sleep(40 * time.Second)
		synctest.Wait()
		if r := c.h.latest(t, ids[running]); !r.GetComplete() {
			t.Fatalf("the running download reports %v after its 40 s, want complete", r)
		}
		if r := c.h.latest(t, ids[waiting]); r.GetHold() != dlv1.Hold_HOLD_UNSPECIFIED || r.GetEngine() != dlv1.EngineState_ENGINE_STATE_DOWNLOADING {
			t.Fatalf("the waiting download reports %v once the slot freed, want downloading", r)
		}
		time.Sleep(40 * time.Second)
		synctest.Wait()
		if _, err := c.Path(files[waiting].hash, 0); err != nil {
			t.Fatalf("Path of the download that waited, after its 40 s = %v", err)
		}
	})
}

func TestSettingsApplyLiveFromTheIntent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		n.world.Transfer(f.hash, 1000, 0)
		id := n.hold(f)
		c := n.start()
		limited := live(f.started(id, 1))
		limited.Intent.Settings = &dlv1.Settings{DownloadLimit: 100, ActiveDownloads: 3, ActiveChecks: 1, Connections: 50, StalledAfterSeconds: 600}
		c.send(limited)
		if got, want := c.engine.running(), (engine.Limits{DownloadLimit: 100, Connections: 50, WatchIncoming: true}); got != want {
			t.Fatalf("the engine runs with %+v, want the intent's %+v", got, want)
		}
		time.Sleep(time.Minute)
		synctest.Wait()
		if c.h.latest(t, id).GetComplete() {
			t.Fatal("a 40 KB download held to 100 B/s is complete after a minute")
		}

		unlimited := live(f.started(id, 1))
		unlimited.Intent.Settings = &dlv1.Settings{ActiveDownloads: 3, ActiveChecks: 1, Connections: 8000, Utp: true, StalledAfterSeconds: 600}
		c.send(unlimited)
		if got, want := c.engine.running(), (engine.Limits{Connections: 8000, UTP: true, WatchIncoming: true}); got != want {
			t.Fatalf("the engine runs with %+v once the settings changed, want %+v", got, want)
		}
		time.Sleep(40 * time.Second)
		synctest.Wait()
		if !c.h.latest(t, id).GetComplete() {
			t.Fatal("the download is incomplete 40 s after its limit was lifted, want it done at 1000 B/s")
		}
	})
}

func TestADiscardRemovesTheTorrentAndConfirms(t *testing.T) {
	for name, deleteFiles := range map[string]bool{"keeping its files": false, "deleting its files": true} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { discards(t, deleteFiles) })
		})
	}
}

func discards(t *testing.T, deleteFiles bool) {
	n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
	f := newFile(t, "Orchard.mkv")
	n.world.Payload(f.hash, f.payload)
	id := n.hold(f)
	c := n.start()
	c.send(live(f.started(id, 1)))
	time.Sleep(timings.Facts)
	synctest.Wait()
	path, err := c.Path(f.hash, 0)
	if err != nil {
		t.Fatal(err)
	}

	discarding := live()
	discarding.Intent.Discards = []*dlv1.Discard{{Id: id.Bytes(), InfoHash: f.hash[:], DeleteFiles: deleteFiles}}
	c.send(discarding)
	want := &dlv1.Parted{Id: id.Bytes()}
	if deleteFiles {
		want.Freed = size
	}
	if got := c.h.parted(); len(got) != 1 || !proto.Equal(got[0], want) {
		t.Fatalf("confirmed %v, want %v", got, want)
	}
	if _, err := c.Path(f.hash, 0); !errors.Is(err, downloading.ErrNotPresent) {
		t.Errorf("Path once discarded = %v, want ErrNotPresent", err)
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) != deleteFiles {
		t.Errorf("the file once discarded, deleting files %v: stat = %v", deleteFiles, err)
	}

	c.send(discarding)
	again := &dlv1.Parted{Id: id.Bytes()}
	if got := c.h.parted(); len(got) != 2 || !proto.Equal(got[1], again) {
		t.Errorf("confirmed %v once the discard of a torrent it no longer holds was listed again, want it confirmed again as %v", got, again)
	}
}

// TestPathLetsATorrentGoBeforeTheEngineDoes has the engine read Path as
// the client asks it to remove a torrent, on a discard and on a release:
// Path must have let the torrent go by then, since the engine forgets its
// handle at once.
func TestPathLetsATorrentGoBeforeTheEngineDoes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		discarded, released := newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv")
		n.world.Payload(discarded.hash, discarded.payload)
		n.world.Payload(released.hash, released.payload)
		discardedID, releasedID := n.hold(discarded), n.hold(released)
		c := n.start()
		c.send(live(discarded.started(discardedID, 1), released.started(releasedID, 2)))
		for _, f := range []file{discarded, released} {
			if _, err := c.Path(f.hash, 0); err != nil {
				t.Fatalf("Path of %s = %v", f.name, err)
			}
		}

		var errs []error
		pathAtRemove := func(f file) func() {
			return func() {
				_, err := c.Path(f.hash, 0)
				errs = append(errs, err)
			}
		}
		c.engine.onRemove(pathAtRemove(discarded))
		discarding := live(released.started(releasedID, 2))
		discarding.Intent.Discards = []*dlv1.Discard{{Id: discardedID.Bytes(), InfoHash: discarded.hash[:]}}
		c.send(discarding)
		c.engine.onRemove(pathAtRemove(released))
		if err := c.Release(t.Context()); err != nil {
			t.Fatal(err)
		}

		if len(errs) != 2 {
			t.Fatalf("the engine was asked to remove %d torrents, want the discarded one and the released one", len(errs))
		}
		for i, f := range []file{discarded, released} {
			if !errors.Is(errs[i], downloading.ErrNotPresent) {
				t.Errorf("Path of %s as the engine removes it = %v, want ErrNotPresent", f.name, errs[i])
			}
		}
	})
}

// seeded runs a client until it seeds files, then stops it, saving their
// resume data.
func (n *node) seeded(files ...file) []storage.UUID {
	n.t.Helper()
	ids := make([]storage.UUID, len(files))
	ts := make([]*dlv1.Torrent, len(files))
	for i, f := range files {
		n.world.Payload(f.hash, f.payload)
		ids[i] = n.hold(f)
		ts[i] = f.started(ids[i], int64(i+1))
	}
	c := n.start()
	c.send(live(ts...))
	for _, f := range files {
		if _, err := c.Path(f.hash, 0); err != nil {
			n.t.Fatalf("Path of %s = %v", f.name, err)
		}
	}
	c.stop()
	time.Sleep(time.Hour)
	return ids
}

func TestARestartResumesFromResumeDataAndSpreadsItsStarts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		first, second := newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv")
		ids := n.seeded(first, second)
		fetched := len(n.ctl.fetched())

		restart := time.Now()
		c := n.start()
		c.send(live(first.started(ids[0], 1), second.started(ids[1], 2)))
		time.Sleep(timings.StartSpread)
		synctest.Wait()

		if got := len(n.ctl.fetched()) - fetched; got != 0 {
			t.Errorf("the restarted client fetched %d torrents, want none", got)
		}
		if got := c.engine.count(); got != 0 {
			t.Errorf("the restarted client checked %d times, want none", got)
		}
		var starts []time.Time
		for i, f := range []file{first, second} {
			at := startedAnnounces(n.world, f.hash, restart)
			if len(at) != 1 || at[0].After(restart.Add(timings.StartSpread)) {
				t.Fatalf("%s announced started at %v after the restart, want once within %v", f.name, at, timings.StartSpread)
			}
			if r := c.h.reportsOf(ids[i])[0]; r.GetHold() != dlv1.Hold_HOLD_WINDOW || r.GetUntilUnixMs() != at[0].UnixMilli() {
				t.Errorf("%s first reports %v, want held for the window until its start at %v", f.name, r, at[0])
			}
			starts = append(starts, at[0])
		}
		if starts[0].Equal(starts[1]) {
			t.Errorf("both torrents announced started at %v, want their starts spread", starts[0])
		}
	})
}

// TestAStopSavesTheResumeDataTheEngineDropped stops a client
// seeding more torrents than the engine queues events for, so some of the
// resume data it saves is dropped on the way. The client asks for it again
// once the engine says so, and stops at once rather than waiting out
// Timings.ShutdownSave.
func TestAStopSavesTheResumeDataTheEngineDropped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		ts := make([]*dlv1.Torrent, 100)
		for i := range ts {
			f := newFile(t, fmt.Sprintf("%02d.mkv", i))
			n.world.Payload(f.hash, f.payload)
			ts[i] = f.started(n.hold(f), int64(i+1))
		}
		c := n.start()
		c.send(live(ts...))
		if got := c.Seeding(); got != len(ts) {
			t.Fatalf("the client seeds %d torrents, want %d", got, len(ts))
		}

		start := time.Now()
		c.stop()
		if took := time.Since(start); took != 0 {
			t.Errorf("the client took %v to stop, want no wait", took)
		}
	})
}

// TestOutOfStepOrDrainingOnlyAFetchWaits restarts a client that seeds one
// torrent while the Node is out of step or draining: it runs the torrent
// it holds resume data for and fetches nothing new.
func TestOutOfStepOrDrainingOnlyAFetchWaits(t *testing.T) {
	for name, s := range map[string]struct {
		snapshot role.Snapshot[*dlv1.Intent]
		hold     dlv1.Hold
	}{
		"out of step": {role.Snapshot[*dlv1.Intent]{Standing: role.Unreachable}, dlv1.Hold_HOLD_CONTROLLER},
		"draining":    {role.Snapshot[*dlv1.Intent]{Standing: role.Live, Draining: true}, dlv1.Hold_HOLD_DRAINING},
	} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
				kept, added := newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv")
				keptID := n.seeded(kept)[0]
				n.world.Payload(added.hash, added.payload)
				addedID := n.hold(added)
				fetched := len(n.ctl.fetched())

				restart := time.Now()
				c := n.start()
				snap := s.snapshot
				snap.Intent = &dlv1.Intent{Torrents: []*dlv1.Torrent{kept.started(keptID, 1), added.started(addedID, 2)}, Settings: settings}
				c.send(snap)
				time.Sleep(timings.StartSpread)
				synctest.Wait()

				if at := startedAnnounces(n.world, kept.hash, restart); len(at) != 1 {
					t.Errorf("the torrent held with resume data announced started %v %s, want once", at, name)
				}
				if got := len(n.ctl.fetched()) - fetched; got != 0 {
					t.Errorf("the client fetched %d torrents %s, want none", got, name)
				}
				if r := c.h.latest(t, addedID); r.GetHold() != s.hold {
					t.Errorf("the torrent to fetch reports %v %s, want held %v", r, name, s.hold)
				}
			})
		})
	}
}

// TestARebuiltNodeGetsItsTorrentsBackAndKeepsItsTotals starts a Node with
// no local data while its files lie where it reported them: the
// Controller's save folder and baseline bring the torrent back whole,
// with no download, and its counters go on from the baseline.
func TestARebuiltNodeGetsItsTorrentsBackAndKeepsItsTotals(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		folder := filepath.Join(t.TempDir(), "Elsewhere")
		if err := os.MkdirAll(folder, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, f.name), f.payload, 0o600); err != nil {
			t.Fatal(err)
		}
		id := storage.NewUUID()
		baseline := &dlv1.TotalsRow{Id: id.Bytes(), Uploaded: 5000, Downloaded: size, ActiveSeconds: 3600, SeedingSeconds: 1800}
		n.ctl.answer(id, &dlv1.TorrentResponse{Metainfo: f.meta, Baseline: baseline, SaveFolder: folder, ContentType: "Movies"})
		c := n.start()
		c.send(live(f.started(id, 1)))

		if path, err := c.Path(f.hash, 0); err != nil || path != filepath.Join(folder, f.name) {
			t.Fatalf("Path = %q, %v; want the file in the reported save folder", path, err)
		}
		if !slices.Contains(c.Folders(), folder) {
			t.Errorf("Folders = %q, want it to hold the save folder outside the downloads folder", c.Folders())
		}
		if r := c.h.latest(t, id); !r.GetComplete() {
			t.Errorf("the rebuilt torrent reports %v, want complete", r)
		}
		time.Sleep(timings.TotalsReport)
		synctest.Wait()
		want := &dlv1.TotalsRow{Id: id.Bytes(), Uploaded: 5000, Downloaded: size, ActiveSeconds: 3900, SeedingSeconds: 2100}
		if got := c.h.totals(); len(got) != 1 || len(got[0]) != 1 || !proto.Equal(got[0][0], want) {
			t.Errorf("totals %v, want %v: the baseline with the time since", got, want)
		}
	})
}

func TestTotalsGoToTheControllerInBatches(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		seeding, paused := newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv")
		seedingID, pausedID := n.hold(seeding), n.hold(paused)
		n.world.Payload(seeding.hash, seeding.payload)
		n.world.Transfer(seeding.hash, size, 100)
		c := n.start()
		c.send(live(seeding.started(seedingID, 1), paused.paused(pausedID, 2)))

		time.Sleep(timings.TotalsReport)
		synctest.Wait()
		time.Sleep(timings.TotalsReport)
		synctest.Wait()
		batches := c.h.totals()
		if len(batches) != 2 {
			t.Fatalf("sent %d totals batches in two intervals, want 2", len(batches))
		}
		for i, rows := range batches {
			if len(rows) != 1 || !bytes.Equal(rows[0].GetId(), seedingID.Bytes()) {
				t.Fatalf("batch %d holds %v, want only the seeding torrent, whose counters moved", i, rows)
			}
		}
		first, second := batches[0][0], batches[1][0]
		if first.GetUploaded() <= 0 || second.GetUploaded() <= first.GetUploaded() || second.GetSeedingSeconds() <= first.GetSeedingSeconds() {
			t.Errorf("batches %v then %v, want the uploads and seeding time to grow", first, second)
		}
		if second.GetDone() != 0 {
			t.Errorf("a complete torrent's totals carry done %d, want none", second.GetDone())
		}
	})
}

func TestATrackerFailureShowsInTheReport(t *testing.T) {
	for reason, status := range map[string]dlv1.TrackerStatus{
		"Torrent not registered with this tracker": dlv1.TrackerStatus_TRACKER_STATUS_UNREGISTERED,
		"Too many requests":                        dlv1.TrackerStatus_TRACKER_STATUS_ERROR,
	} {
		t.Run(reason, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				n := newNode(t, enginetest.TrackerRules{Failure: reason})
				f := newFile(t, "Orchard.mkv")
				n.world.Payload(f.hash, f.payload)
				id := n.hold(f)
				c := n.start()
				c.send(live(f.started(id, 1)))

				trackers := c.h.latest(t, id).GetTrackers()
				if len(trackers) != 1 || trackers[0].GetStatus() != status || trackers[0].GetMessage() != reason || trackers[0].GetRetryAtUnixMs() == 0 {
					t.Errorf("trackers %v, want one %v with its reason and a retry time", trackers, status)
				}
			})
		})
	}
}

func TestTheFactsNameTheEngineAndWhetherItListens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{})
		c := n.start()
		want := &dlv1.Facts{Libtorrent: "0.0.0", PeerIdPrefix: "-LT0000-", UserAgent: "libtorrent/0.0.0.0", Listen: listen.String(), Bound: listen.String()}
		if got := c.h.lastFacts(); !proto.Equal(got, want) {
			t.Fatalf("facts = %v, want %v", got, want)
		}

		taken := newNode(t, enginetest.TrackerRules{})
		taken.world.TakePort(listen)
		c = taken.start()
		time.Sleep(timings.Facts)
		synctest.Wait()
		want = &dlv1.Facts{Libtorrent: "0.0.0", PeerIdPrefix: "-LT0000-", UserAgent: "libtorrent/0.0.0.0", Listen: listen.String(), ListenError: "address already in use"}
		if got := c.h.lastFacts(); !proto.Equal(got, want) {
			t.Fatalf("facts with the port taken = %v, want %v", got, want)
		}
	})
}
