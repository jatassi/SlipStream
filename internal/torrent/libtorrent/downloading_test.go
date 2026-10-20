package libtorrent_test

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// added is an engine that remembers the handle of each torrent it adds,
// by info hash, and each it resumes, so the test can point it at a seeder
// once it runs.
type added struct {
	engine.Engine
	mu      sync.Mutex
	handles map[bencode.InfoHash]engine.Handle
	resumed map[engine.Handle]bool
}

func (a *added) Add(p engine.AddParams) (engine.Handle, error) {
	h, err := a.Engine.Add(p)
	if m, merr := bencode.DecodeMetainfo(p.Metainfo); err == nil && merr == nil {
		a.mu.Lock()
		a.handles[m.InfoHash] = h
		a.mu.Unlock()
	}
	return h, err
}

func (a *added) Resume(h engine.Handle) error {
	a.mu.Lock()
	a.resumed[h] = true
	a.mu.Unlock()
	return a.Engine.Resume(h)
}

func (a *added) running(h bencode.InfoHash) (engine.Handle, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	handle, ok := a.handles[h]
	return handle, ok && a.resumed[handle]
}

// intents is Downloading's holder half as the test drives it.
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

// controller answers each Torrent fetch from the metainfo it holds by id.
type controller map[storage.UUID][]byte

func (c controller) Torrent(_ context.Context, req *dlv1.TorrentRequest) (*dlv1.TorrentResponse, error) {
	id, err := storage.UUIDFromBytes(req.GetId())
	if err != nil {
		return nil, err
	}
	return &dlv1.TorrentResponse{Metainfo: c[id], ContentType: "Movies"}, nil
}

type seeded struct {
	name    string
	payload []byte
	meta    []byte
	hash    bencode.InfoHash
}

// seed has e seed a payload named name from dir. The engine must close
// before dir is removed, as Windows removes no file held open, so dir is
// made before e opens.
func seed(t *testing.T, e engine.Engine, dir, name string) seeded {
	t.Helper()
	payload := make([]byte, 1<<20)
	_, _ = rand.NewChaCha8([32]byte{name[0]}).Read(payload)
	meta := singleFileTorrent(t, name, payload, 64<<10)
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	started(t, e, engine.AddParams{Metainfo: meta, SavePath: dir, SeedMode: true})
	return seeded{name: name, payload: payload, meta: meta, hash: m.InfoHash}
}

func until(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting until %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestDownloadingLetsTorrentsGoAndKeepsTheirFiles downloads two torrents
// through Downloading's client on libtorrent, then discards one, keeping
// its files, and releases the Role: each torrent leaves the engine, and
// each file it downloaded stays on disk whole, under the Content Type's
// folder.
func TestDownloadingLetsTorrentsGoAndKeepsTheirFiles(t *testing.T) {
	seedDir, dir := t.TempDir(), t.TempDir()
	seeder := open(t)
	first, second := seed(t, seeder, seedDir, "First.mkv"), seed(t, seeder, seedDir, "Second.mkv")
	seederAddr, err := seeder.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}

	e := &added{Engine: open(t), handles: map[bencode.InfoHash]engine.Handle{}, resumed: map[engine.Handle]bool{}}
	c := downloading.NewClient(downloading.Config{
		Engine: e, Dir: dir, Timings: downloading.DefaultTimings(), RetryMin: time.Second, RetryMax: time.Second,
		Rand: rand.New(rand.NewPCG(1, 2)), ProvesPort: downloading.PublicPeer(false), Log: slog.New(slog.DiscardHandler), //nolint:gosec // seeded on purpose so tests replay
	})
	ids := map[bencode.InfoHash]storage.UUID{first.hash: storage.NewUUID(), second.hash: storage.NewUUID()}
	ctl := controller{ids[first.hash]: first.meta, ids[second.hash]: second.meta}
	in := make(intents)
	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	wg.Go(func() { _ = c.Loop(ctx) })
	wg.Go(func() {
		if err := c.Run(storagetest.Local(t, downloading.Schema()), ctl)(ctx, in); err != nil {
			t.Errorf("run: %v", err)
		}
	})
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	settings := &dlv1.Settings{ActiveDownloads: 3, ActiveChecks: 1, StalledAfterSeconds: 600}
	torrent := func(s seeded, order int64) *dlv1.Torrent {
		return &dlv1.Torrent{Id: ids[s.hash].Bytes(), InfoHash: s.hash[:], Wanted: dlv1.Wanted_WANTED_STARTED, Order: order}
	}
	in <- role.Snapshot[*dlv1.Intent]{Intent: &dlv1.Intent{Torrents: []*dlv1.Torrent{torrent(first, 1), torrent(second, 2)}, Settings: settings}, Standing: role.Live}
	for _, s := range []seeded{first, second} {
		until(t, "the client starts "+s.name, func() bool { _, ok := e.running(s.hash); return ok })
		h, _ := e.running(s.hash)
		if err := e.ConnectPeer(h, seederAddr); err != nil {
			t.Fatalf("connect %s to the seeder: %v", s.name, err)
		}
	}
	for _, s := range []seeded{first, second} {
		until(t, "the client finishes "+s.name, func() bool { _, err := c.Path(s.hash, 0); return err == nil })
	}

	gone := func(s seeded) {
		t.Helper()
		h, _ := e.running(s.hash)
		until(t, "the engine no longer holds "+s.name, func() bool {
			_, err := e.Files(h)
			return errors.Is(err, engine.ErrUnknownTorrent)
		})
		if _, err := c.Path(s.hash, 0); !errors.Is(err, downloading.ErrNotPresent) {
			t.Errorf("Path of %s once let go = %v, want ErrNotPresent", s.name, err)
		}
		got, err := os.ReadFile(filepath.Join(dir, "Movies", s.hash.String(), s.name))
		if err != nil || !bytes.Equal(got, s.payload) {
			t.Errorf("%s once let go holds %d bytes, %v; want the %d bytes it downloaded", s.name, len(got), err, len(s.payload))
		}
	}

	in <- role.Snapshot[*dlv1.Intent]{Intent: &dlv1.Intent{
		Torrents: []*dlv1.Torrent{torrent(second, 2)},
		Discards: []*dlv1.Discard{{Id: ids[first.hash].Bytes(), InfoHash: first.hash[:]}},
		Settings: settings,
	}, Standing: role.Live}
	gone(first)
	h, _ := e.running(second.hash)
	if _, err := e.Files(h); err != nil {
		t.Fatalf("the engine dropped %s, which the intent still lists: %v", second.name, err)
	}

	if err := c.Release(t.Context()); err != nil {
		t.Fatalf("release: %v", err)
	}
	gone(second)
}
