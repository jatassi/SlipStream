package downloading_test

import (
	"bytes"
	"context"
	"log/slog"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
	"google.golang.org/protobuf/proto"
)

// keeping is a snapshot of a Node waiting for its Holdings to be adopted:
// keep_all, no Torrent, and the Role drained, as the board's Keeping
// state sends it.
func keeping(discards ...*dlv1.Discard) role.Snapshot[*dlv1.Intent] {
	return role.Snapshot[*dlv1.Intent]{
		Intent:   &dlv1.Intent{KeepAll: true, Discards: discards, Settings: settings, Thresholds: thresholds},
		Standing: role.Live, Draining: true,
	}
}

// unsaved plants a row for f the Node never saved resume data for.
func (n *node) unsaved(f file) {
	n.t.Helper()
	err := n.local.Write(n.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO downloading_torrent (info_hash, save_folder, private) VALUES (?, ?, 0)", f.hash[:], filepath.Join(n.dir, "Movies", f.hash.String()))
		return err
	})
	if err != nil {
		n.t.Fatal(err)
	}
}

// torrentRows is the info hash of every torrent the Node keeps a row for.
func (n *node) torrentRows() []bencode.InfoHash {
	n.t.Helper()
	rows, err := n.local.QueryContext(n.t.Context(), "SELECT info_hash FROM downloading_torrent ORDER BY info_hash")
	if err != nil {
		n.t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []bencode.InfoHash
	for rows.Next() {
		var h []byte
		if err := rows.Scan(&h); err != nil {
			n.t.Fatal(err)
		}
		out = append(out, bencode.InfoHash(h))
	}
	if err := rows.Err(); err != nil {
		n.t.Fatal(err)
	}
	return out
}

// seedTime is the seeding seconds f's saved resume data holds.
func (n *node) seedTime(f file) time.Duration {
	n.t.Helper()
	var resume []byte
	if err := n.local.QueryRowContext(n.t.Context(), "SELECT resume FROM downloading_torrent WHERE info_hash = ?", f.hash[:]).Scan(&resume); err != nil {
		n.t.Fatal(err)
	}
	v, err := bencode.Decode(resume)
	if err != nil {
		n.t.Fatal(err)
	}
	s, _ := v.(map[string]any)["seeding_time"].(int64)
	return time.Duration(s) * time.Second
}

func sortedHashes(files ...file) []bencode.InfoHash {
	out := make([]bencode.InfoHash, len(files))
	for i, f := range files {
		out[i] = f.hash
	}
	slices.SortFunc(out, func(a, b bencode.InfoHash) int { return bytes.Compare(a[:], b[:]) })
	return out
}

func stoppedAnnounces(w *enginetest.World, h bencode.InfoHash, since time.Time) int {
	n := 0
	for _, a := range w.Announces(trackerURL) {
		if a.InfoHash == h && a.Event == "stopped" && !a.At.Before(since) {
			n++
		}
	}
	return n
}

// TestKeepAllAddsEveryResumedRowAndKeepsTheRest restarts a client that
// seeded two torrents, with a row it never saved resume data for, into a
// Keeping intent that lists none of them and discards one: both seeded
// ones come back from resume data with no fetch and seed on, the unsaved
// row stays a row, and the discard changes nothing.
func TestKeepAllAddsEveryResumedRowAndKeepsTheRest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		orchard, meadow, harbor := newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv"), newFile(t, "Harbor.mkv")
		ids := n.seeded(orchard, meadow)
		n.unsaved(harbor)
		fetched := len(n.ctl.fetched())
		seeded := n.seedTime(orchard)

		restart := time.Now()
		c := n.start()
		c.send(keeping(&dlv1.Discard{Id: ids[0].Bytes(), InfoHash: orchard.hash[:], DeleteFiles: true}))
		time.Sleep(time.Hour)
		synctest.Wait()

		for _, f := range []file{orchard, meadow} {
			if at := startedAnnounces(n.world, f.hash, restart); len(at) != 1 {
				t.Errorf("%s announced started %v while kept, want once", f.name, at)
			}
		}
		if got := c.Seeding(); got != 2 {
			t.Errorf("the kept client seeds %d torrents, want 2", got)
		}
		if got := len(n.ctl.fetched()) - fetched; got != 0 {
			t.Errorf("the kept client fetched %d torrents, want none", got)
		}
		if got := c.h.parted(); len(got) != 0 {
			t.Errorf("the kept client confirmed %v, want no discard", got)
		}
		c.stop()
		if got, want := n.torrentRows(), sortedHashes(orchard, meadow, harbor); !slices.Equal(got, want) {
			t.Errorf("the Node keeps rows %v, want %v", got, want)
		}
		if got := n.seedTime(orchard); got < seeded+50*time.Minute {
			t.Errorf("Orchard's seed time went from %v to %v over an hour kept, want it to accrue", seeded, got)
		}
	})
}

// TestKeepAllAcrossARepairKeepsEveryRow has a client seeding two listed
// torrents lose them from its intent as the Node pairs again and keeps
// everything, then restart into the same intent: nothing is removed and
// every row stays.
func TestKeepAllAcrossARepairKeepsEveryRow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		orchard, meadow := newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv")
		for _, f := range []file{orchard, meadow} {
			n.world.Payload(f.hash, f.payload)
		}
		c := n.start()
		c.send(live(orchard.started(n.hold(orchard), 1), meadow.started(n.hold(meadow), 2)))
		for _, f := range []file{orchard, meadow} {
			if _, err := c.Path(f.hash, 0); err != nil {
				t.Fatalf("Path of %s = %v", f.name, err)
			}
		}
		var removes atomic.Int64
		c.engine.onRemove(func() { removes.Add(1) })

		c.send(keeping())
		time.Sleep(timings.ResumeSave)
		synctest.Wait()
		c.stop()
		c = n.start()
		c.engine.onRemove(func() { removes.Add(1) })
		c.send(keeping())
		time.Sleep(timings.StartSpread)
		synctest.Wait()

		if got := removes.Load(); got != 0 {
			t.Errorf("the client removed %d torrents across the re-pair, want none", got)
		}
		if got := c.Seeding(); got != 2 {
			t.Errorf("after the re-pair the client seeds %d torrents, want 2", got)
		}
		if got, want := n.torrentRows(), sortedHashes(orchard, meadow); !slices.Equal(got, want) {
			t.Errorf("the Node keeps rows %v, want %v", got, want)
		}
	})
}

// TestPauseAllPausesEveryBuiltInTorrentAndKeepsIt seeds two torrents, then
// sends pause_all with only one listed: both pause, and both stay.
func TestPauseAllPausesEveryBuiltInTorrentAndKeepsIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		orchard, meadow := newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv")
		for _, f := range []file{orchard, meadow} {
			n.world.Payload(f.hash, f.payload)
		}
		c := n.start()
		id := n.hold(orchard)
		c.send(live(orchard.started(id, 1), meadow.started(n.hold(meadow), 2)))
		for _, f := range []file{orchard, meadow} {
			if _, err := c.Path(f.hash, 0); err != nil {
				t.Fatalf("Path of %s = %v", f.name, err)
			}
		}

		paused := time.Now()
		s := live(orchard.started(id, 1))
		s.Intent.PauseAll = true
		c.send(s)
		time.Sleep(time.Hour)
		synctest.Wait()

		for _, f := range []file{orchard, meadow} {
			if got := stoppedAnnounces(n.world, f.hash, paused); got != 1 {
				t.Errorf("%s announced stopped %d times under pause_all, want once", f.name, got)
			}
		}
		if r := c.h.latest(t, id); r.GetHold() != dlv1.Hold_HOLD_WANTED {
			t.Errorf("the listed torrent reports %v under pause_all, want held as paused", r)
		}
		if got := c.Seeding(); got != 2 {
			t.Errorf("under pause_all the client holds %d finished torrents, want 2", got)
		}
		c.stop()
		if got, want := n.torrentRows(), sortedHashes(orchard, meadow); !slices.Equal(got, want) {
			t.Errorf("the Node keeps rows %v, want %v", got, want)
		}
	})
}

// roleRun is a Node's Downloading Role over its built-in client alone.
type roleRun struct {
	*downloading.Node
	h    *holder
	stop func()
}

func (n *node) startRole() *roleRun {
	n.t.Helper()
	e, err := n.world.Open(engine.Config{ListenInterface: listen})
	if err != nil {
		n.t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	builtin := downloading.NewClient(downloading.Config{
		Engine: e, Dir: n.dir, Listen: listen, Timings: timings, RetryMin: time.Second, RetryMax: time.Minute,
		Rand: rand.New(rand.NewPCG(1, 2)), ProvesPort: downloading.PublicPeer(false), Log: log, //nolint:gosec // seeded on purpose so tests replay
		Volume: n.volume,
	})
	node := downloading.NewNode(builtin, downloading.NodeConfig{Timings: timings, Rand: rand.New(rand.NewPCG(3, 4)), Log: log}) //nolint:gosec // seeded on purpose so tests replay
	r := &roleRun{Node: node, h: &holder{intents: make(chan role.Snapshot[*dlv1.Intent])}}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { _ = builtin.Loop(ctx) })
	wg.Go(func() { _ = node.Run(n.local, n.ctl)(ctx, r.h) })
	r.stop = sync.OnceFunc(func() {
		_ = builtin.Shutdown(n.local)(context.Background())
		cancel()
		_ = e.Close()
		wg.Wait()
	})
	n.t.Cleanup(r.stop)
	synctest.Wait()
	return r
}

func (r *roleRun) send(s role.Snapshot[*dlv1.Intent]) {
	r.h.intents <- s
	synctest.Wait()
}

func asking(round uint64, pageSize uint32) role.Snapshot[*dlv1.Intent] {
	s := keeping()
	s.Intent.Holdings = &dlv1.HoldingsRequest{Round: round, PageSize: pageSize}
	return s
}

func (h *holder) holdings() []*dlv1.Holdings {
	var out []*dlv1.Holdings
	for _, r := range h.reported() {
		if p := r.GetHoldings(); p != nil {
			out = append(out, p)
		}
	}
	return out
}

// TestHoldingsPagesCarryRoundCountAndFolder asks a Node that seeded two
// torrents and holds a row with no resume data for its Holdings in pages
// of two: two pages of round 7 list all three in info hash order with
// their folders, the unsaved one by hash alone. The same round asked again
// sends nothing; the next round goes out whole.
func TestHoldingsPagesCarryRoundCountAndFolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		orchard, meadow, harbor := newFile(t, "Orchard.mkv"), newPrivateFile(t, "Meadow.mkv"), newFile(t, "Harbor.mkv")
		n.seeded(orchard, meadow)
		n.unsaved(harbor)
		r := n.startRole()
		r.send(asking(7, 2))

		pages := r.h.holdings()
		if len(pages) != 2 {
			t.Fatalf("the Node sent %d Holdings pages, want 2", len(pages))
		}
		holding := func(f file, private bool) *dlv1.Holding {
			return &dlv1.Holding{
				InfoHash: f.hash[:], SaveFolder: filepath.Join(n.dir, "Movies", f.hash.String()),
				Content: &dlv1.HoldingContent{Name: f.name, Size: size}, Done: 1, Private: private, Hosts: []string{"tracker.example"},
			}
		}
		want := map[bencode.InfoHash]*dlv1.Holding{
			orchard.hash: holding(orchard, false), meadow.hash: holding(meadow, true),
			harbor.hash: {InfoHash: harbor.hash[:], SaveFolder: filepath.Join(n.dir, "Movies", harbor.hash.String())},
		}
		got := pageEntries(t, pages, 7, want)
		if want := sortedHashes(orchard, meadow, harbor); !slices.Equal(got, want) {
			t.Errorf("the Holdings list %v, want %v in info hash order", got, want)
		}

		r.send(asking(7, 2))
		if got := len(r.h.holdings()); got != 2 {
			t.Errorf("round 7 asked again sent %d pages in all, want still 2", got)
		}
		r.send(asking(8, 1000))
		if pages := r.h.holdings(); len(pages) != 3 || pages[2].GetRound() != 8 || pages[2].GetPages() != 1 || len(pages[2].GetEntries()) != 3 {
			t.Errorf("round 8 sent %v, want one page listing all three", pages[2:])
		}
	})
}

// pageEntries checks every page is page i of round's len(pages) and each
// entry is want's for its hash, and returns the hashes in page order.
func pageEntries(t *testing.T, pages []*dlv1.Holdings, round uint64, want map[bencode.InfoHash]*dlv1.Holding) []bencode.InfoHash {
	t.Helper()
	var got []bencode.InfoHash
	for i, p := range pages {
		if p.GetRound() != round || int(p.GetPage()) != i || int(p.GetPages()) != len(pages) {
			t.Errorf("page %d is round %d page %d of %d, want round %d page %d of %d", i, p.GetRound(), p.GetPage(), p.GetPages(), round, i, len(pages))
		}
		for _, e := range p.GetEntries() {
			h := bencode.InfoHash(e.GetInfoHash())
			got = append(got, h)
			if !proto.Equal(e, want[h]) {
				t.Errorf("the Holdings entry is %v, want %v", e, want[h])
			}
		}
	}
	return got
}

// TestARestartedHolderSendsTheRoundAgain restarts a Node that sent round
// 3 into an intent still asking for it: it sends the round again.
func TestARestartedHolderSendsTheRoundAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		orchard := newFile(t, "Orchard.mkv")
		n.seeded(orchard)
		r := n.startRole()
		r.send(asking(3, 1000))
		r.stop()
		r = n.startRole()
		r.send(asking(3, 1000))

		if pages := r.h.holdings(); len(pages) != 1 || pages[0].GetRound() != 3 || len(pages[0].GetEntries()) != 1 {
			t.Fatalf("the restarted Node sent %v, want round 3 again", pages)
		}
	})
}

// TestAnEmptyRoundIsOnePage asks a Node that holds nothing: one empty page
// tells the Controller the round is in.
func TestAnEmptyRoundIsOnePage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		r := n.startRole()
		r.send(asking(1, 1000))
		if pages := r.h.holdings(); len(pages) != 1 || pages[0].GetPages() != 1 || len(pages[0].GetEntries()) != 0 {
			t.Fatalf("a Node holding nothing sent %v, want one empty page", pages)
		}
	})
}

// TestBuiltInMetainfoRebuildsFromResume asks a Node for the metainfo of a
// torrent it seeded: it hashes to the torrent's info hash and names its
// tracker. A torrent with no resume data is NotFound.
func TestBuiltInMetainfoRebuildsFromResume(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		orchard, harbor := newFile(t, "Orchard.mkv"), newFile(t, "Harbor.mkv")
		n.seeded(orchard)
		n.unsaved(harbor)
		r := n.startRole()
		r.send(keeping())

		res, err := r.Metainfo(t.Context(), &dlv1.MetainfoRequest{InfoHash: orchard.hash[:]})
		if err != nil {
			t.Fatal(err)
		}
		m, err := bencode.DecodeMetainfo(res.GetMetainfo())
		if err != nil {
			t.Fatal(err)
		}
		if m.InfoHash != orchard.hash || !slices.Equal(m.Trackers(), []string{trackerURL}) {
			t.Errorf("the rebuilt metainfo hashes to %s with trackers %v, want %s with %s", m.InfoHash, m.Trackers(), orchard.hash, trackerURL)
		}
		if _, err := r.Metainfo(t.Context(), &dlv1.MetainfoRequest{InfoHash: harbor.hash[:]}); err == nil {
			t.Error("the metainfo of a torrent with no resume data answered, want NotFound")
		}
	})
}
