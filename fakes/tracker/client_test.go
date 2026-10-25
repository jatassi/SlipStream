package tracker_test

import (
	"context"
	"crypto/sha1" //nolint:gosec // v1 piece hashes are SHA-1 by the protocol
	"crypto/tls"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"math/rand/v2"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/ca"
	"github.com/jatassi/slipstream/fakes/internal/front"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

// release is a torrent a Node seeds: its files already lie in folder.
type release struct {
	id     storage.UUID
	meta   []byte
	hash   bencode.InfoHash
	folder string
}

// seededReleases are n small private releases announcing to announce,
// each with its file written.
func seededReleases(t *testing.T, n int, announce string) []release {
	t.Helper()
	out := make([]release, n)
	for i := range out {
		payload := make([]byte, 1000)
		_, _ = rand.NewChaCha8([32]byte{byte(i), byte(i >> 8)}).Read(payload)
		sum := sha1.Sum(payload) //nolint:gosec // v1 piece hashes
		name := fmt.Sprintf("Release.%04d.mkv", i)
		meta, err := bencode.Encode(map[string]any{
			"announce": announce,
			"info":     map[string]any{"length": len(payload), "name": name, "piece length": releases.PieceLength, "pieces": sum[:], "private": 1},
		})
		if err != nil {
			t.Fatal(err)
		}
		m, err := bencode.DecodeMetainfo(meta)
		if err != nil {
			t.Fatal(err)
		}
		folder := filepath.Join(t.TempDir(), name)
		if err := os.MkdirAll(folder, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, name), payload, 0o600); err != nil {
			t.Fatal(err)
		}
		out[i] = release{id: storage.NewUUID(), meta: meta, hash: m.InfoHash, folder: folder}
	}
	return out
}

// serveTracker serves fake-tracker as Scenario s says behind a TLS front
// from a run CA, which libtorrent trusts through SSL_CERT_FILE, for n
// seeded releases. It returns the releases and a reader of the tracker's
// announces.
func serveTracker(t *testing.T, s scenario.Scenario, n int) (rels []release, announces func() []tracker.Announce) {
	t.Helper()
	authority, err := ca.New("client")
	if err != nil {
		t.Fatal(err)
	}
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, authority.PEM(), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSL_CERT_FILE", caFile)
	runDir := t.TempDir()
	log, err := events.Create(runDir, func(events.Record) []events.Record { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	logger := slog.New(slog.DiscardHandler)
	trackerFront := fault.Front{Target: fault.Target(tracker.Fake.Name), Origin: "seedbox", Layers: fault.Conn | fault.TLS | fault.HTTP}
	plan := fault.NewPlan([]fault.Front{trackerFront}, nil)
	set := front.NewSet(plan, func(r events.Record) { _, _ = log.Append(r) }, logger)
	tf, err := set.Listen(t.Context(), trackerFront)
	if err != nil {
		t.Fatal(err)
	}
	passkey := releases.Passkey(s)
	rels = seededReleases(t, n, fake.Address{AddrPort: tf.Addr(), TLS: true}.URL("/announce/"+passkey))
	catalog := &releases.Catalog{Passkey: passkey}
	for i, r := range rels {
		catalog.Releases = append(catalog.Releases, releases.Release{Release: scenario.Release{Name: fmt.Sprintf("release-%d", i)}, InfoHash: r.hash})
	}
	env := fake.NewEnv(s, tracker.Fake, logger, plan, log)
	env.Releases = catalog
	server, err := tracker.Fake.New(s, env)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := authority.Leaf("127.0.0.1", "tracker.fake.localhost")
	if err != nil {
		t.Fatal(err)
	}
	in := front.NewInbox(tf.Addr())
	tf.Serve(front.Served(in, &tls.Config{Certificates: []tls.Certificate{leaf}, MinVersion: tls.VersionTLS12}))
	ctx, cancel := context.WithCancel(context.WithoutCancel(t.Context()))
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, in) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("fake-tracker: %v", err)
		}
		set.Close()
	})
	return rels, func() []tracker.Announce {
		t.Helper()
		recs, err := events.Select(runDir, events.Query{Kind: tracker.Announces.Kind})
		if err != nil {
			t.Fatal(err)
		}
		out := make([]tracker.Announce, len(recs))
		for i, r := range recs {
			if err := json.Unmarshal(r.Data, &out[i]); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
}

// intents hands the client each intent the test sends.
type intents chan role.Snapshot[*dlv1.Intent]

func (in intents) Intent(ctx context.Context) iter.Seq[role.Snapshot[*dlv1.Intent]] {
	return func(yield func(role.Snapshot[*dlv1.Intent]) bool) {
		for {
			select {
			case s := <-in:
				if !yield(s) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

func (intents) SetFacts(*dlv1.Facts) {}

func (intents) Report(context.Context, *storage.Tx[storage.Local], *dlv1.Report) error { return nil }

// controller answers a Node's fetch of each release with its metainfo and
// the folder its files lie in.
type controller map[storage.UUID]release

func (c controller) Torrent(_ context.Context, req *dlv1.TorrentRequest) (*dlv1.TorrentResponse, error) {
	id, err := storage.UUIDFromBytes(req.GetId())
	if err != nil {
		return nil, err
	}
	r, ok := c[id]
	if !ok {
		return nil, rpc.Errorf(rpc.CodeNotFound, "no torrent %s", id)
	}
	return &dlv1.TorrentResponse{Metainfo: r.meta, SaveFolder: r.folder, ContentType: "Movies"}, nil
}

// timings shorten the client's waits so a test of real announces ends in
// seconds; the tracker's min interval is announceFloor.
func timings() downloading.Timings {
	t := downloading.DefaultTimings()
	t.AnnounceFloor, t.AnnounceJitter, t.StartSpread = 20*time.Second, time.Second, 10*time.Second
	return t
}

// seedbox is a Node's built-in client over libtorrent, across restarts.
type seedbox struct {
	t      *testing.T
	local  *storage.DB[storage.Local]
	ctl    controller
	client *downloading.Client
}

func newSeedbox(t *testing.T, rels []release) *seedbox {
	ctl := controller{}
	for _, r := range rels {
		ctl[r.id] = r
	}
	return &seedbox{t: t, local: storagetest.Local(t, downloading.Schema()), ctl: ctl}
}

// start runs a client on a new libtorrent session, which stop shuts down
// as a Node does: its resume data saved, then the session closed.
func (n *seedbox) start() (send func(wanted dlv1.Wanted), stop func()) {
	n.t.Helper()
	tm := timings()
	e, err := libtorrent.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.1:0"), AnnounceFloor: tm.AnnounceFloor, Rand: rand.New(rand.NewPCG(1, 2))}) //nolint:gosec // picks listen ports
	if err != nil {
		n.t.Fatal(err)
	}
	c := downloading.NewClient(downloading.Config{
		Engine: e, Dir: n.t.TempDir(), Timings: tm, RetryMin: time.Second, RetryMax: 10 * time.Second,
		Rand: rand.New(rand.NewPCG(3, 4)), Log: slog.New(slog.DiscardHandler), //nolint:gosec // seeded on purpose so tests replay
	})
	n.client = c
	in := intents(make(chan role.Snapshot[*dlv1.Intent]))
	ctx, cancel := context.WithCancel(context.WithoutCancel(n.t.Context()))
	var wg sync.WaitGroup
	wg.Go(func() { _ = c.Loop(ctx) })
	wg.Go(func() { _ = c.Run(n.local, n.ctl)(ctx, in) })
	stop = sync.OnceFunc(func() {
		_ = c.Shutdown(n.local)(context.WithoutCancel(n.t.Context()))
		cancel()
		_ = e.Close()
		wg.Wait()
	})
	n.t.Cleanup(stop)
	return func(wanted dlv1.Wanted) {
		var ts []*dlv1.Torrent
		for _, r := range n.ctl {
			ts = append(ts, &dlv1.Torrent{Id: r.id.Bytes(), InfoHash: r.hash[:], Wanted: wanted, Order: int64(len(ts) + 1)})
		}
		in <- role.Snapshot[*dlv1.Intent]{Standing: role.Live, Intent: &dlv1.Intent{
			Torrents:   ts,
			Settings:   &dlv1.Settings{ActiveDownloads: 3, ActiveChecks: 50, Connections: 8000, Utp: true, StalledAfterSeconds: 600},
			Thresholds: &dlv1.Thresholds{FaultStallSeconds: 12 * 3600, UnregisteredGapSeconds: 1800},
		}}
	}, stop
}

// until waits up to d for the tracker's announces to satisfy cond.
func until(t *testing.T, d time.Duration, what string, announces func() []tracker.Announce, cond func([]tracker.Announce) bool) []tracker.Announce {
	t.Helper()
	for deadline := time.Now().Add(d); ; time.Sleep(100 * time.Millisecond) {
		got := announces()
		if cond(got) {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out after %v waiting until %s; the tracker heard %s", d, what, summary(got))
		}
	}
}

func count(event string) func([]tracker.Announce) int {
	return func(as []tracker.Announce) int {
		n := 0
		for _, a := range as {
			if a.Event == event {
				n++
			}
		}
		return n
	}
}

func summary(as []tracker.Announce) string {
	var b strings.Builder
	for _, a := range as {
		fmt.Fprintf(&b, "\n  %s %s peer %s accepted %v %s", a.InfoHash[:8], eventName(a.Event), a.PeerID[:16], a.Accepted, a.Failure)
	}
	return b.String()
}

func eventName(e string) string {
	if e == "" {
		return "regular"
	}
	return e
}

func rejections(as []tracker.Announce) []tracker.Announce {
	var out []tracker.Announce
	for _, a := range as {
		if !a.Accepted {
			out = append(out, a)
		}
	}
	return out
}

var unit3dScenario = scenario.Scenario{Seed: "client", Tracker: scenario.Tracker{Interval: time.Minute, MinInterval: 20 * time.Second, Mode: scenario.UNIT3D}}

// TestAQuickPauseAndResumeDrawsNoRejection pauses a seeding torrent and
// resumes it at once: libtorrent would announce started inside UNIT3D's
// window, so the client holds the start until the tracker accepts it.
func TestAQuickPauseAndResumeDrawsNoRejection(t *testing.T) {
	rels, announces := serveTracker(t, unit3dScenario, 1)
	send, _ := newSeedbox(t, rels).start()
	send(dlv1.Wanted_WANTED_STARTED)
	until(t, time.Minute, "the torrent starts", announces, func(as []tracker.Announce) bool { return count("started")(as) == 1 })
	send(dlv1.Wanted_WANTED_PAUSED)
	until(t, time.Minute, "the torrent stops", announces, func(as []tracker.Announce) bool { return count("stopped")(as) == 1 })
	send(dlv1.Wanted_WANTED_STARTED)
	got := until(t, 2*time.Minute, "the torrent starts again", announces, func(as []tracker.Announce) bool { return count("started")(as) == 2 })
	if r := rejections(got); len(r) != 0 {
		t.Fatalf("UNIT3D rejected %s", summary(r))
	}
}

// TestARestartOf1000TorrentsDrawsNoRejection restarts a Node seeding a
// thousand torrents: it announces each started again from its new peer
// ID, which UNIT3D accepts, spread over the start spread.
func TestARestartOf1000TorrentsDrawsNoRejection(t *testing.T) {
	const n = 1000
	rels, announces := serveTracker(t, unit3dScenario, n)
	box := newSeedbox(t, rels)
	send, stop := box.start()
	send(dlv1.Wanted_WANTED_STARTED)
	until(t, 3*time.Minute, "every torrent starts", announces, func(as []tracker.Announce) bool { return count("started")(as) == n })
	stop()
	until(t, time.Minute, "every torrent stops", announces, func(as []tracker.Announce) bool { return count("stopped")(as) == n })

	restart := time.Now()
	send, _ = box.start()
	send(dlv1.Wanted_WANTED_STARTED)
	got := until(t, 3*time.Minute, "every torrent starts again", announces, func(as []tracker.Announce) bool { return count("started")(as) == 2*n })
	if r := rejections(got); len(r) != 0 {
		t.Fatalf("UNIT3D rejected %d announces: %s", len(r), summary(r[:min(len(r), 10)]))
	}
	if took := time.Since(restart); took < timings().StartSpread/2 {
		t.Errorf("the restarted Node announced every started within %v, want them spread over %v", took, timings().StartSpread)
	}
}

// act has the running client act on rels.
func (n *seedbox) act(act dlv1.Act, rels ...release) []*dlv1.ActResult {
	n.t.Helper()
	req := &dlv1.ActRequest{Act: act}
	for _, r := range rels {
		req.Ids = append(req.Ids, r.id.Bytes())
	}
	res, err := n.client.Act(n.t.Context(), req)
	if err != nil {
		n.t.Fatal(err)
	}
	return res.GetResults()
}

// TestARecheckDrawsNoRejection rechecks a seeding torrent: the client
// stops it once UNIT3D allows a stopped, checks it without announcing,
// and starts it again once the tracker's window ends.
func TestARecheckDrawsNoRejection(t *testing.T) {
	rels, announces := serveTracker(t, unit3dScenario, 1)
	box := newSeedbox(t, rels)
	send, _ := box.start()
	send(dlv1.Wanted_WANTED_STARTED)
	until(t, time.Minute, "the torrent starts", announces, func(as []tracker.Announce) bool { return count("started")(as) == 1 })
	if r := box.act(dlv1.Act_ACT_RECHECK, rels[0]); r[0].GetRefused() != dlv1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("recheck = %v, want it done", r[0])
	}
	got := until(t, 2*time.Minute, "the checked torrent starts again", announces, func(as []tracker.Announce) bool {
		return count("stopped")(as) == 1 && count("started")(as) == 2
	})
	if r := rejections(got); len(r) != 0 {
		t.Fatalf("UNIT3D rejected %s", summary(r))
	}
}

// TestAnnounceNowDrawsNoRejection asks a seeding torrent to announce at
// once: the client refuses inside UNIT3D's window, and the announce it
// makes once the window ends is accepted.
func TestAnnounceNowDrawsNoRejection(t *testing.T) {
	rels, announces := serveTracker(t, unit3dScenario, 1)
	box := newSeedbox(t, rels)
	send, _ := box.start()
	send(dlv1.Wanted_WANTED_STARTED)
	until(t, time.Minute, "the torrent starts", announces, func(as []tracker.Announce) bool { return count("started")(as) == 1 })
	r := box.act(dlv1.Act_ACT_ANNOUNCE, rels[0])[0]
	if r.GetRefused() != dlv1.Refusal_REFUSAL_WINDOW {
		t.Fatalf("announcing right after started = %v, want refused for the window", r)
	}
	time.Sleep(time.Until(time.UnixMilli(r.GetAcceptsAtUnixMs() + 1)))
	if r := box.act(dlv1.Act_ACT_ANNOUNCE, rels[0])[0]; r.GetRefused() != dlv1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("announcing once the window ended = %v, want it done", r)
	}
	got := until(t, time.Minute, "the torrent announces", announces, func(as []tracker.Announce) bool { return count("")(as) == 1 })
	if r := rejections(got); len(r) != 0 {
		t.Fatalf("UNIT3D rejected %s", summary(r))
	}
}

// TestGazelleAcceptsTheRealAnnounce has libtorrent announce to a tracker
// that allows only libtorrent's peer ID prefix: SlipStream never sets its
// own.
func TestGazelleAcceptsTheRealAnnounce(t *testing.T) {
	rels, announces := serveTracker(t, scenario.Scenario{Seed: "client", Tracker: scenario.Tracker{Interval: time.Minute, Mode: scenario.Gazelle}}, 1)
	send, _ := newSeedbox(t, rels).start()
	send(dlv1.Wanted_WANTED_STARTED)
	got := until(t, time.Minute, "the torrent starts", announces, func(as []tracker.Announce) bool { return count("started")(as) == 1 })
	if r := rejections(got); len(r) != 0 {
		t.Fatalf("Gazelle rejected %s", summary(r))
	}
}
