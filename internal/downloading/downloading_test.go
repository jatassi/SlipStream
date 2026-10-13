package downloading_test

import (
	"context"
	"errors"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// holder is the Role's holder half as a test drives it: it yields the
// snapshots sent on intents and records facts and reports.
type holder struct {
	intents chan role.Snapshot[*dlv1.Intent]

	mu      sync.Mutex
	facts   []*dlv1.Facts
	reports []*dlv1.Result
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

func (h *holder) Report(_ context.Context, _ *storage.Tx[storage.Local], r *dlv1.Result) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reports = append(h.reports, r)
	return nil
}

func (h *holder) reported() []*dlv1.Result {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*dlv1.Result(nil), h.reports...)
}

func torrent(t *testing.T, name string, payload []byte) (meta []byte, hash bencode.InfoHash) {
	t.Helper()
	meta, err := bencode.Encode(map[string]any{
		"announce": "https://127.0.0.1:1/announce/k",
		"info":     map[string]any{"length": len(payload), "name": name, "piece length": 16384, "pieces": make([]byte, 20), "private": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	return meta, m.InfoHash
}

type client struct {
	*downloading.Client
	dir string
	h   *holder
}

func start(t *testing.T, payloads map[bencode.InfoHash][]byte) client {
	t.Helper()
	e, err := enginetest.Payloads(payloads)(engine.Config{})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	c := downloading.NewClient(downloading.Config{Engine: e, Dir: dir, Timings: downloading.DefaultTimings(), Log: slog.New(slog.DiscardHandler)})
	h := &holder{intents: make(chan role.Snapshot[*dlv1.Intent])}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { _ = c.Drain(ctx) })
	wg.Go(func() {
		if err := c.Run(storagetest.Local(t))(ctx, h); err != nil {
			t.Errorf("run: %v", err)
		}
	})
	t.Cleanup(func() {
		cancel()
		_ = e.Close()
		wg.Wait()
	})
	return client{c, dir, h}
}

func intent(torrents ...*dlv1.Torrent) *dlv1.Intent { return &dlv1.Intent{Torrents: torrents} }

func TestAKeptTorrentDownloadsAndReportsEachGrabOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		payload := []byte("orchard")
		meta, hash := torrent(t, "Orchard.mkv", payload)
		c := start(t, map[bencode.InfoHash][]byte{hash: payload})
		first, second := storage.NewUUID(), storage.NewUUID()
		kept := intent(&dlv1.Torrent{InfoHash: hash[:], Metainfo: meta, GrabId: first.Bytes()})
		c.h.intents <- role.Snapshot[*dlv1.Intent]{Intent: kept, Standing: role.Live}
		synctest.Wait()
		c.h.intents <- role.Snapshot[*dlv1.Intent]{Intent: intent(append(kept.Torrents, &dlv1.Torrent{InfoHash: hash[:], Metainfo: meta, GrabId: second.Bytes()})...), Standing: role.Live}
		synctest.Wait()

		completed := &dlv1.Completed{InfoHash: hash[:], Files: []*dlv1.File{{Index: 0, Path: "Orchard.mkv", Size: 7}}}
		want := []*dlv1.Result{
			{GrabId: first.Bytes(), Outcome: &dlv1.Result_Completed{Completed: completed}},
			{GrabId: second.Bytes(), Outcome: &dlv1.Result_Completed{Completed: completed}},
		}
		got := c.h.reported()
		if len(got) != len(want) || !proto.Equal(got[0], want[0]) || !proto.Equal(got[1], want[1]) {
			t.Fatalf("reports = %v, want %v", got, want)
		}
		path, err := c.Path(hash, 0)
		if want := filepath.Join(c.dir, hash.String(), "Orchard.mkv"); err != nil || path != want {
			t.Fatalf("Path = %q, %v; want %q", path, err, want)
		}
		if b, err := os.ReadFile(path); err != nil || string(b) != "orchard" {
			t.Fatalf("the download holds %q, %v; want orchard", b, err)
		}

		time.Sleep(downloading.DefaultTimings().Facts)
		synctest.Wait()
		c.h.mu.Lock()
		facts := c.h.facts
		c.h.mu.Unlock()
		wantFacts := &dlv1.Facts{Torrents: []*dlv1.TorrentFacts{{InfoHash: hash[:], State: "seeding", Done: 7, Wanted: 7}}}
		if len(facts) != 1 || !proto.Equal(facts[0], wantFacts) {
			t.Fatalf("facts = %v, want one %v", facts, wantFacts)
		}
	})
}

func TestOutOfStepTheClientStartsNothingNew(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		meta, hash := torrent(t, "Orchard.mkv", []byte("orchard"))
		c := start(t, nil)
		c.h.intents <- role.Snapshot[*dlv1.Intent]{Intent: intent(&dlv1.Torrent{InfoHash: hash[:], Metainfo: meta, GrabId: storage.NewUUID().Bytes()}), Standing: role.Unreachable}
		synctest.Wait()
		if _, err := c.Path(hash, 0); !errors.Is(err, downloading.ErrNotPresent) {
			t.Fatalf("Path of a torrent added out of step = %v, want ErrNotPresent", err)
		}
	})
}

func TestATorrentTheEngineRefusesFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := start(t, nil)
		hash := bencode.InfoHash{7}
		grab := storage.NewUUID()
		c.h.intents <- role.Snapshot[*dlv1.Intent]{Intent: intent(&dlv1.Torrent{InfoHash: hash[:], Metainfo: []byte("not bencode"), GrabId: grab.Bytes()}), Standing: role.Live}
		synctest.Wait()
		got := c.h.reported()
		want := &dlv1.Result{GrabId: grab.Bytes(), Outcome: &dlv1.Result_Failed{Failed: &dlv1.Failed{InfoHash: hash[:], Error: "bdecode error 0: bencode: the metainfo is not a dictionary"}}}
		if len(got) != 1 || !proto.Equal(got[0], want) {
			t.Fatalf("reports = %v, want %v", got, want)
		}
	})
}
