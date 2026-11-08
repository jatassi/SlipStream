package tracker_test

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // v1 piece hashes are SHA-1 by the protocol
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
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
	"github.com/jatassi/slipstream/fakes/internal/seeders"
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

// release is a torrent a Node seeds: its files already lie in folder. meta
// announces as the Node's site account, and seed, the same torrent, as
// fixture-seeders.
type release struct {
	id     storage.UUID
	meta   []byte
	seed   []byte
	hash   bencode.InfoHash
	folder string
}

// nodeAccount is the site account the Node announces as.
const nodeAccount = "torznab"

// seededReleases are n small releases announcing to the announce URL of
// each account, each with its file written, private unless public.
func seededReleases(t *testing.T, n int, announce func(account string) string, public bool) []release {
	t.Helper()
	out := make([]release, n)
	for i := range out {
		payload := make([]byte, 1000)
		_, _ = rand.NewChaCha8([32]byte{byte(i), byte(i >> 8)}).Read(payload)
		sum := sha1.Sum(payload) //nolint:gosec // v1 piece hashes
		name := fmt.Sprintf("Release.%04d.mkv", i)
		info := map[string]any{"length": len(payload), "name": name, "piece length": releases.PieceLength, "pieces": sum[:]}
		if !public {
			info["private"] = 1
		}
		meta, err := bencode.Encode(map[string]any{"announce": announce(nodeAccount), "info": info})
		if err != nil {
			t.Fatal(err)
		}
		seed, err := bencode.Encode(map[string]any{"announce": announce(releases.Seeders), "info": info})
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
		out[i] = release{id: storage.NewUUID(), meta: meta, seed: seed, hash: m.InfoHash, folder: folder}
	}
	return out
}

// trackerRun is fake-tracker serving a test's releases, recording into
// runDir, behind fronts whose faults plan plays.
type trackerRun struct {
	t      *testing.T
	s      scenario.Scenario
	rels   []release
	runDir string
	plan   *fault.Plan
}

// trackerFront is the front the seedbox origin reaches fake-tracker
// through.
var trackerFront = fault.Front{Target: fault.Target(tracker.Fake.Name), Origin: "seedbox", Layers: fault.Conn | fault.TLS | fault.HTTP}

// serveTracker serves fake-tracker for n seeded releases and returns them
// and a reader of the tracker's announces.
func serveTracker(t *testing.T, s scenario.Scenario, n int) (rels []release, announces func() []tracker.Announce) {
	t.Helper()
	w := startTracker(t, s, n, trackerOpts{})
	return w.rels, w.announces
}

// trackerOpts are how a test's fake-tracker runs.
type trackerOpts struct {
	// swarm has fixture-seeders seed each release, and the tracker hand a
	// leecher a seeder front that reaches them.
	swarm bool
	// listed is the seeders the feeds list of each release.
	listed int
	// udp makes the Node's releases public, announcing to fake-tracker's
	// BEP 15 socket, while fixture-seeders still announce over HTTPS.
	udp bool
}

// startTracker serves fake-tracker as Scenario s says behind a TLS front
// from a run CA, which libtorrent trusts through SSL_CERT_FILE, for n
// releases, and with opts.udp its BEP 15 announce on a loopback socket.
func startTracker(t *testing.T, s scenario.Scenario, n int, opts trackerOpts) *trackerRun {
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
	w := &trackerRun{t: t, s: s, runDir: t.TempDir()}
	log, err := events.Create(w.runDir, func(events.Record) []events.Record { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	logger := slog.New(slog.DiscardHandler)
	name := func(i int) string { return fmt.Sprintf("release-%d", i) }
	fronts := []fault.Front{trackerFront}
	if opts.swarm {
		for i := range n {
			fronts = append(fronts, fault.Front{Target: tracker.SeedersTarget(name(i)), Origin: trackerFront.Origin, Layers: fault.Conn})
		}
	}
	w.plan = fault.NewPlan(fronts, nil)
	set := front.NewSet(w.plan, func(r events.Record) { _, _ = log.Append(r) }, logger)
	tf, err := set.Listen(t.Context(), trackerFront)
	if err != nil {
		t.Fatal(err)
	}
	at := fake.Address{AddrPort: tf.Addr(), TLS: true}
	socket, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:0")))
	if err != nil {
		t.Fatal(err)
	}
	announce := func(account string) string {
		if opts.udp && account == nodeAccount {
			return "udp://" + socket.LocalAddr().String() + "/announce"
		}
		return at.URL("/announce/" + releases.Passkey(s, account))
	}
	w.rels = seededReleases(t, n, announce, opts.udp)
	env := fake.NewEnv(s, tracker.Fake, logger, w.plan, log)
	env.Releases = &releases.Catalog{}
	for i, r := range w.rels {
		rel := scenario.Release{Name: name(i), Seeded: opts.swarm, Seeders: opts.listed, Public: opts.udp}
		env.Releases.Releases = append(env.Releases.Releases, releases.Release{Release: rel, Dir: r.folder, Metainfo: r.seed, InfoHash: r.hash})
	}
	for i, f := range fronts[1:] {
		sf, err := set.Listen(t.Context(), f)
		if err != nil {
			t.Fatal(err)
		}
		sf.Serve(front.Forward(w.seeder(w.rels[i])))
		env.Fronts = append(env.Fronts, fake.Front{Target: f.Target, Node: f.Origin, Address: fake.Address{AddrPort: sf.Addr()}})
	}
	leaf, err := authority.Leaf("127.0.0.1", "tracker.fake.localhost")
	if err != nil {
		t.Fatal(err)
	}
	in := front.NewInbox(tf.Addr())
	tf.Serve(front.Served(in, &tls.Config{Certificates: []tls.Certificate{leaf}, MinVersion: tls.VersionTLS12}))
	t.Cleanup(set.Close)
	serveBoth(t, s, env, log, in, socket)
	if opts.swarm {
		seeding, err := seeders.Start(t.Context(), env.Releases, caFile)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = seeding.Close() })
	}
	return w
}

// serveBoth serves fake-tracker from env on in, and its BEP 15 announce
// on socket as the seedbox origin's, recording into sink, until the test
// ends.
func serveBoth(t *testing.T, s scenario.Scenario, env fake.Env, sink fake.Sink, in net.Listener, socket *net.UDPConn) {
	t.Helper()
	server, err := tracker.Fake.New(s, env)
	if err != nil {
		t.Fatal(err)
	}
	udpEnv := fake.NewEnv(s, tracker.UDP, env.Log, env.Faults, sink)
	udpEnv.Releases, udpEnv.Fronts, udpEnv.Shared = env.Releases, env.Fronts, env.Shared
	udp, err := tracker.UDP.NewUDP(s, udpEnv)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.WithoutCancel(t.Context()))
	done, udpDone := make(chan error, 1), make(chan error, 1)
	go func() { done <- server.Serve(ctx, in) }()
	go func() { udpDone <- udp.ServeUDP(ctx, map[string]*net.UDPConn{trackerFront.Origin: socket}) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("fake-tracker: %v", err)
		}
		if err := <-udpDone; err != nil {
			t.Errorf("fake-tracker's UDP announce: %v", err)
		}
	})
}

// seeder finds where fixture-seeders seed r, from their announce.
func (w *trackerRun) seeder(r release) func() (netip.AddrPort, error) {
	passkey := releases.Passkey(w.s, releases.Seeders)
	return func() (netip.AddrPort, error) {
		as, err := captured[tracker.Announce](w.runDir, tracker.Announces.Kind)
		if err != nil {
			return netip.AddrPort{}, err
		}
		for _, a := range as {
			if a.Passkey == passkey && a.InfoHash == r.hash.String() && a.Left == 0 {
				return netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), a.Port), nil
			}
		}
		return netip.AddrPort{}, errors.New("no seeder announced")
	}
}

func (w *trackerRun) announces() []tracker.Announce {
	w.t.Helper()
	as, err := captured[tracker.Announce](w.runDir, tracker.Announces.Kind)
	if err != nil {
		w.t.Fatal(err)
	}
	return as
}

// history is the tracker's latest record of account's history of r, zero
// before the account announced it.
func (w *trackerRun) history(account string, r release) tracker.History {
	w.t.Helper()
	hs, err := captured[tracker.History](w.runDir, tracker.Seeding.Kind)
	if err != nil {
		w.t.Fatal(err)
	}
	var latest tracker.History
	for _, h := range hs {
		if h.Account == account && h.InfoHash == r.hash.String() {
			latest = h
		}
	}
	return latest
}

// seeded waits up to d until the tracker's history of r for the Node's
// account satisfies cond, failing the test at any look that finds box's
// count of r ahead of it.
func (w *trackerRun) seeded(box *seedbox, r release, d time.Duration, what string, cond func(tracker.History) bool) tracker.History {
	w.t.Helper()
	for deadline := time.Now().Add(d); ; time.Sleep(100 * time.Millisecond) {
		counted := box.reports.announced(r)
		h := w.history(nodeAccount, r)
		if n := len(counted); n > 0 && counted[n-1] > h.SeedSeconds {
			w.t.Fatalf("the Node counts %d s announced while the tracker credited %d s", counted[n-1], h.SeedSeconds)
		}
		if cond(h) {
			return h
		}
		if time.Now().After(deadline) {
			w.t.Fatalf("timed out after %v waiting until %s; the tracker keeps %+v and the Node counted %v", d, what, h, counted)
		}
	}
}

func captured[T any](runDir, kind string) ([]T, error) {
	recs, err := events.Select(runDir, events.Query{Kind: kind})
	if err != nil {
		return nil, err
	}
	out := make([]T, len(recs))
	for i, r := range recs {
		if err := json.Unmarshal(r.Data, &out[i]); err != nil {
			return nil, fmt.Errorf("read a %s record: %w", kind, err)
		}
	}
	return out, nil
}

// intents hands the client each intent the test sends, and keeps each
// report it makes.
type intents struct {
	ch      chan role.Snapshot[*dlv1.Intent]
	reports *reports
}

func (in intents) Intent(ctx context.Context) iter.Seq[role.Snapshot[*dlv1.Intent]] {
	return func(yield func(role.Snapshot[*dlv1.Intent]) bool) {
		for {
			select {
			case s := <-in.ch:
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

func (in intents) Report(_ context.Context, _ *storage.Tx[storage.Local], r *dlv1.Report) error {
	in.reports.mu.Lock()
	defer in.reports.mu.Unlock()
	in.reports.all = append(in.reports.all, r)
	return nil
}

// reports is every report a seedbox's clients made, oldest first.
type reports struct {
	mu  sync.Mutex
	all []*dlv1.Report
}

// announced is the announced seconds of each totals row reported of r,
// oldest first.
func (rs *reports) announced(r release) []int64 {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	var out []int64
	for _, rep := range rs.all {
		rows := rep.GetTotals().GetRows()
		for _, tr := range rep.GetTorrents().GetReports() {
			rows = append(rows, tr.GetTotals())
		}
		for _, row := range rows {
			if bytes.Equal(row.GetId(), r.id.Bytes()) {
				out = append(out, row.GetAnnouncedSeconds())
			}
		}
	}
	return out
}

// failedAnnouncing reports whether a report of r showed a tracker failing.
func (rs *reports) failedAnnouncing(r release) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	for _, rep := range rs.all {
		for _, tr := range rep.GetTorrents().GetReports() {
			if bytes.Equal(tr.GetId(), r.id.Bytes()) && slices.ContainsFunc(tr.GetTrackers(), func(l *dlv1.TrackerLine) bool { return l.GetStatus() == dlv1.TrackerStatus_TRACKER_STATUS_ERROR }) {
				return true
			}
		}
	}
	return false
}

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
	t       *testing.T
	local   *storage.DB[storage.Local]
	ctl     controller
	timings downloading.Timings
	client  *downloading.Client
	reports reports
}

func newSeedbox(t *testing.T, rels []release) *seedbox {
	ctl := controller{}
	for _, r := range rels {
		ctl[r.id] = r
	}
	return &seedbox{t: t, local: storagetest.Local(t, downloading.Schema()), ctl: ctl, timings: timings()}
}

// start runs a client on a new libtorrent session, which stop shuts down
// as a Node does: its resume data saved, then the session closed.
func (n *seedbox) start() (send func(wanted dlv1.Wanted), stop func()) {
	n.t.Helper()
	tm := n.timings
	e, err := libtorrent.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.1:0"), AnnounceFloor: tm.AnnounceFloor, Rand: rand.New(rand.NewPCG(1, 2))}) //nolint:gosec // picks listen ports
	if err != nil {
		n.t.Fatal(err)
	}
	c := downloading.NewClient(downloading.Config{
		Engine: e, Dir: n.t.TempDir(), Timings: tm, RetryMin: time.Second, RetryMax: 10 * time.Second,
		Rand: rand.New(rand.NewPCG(3, 4)), Log: slog.New(slog.DiscardHandler), //nolint:gosec // seeded on purpose so tests replay
	})
	n.client = c
	in := intents{ch: make(chan role.Snapshot[*dlv1.Intent]), reports: &n.reports}
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
		in.ch <- role.Snapshot[*dlv1.Intent]{Standing: role.Live, Intent: &dlv1.Intent{
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

// untilEvery waits until the tracker has heard want announces of event,
// failing only once a minute passes without a new one: the client works
// through a thousand torrents at a pace set by the CPU it gets, which
// -race on a shared runner cuts past any fixed bound.
func untilEvery(t *testing.T, event string, want int, announces func() []tracker.Announce) []tracker.Announce {
	t.Helper()
	heard, at := -1, time.Now()
	for ; ; time.Sleep(time.Second) {
		got := announces()
		n := count(event)(got)
		if n == want {
			return got
		}
		if n != heard {
			heard, at = n, time.Now()
		}
		if time.Since(at) > time.Minute {
			t.Fatalf("the tracker heard %d of %d %s announces and no new one for a minute:%s", n, want, eventName(event), summary(got))
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
	untilEvery(t, "started", n, announces)
	stop()
	untilEvery(t, "stopped", n, announces)

	restart := time.Now()
	send, _ = box.start()
	send(dlv1.Wanted_WANTED_STARTED)
	got := untilEvery(t, "started", 2*n, announces)
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

// TestAnnouncedSecondsNeverAheadOfTheTracker downloads a private torrent
// from fixture-seeders, seeds it, pauses and resumes it, and seeds through
// a tracker outage. At every look the Node's count of the seed time the
// tracker credited is no more than the seed time fake-tracker keeps for the
// Node's account, and it ends within a second or two per credited announce
// of it: what a lost interval would cost it is far more.
func TestAnnouncedSecondsNeverAheadOfTheTracker(t *testing.T) {
	const interval = 20 * time.Second
	s := scenario.Scenario{Seed: "client", Tracker: scenario.Tracker{Interval: interval, MinInterval: interval, Mode: scenario.UNIT3D}}
	w := startTracker(t, s, 1, trackerOpts{swarm: true})
	leech := w.rels[0]
	leech.folder = t.TempDir()
	box := newSeedbox(t, []release{leech})
	box.timings.TotalsReport = time.Second
	send, _ := box.start()
	seeded := func(d time.Duration, what string, cond func(tracker.History) bool) tracker.History {
		t.Helper()
		return w.seeded(box, leech, d, what, cond)
	}

	send(dlv1.Wanted_WANTED_STARTED)
	seeded(2*time.Minute, "the torrent completes and seeds two intervals", func(h tracker.History) bool { return h.Seeder && h.SeedSeconds >= int64(interval/time.Second)*3/2 })
	send(dlv1.Wanted_WANTED_PAUSED)
	seeded(time.Minute, "the paused torrent's stopped reaches the tracker", func(h tracker.History) bool { return !h.Active })
	send(dlv1.Wanted_WANTED_STARTED)
	resumed := seeded(time.Minute, "the resumed torrent's started reaches the tracker", func(h tracker.History) bool { return h.Active })
	beforeOutage := seeded(time.Minute, "the resumed torrent seeds an interval", func(h tracker.History) bool { return h.SeedSeconds > resumed.SeedSeconds })

	refuse, err := fault.Parse("refuse")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.plan.Set(fault.Rule{ID: "tracker-down", Target: trackerFront.Target, Origin: trackerFront.Origin, Fault: refuse}); err != nil {
		t.Fatal(err)
	}
	seeded(time.Minute, "an announce fails in the outage", func(tracker.History) bool { return box.reports.failedAnnouncing(leech) })
	w.plan.Clear("tracker-down")
	seeded(2*time.Minute, "the first announce after the outage credits the time across it", func(h tracker.History) bool {
		return h.SeedSeconds-beforeOutage.SeedSeconds > int64(interval/time.Second)
	})

	credited := int64(0)
	passkey := releases.Passkey(s, nodeAccount)
	for _, a := range w.announces() {
		if a.Passkey == passkey && a.Accepted && a.Left == 0 && a.Event != "started" {
			credited++
		}
	}
	h := seeded(10*time.Second, "the Node counts what the tracker credited", func(h tracker.History) bool {
		counted := box.reports.announced(leech)
		return len(counted) > 0 && counted[len(counted)-1] >= h.SeedSeconds-2*credited
	})
	counted := box.reports.announced(leech)
	if !slices.IsSorted(counted) {
		t.Errorf("the Node reported announced seconds %v, want them never to fall", counted)
	}
	t.Logf("the tracker credited %d s over %d announces and the Node counted %d s", h.SeedSeconds, credited, counted[len(counted)-1])
}

// udpAnnounces are the BEP 15 announces fake-tracker heard.
func (w *trackerRun) udpAnnounces() []tracker.Announce {
	w.t.Helper()
	as, err := captured[tracker.Announce](w.runDir, tracker.UDPAnnounces.Kind)
	if err != nil {
		w.t.Fatal(err)
	}
	return as
}

// swarmOf is the counts the Node's Inspect gives r's one tracker line,
// nil while it gives none.
func (n *seedbox) swarmOf(r release) *engine.Swarm {
	n.t.Helper()
	res, err := n.client.Inspect(n.t.Context(), &dlv1.InspectRequest{Id: r.id.Bytes()})
	if err != nil {
		n.t.Fatal(err)
	}
	lines := res.GetTrackers()
	if len(lines) != 1 || lines[0].Seeders == nil {
		return nil
	}
	return &engine.Swarm{Seeders: int(lines[0].GetSeeders()), Leechers: int(lines[0].GetLeechers())}
}

// untilSwarm waits up to two minutes until the Node's Inspect gives r's
// tracker line the counts want, and returns each count it gave on the way.
func (n *seedbox) untilSwarm(r release, want engine.Swarm) []engine.Swarm {
	n.t.Helper()
	var seen []engine.Swarm
	for deadline := time.Now().Add(2 * time.Minute); len(seen) == 0 || seen[len(seen)-1] != want; time.Sleep(100 * time.Millisecond) {
		if sw := n.swarmOf(r); sw != nil && (len(seen) == 0 || seen[len(seen)-1] != *sw) {
			seen = append(seen, *sw)
		}
		if time.Now().After(deadline) {
			n.t.Fatalf("timed out waiting until the Inspector reads %+v; it read %+v", want, seen)
		}
	}
	return seen
}

// TestSwarmCountsReachTheInspector downloads a release from fixture-seeders
// that the feeds list 141 more seeders of, over HTTPS as a private
// release and over BEP 15 as a public one. fake-tracker answers each
// announce with the seeders it lists and the peers it saw, and the Node's
// Inspect reads those counts: 142 seeders and the Node leeching, then 143
// and none once it seeds.
func TestSwarmCountsReachTheInspector(t *testing.T) {
	const listed = 141
	s := scenario.Scenario{Seed: "client", Tracker: scenario.Tracker{Interval: time.Minute}}
	for _, tc := range []struct {
		name string
		udp  bool
	}{{"HTTPS", false}, {"UDP", true}} {
		t.Run(tc.name, func(t *testing.T) {
			w := startTracker(t, s, 1, trackerOpts{swarm: true, listed: listed, udp: tc.udp})
			leech := w.rels[0]
			leech.folder = t.TempDir()
			box := newSeedbox(t, []release{leech})
			send, _ := box.start()
			send(dlv1.Wanted_WANTED_STARTED)
			leeching, seeding := engine.Swarm{Seeders: listed + 1, Leechers: 1}, engine.Swarm{Seeders: listed + 2}
			seen := box.untilSwarm(leech, seeding)
			for _, sw := range seen {
				if sw != leeching && sw != seeding {
					t.Errorf("the Inspector read %+v, which fake-tracker never answered; it read %+v", sw, seen)
				}
			}
			heard := w.udpAnnounces()
			if tc.udp != (len(heard) > 0) {
				t.Errorf("over %s the Node sent %d BEP 15 announces", tc.name, len(heard))
			}
		})
	}
}
