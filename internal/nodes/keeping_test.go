package nodes_test

import (
	"context"
	"math/rand/v2"
	"net/netip"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// TestAPendingNodeWithNoRoleReadsKeepingAndKeepsDownloadingRunning has
// box hold Downloading over its built-in client, which keeps a row for one
// torrent, then wait to be adopted as the Admin takes Downloading from it.
// Box reads Keeping, never Draining, its supervisor keeps the holder
// running, and the row stays.
func TestAPendingNodeWithNoRoleReadsKeepingAndKeepsDownloadingRunning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var pending atomic.Pointer[nodes.ID]
		c := startController(t, func(cfg *nodes.Config) {
			cfg.Served = append(cfg.Served, downloading.Def.Serve(keepAll, appliesNothing))
			cfg.Pending = func(context.Context, storage.Reader[storage.Library]) (map[nodes.ID]bool, error) {
				if id := pending.Load(); id != nil {
					return map[nodes.ID]bool{*id: true}, nil
				}
				return nil, nil
			}
		})
		w := c.worker(t, "box", "127.0.0.3")
		_, box := c.pair(t, w, c.mint(t), "box")
		local := holdDownloading(t, w, c)
		c.apply(t, nodes.Roles{box: role.Of(role.Downloading)})
		synctest.Wait()
		if got := running(c, box); !slices.Contains(got, string(role.Downloading)) {
			t.Fatalf("box runs %v holding Downloading, want downloading", got)
		}

		pending.Store(&box)
		c.apply(t, nodes.Roles{box: 0})
		time.Sleep(10 * time.Minute)
		synctest.Wait()
		if st := c.svc.Who(role.Downloading).State(box); st != nodes.Keeping || c.svc.Draining(role.Downloading, box) {
			t.Errorf("box's Downloading is %q, draining %v; want keeping, not draining", st, c.svc.Draining(role.Downloading, box))
		}
		if got := running(c, box); !slices.Contains(got, string(role.Downloading)) {
			t.Errorf("box runs %v while it keeps, want downloading", got)
		}
		var rows int
		if err := local.QueryRowContext(t.Context(), "SELECT count(*) FROM downloading_torrent").Scan(&rows); err != nil {
			t.Fatal(err)
		}
		if rows != 1 {
			t.Errorf("box keeps %d downloading_torrent rows, want its one", rows)
		}
	})
}

func keepAll(context.Context, storage.Reader[storage.Library], role.NodeID) (*dlv1.Intent, error) {
	return &dlv1.Intent{KeepAll: true}, nil
}

func appliesNothing(context.Context, *storage.Tx[storage.Library], role.NodeID, *dlv1.Report) error {
	return nil
}

func running(c *controller, node nodes.ID) []string {
	f, _ := c.svc.Controls().NodeFacts(node)
	return f.GetRunning()
}

// holdDownloading runs w's holder half holding Downloading over a built-in
// client on an in-memory engine, its local.db keeping a row for one
// torrent it never saved resume data for, until the test ends.
func holdDownloading(t *testing.T, w *linktest.Node, c *controller) *storage.DB[storage.Local] {
	t.Helper()
	local := storagetest.Local(t, role.LocalSchema(), downloading.Schema())
	dir := t.TempDir()
	err := local.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO downloading_torrent (info_hash, save_folder, private) VALUES (?, ?, 0)", make([]byte, 20), dir)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	listen := netip.MustParseAddrPort("0.0.0.0:6881")
	e, err := enginetest.NewWorld().Open(engine.Config{ListenInterface: listen})
	if err != nil {
		t.Fatal(err)
	}
	timings := downloading.DefaultTimings()
	builtin := downloading.NewClient(downloading.Config{
		Engine: e, Dir: dir, Listen: listen, Timings: timings, RetryMin: time.Second, RetryMax: time.Minute,
		Rand: rand.New(rand.NewPCG(1, 2)), ProvesPort: downloading.PublicPeer(false), Log: discard, //nolint:gosec // seeded on purpose so tests replay
		Volume: func(context.Context, string) (storing.Volume, error) { return storing.Volume{}, nil },
	})
	node := downloading.NewNode(builtin, downloading.NodeConfig{Timings: timings, Rand: rand.New(rand.NewPCG(3, 4)), Log: discard}) //nolint:gosec // seeded on purpose so tests replay
	h, err := role.NewHolding(t.Context(), role.HoldingConfig{
		Local:    local,
		Upstream: role.OverLink(w.Link, c.node.Key, linktest.Timings),
		Held:     []role.Held{downloading.Def.Hold(node.Run(local, downloadingv1connect.UnimplementedDownloadingControlServiceHandler{}), node.Release)},
		Timings:  role.Timings{RestartMin: time.Second, RestartMax: time.Minute, Beat: 15 * time.Second},
		Rand:     rand.New(rand.NewPCG(5, 6)), //nolint:gosec // seeded on purpose so tests replay
		Log:      discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { _ = builtin.Loop(ctx) })
	wg.Go(func() { _ = h.Run(ctx) })
	t.Cleanup(func() {
		cancel()
		_ = e.Close()
		wg.Wait()
	})
	return local
}
