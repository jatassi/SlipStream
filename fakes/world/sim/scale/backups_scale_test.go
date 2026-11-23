//go:build scale

package scale_test

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/contenttypes/series"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/lifecycle"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node"
	nodespkg "github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	libraryMiB   = flag.Int("scale.library", 1024, "MiB of library.db to back up, mostly the torrents' metainfo")
	workers      = flag.Int("scale.nodes", 10, "Workers that hold torrents")
	heldEach     = flag.Int("scale.torrents", 5000, "torrents each Worker's client holds")
	holdingsPage = flag.Int("scale.page", 1000, "Holdings entries per page and per adoption Write")
)

// The share of each Worker's held torrents the library doesn't record, and
// of its recorded torrents it doesn't hold.
const (
	unknownShare = 0.10
	notHeldShare = 0.05
)

// TestBackupsScale is the scale simulation of Backups and restore, run on
// demand and never in CI:
//
//	go test -tags scale -run TestBackupsScale ./fakes/world/sim/scale/ -v -timeout 2h
//
// It builds a library of about 1 GiB whose Controller records torrents on
// ten Workers, takes a Backup while a writer commits, stages its restore on
// a new machine, and begins that restore's Reconciling: each Worker reports
// 5,000 held torrents, 10% of them unknown, and lacks 5% of those recorded
// on it. It reports the Backup's time, the longest Write while it ran, the
// staging time, and each Worker's adoption time, and asserts only that
// every Worker is adopted.
func TestBackupsScale(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	schemas := node.LibrarySchemas(contenttype.NewSet(movie.Def, series.Def))
	lib, err := storage.OpenLibrary(ctx, filepath.Join(dir, storage.LibraryFile), schemas, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })
	keys, err := secret.Open(ctx, lib, filepath.Join(dir, secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}

	known, unknown := heldSplit()
	notHeld := int(float64(known) * notHeldShare / (1 - notHeldShare))
	recorded := known + notHeld
	pieces := max(1, (*libraryMiB<<20)/(*workers*recorded)/20)
	ids := make([]role.NodeID, *workers)
	held := make([][]torrent, *workers)
	start := time.Now()
	for i := range ids {
		ids[i] = role.NodeID{UUID: storage.NewUUID()}
		write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return nodestest.Insert(ctx, tx, ids[i], fmt.Sprintf("worker-%d", i+1))
		})
		var mine []torrent
		for from := 0; from < recorded; from += *holdingsPage {
			batch := torrents(t, i, from, min(*holdingsPage, recorded-from), pieces)
			write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				for _, tr := range batch {
					want := dc.Want{Target: dc.Target{Node: ids[i]}, Metainfo: tr.meta, Raw: tr.raw, ContentType: works.TypeID("movie"), WrittenOff: func(role.NodeID) bool { return false }}
					if _, _, err := dc.Attach(ctx, tx, want, storage.Now()); err != nil {
						return err
					}
				}
				return nil
			})
			mine = append(mine, batch...)
		}
		held[i] = append(mine[:known:known], torrents(t, i, recorded, unknown, 1)...)
	}
	size, err := lib.Estimate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("library: %d Workers, each recording %d torrents and holding %d, %d of them unknown and %d recorded ones not held; %s, built in %s",
		*workers, recorded, known+unknown, unknown, notHeld, backups.Size(size), time.Since(start).Round(time.Second))

	engine := newEngine(t, dir, 1)
	took, longest, entry := backUpUnderAWriter(t, lib, keys, engine, ids[0])
	t.Logf("Backup: %s of %s in %s; the longest Write while it ran took %s", entry.Name, backups.Size(entry.Size), took.Round(time.Millisecond), longest.Round(time.Millisecond))

	vault, err := engine.Vault(keys)
	if err != nil {
		t.Fatal(err)
	}
	_, pass, err := vault.Current(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spare := t.TempDir()
	stager := &backups.Stager{Engine: newEngine(t, spare, 2), Storage: storage.DefaultTimings(), Log: slog.New(slog.DiscardHandler)}
	start = time.Now()
	staged, err := stager.Stage(ctx, filepath.Join(dir, "backups", entry.Name.String()), pass, true, lifecycle.ByRestore)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("restore on a new machine: staged and migrated in %s, listing %d Nodes", time.Since(start).Round(time.Millisecond), len(staged.Summary.Nodes))
	if err := stager.Discard(); err != nil {
		t.Fatal(err)
	}

	adopt(t, lib, ids, held)
}

func heldSplit() (known, unknown int) {
	unknown = int(float64(*heldEach) * unknownShare)
	return *heldEach - unknown, unknown
}

func write(t *testing.T, lib *storage.DB[storage.Library], fn func(context.Context, *storage.Tx[storage.Library]) error) {
	t.Helper()
	if err := lib.Write(t.Context(), fn); err != nil {
		t.Fatal(err)
	}
}

func newEngine(t *testing.T, dir string, machine byte) *backups.Engine {
	t.Helper()
	index, err := backups.OpenIndex(filepath.Join(dir, backups.IndexFile), 0)
	if err != nil {
		t.Fatal(err)
	}
	return backups.NewEngine(backups.EngineConfig{
		Dir: dir, Folder: filepath.Join(dir, "backups"), Index: index, Schemas: node.LibrarySchemas(contenttype.NewSet(movie.Def, series.Def)),
		ControllerKey: append([]byte{machine}, make([]byte, 31)...), Version: "2.1.0", Timings: backups.DefaultTimings(),
		Free:  func(context.Context, string) (int64, error) { return 1 << 50, nil },
		Clock: time.Now,
		Log:   slog.New(slog.DiscardHandler),
	})
}

// backUpUnderAWriter takes a Backup of lib while a writer attaches one
// torrent per Write, and answers how long the Backup took and the longest
// of those Writes.
func backUpUnderAWriter(t *testing.T, lib *storage.DB[storage.Library], keys *secret.Keys, engine *backups.Engine, onto role.NodeID) (took, longest time.Duration, e backups.Entry) {
	t.Helper()
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := 0; ; i++ {
			select {
			case <-done:
				return
			default:
			}
			tr := torrents(t, 1_000_000, i, 1, 1)[0]
			start := time.Now()
			write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				want := dc.Want{Target: dc.Target{Node: onto}, Metainfo: tr.meta, Raw: tr.raw, ContentType: works.TypeID("movie"), WrittenOff: func(role.NodeID) bool { return false }}
				_, _, err := dc.Attach(ctx, tx, want, storage.Now())
				return err
			})
			longest = max(longest, time.Since(start))
		}
	})
	start := time.Now()
	e, err := engine.TakeOffline(t.Context(), lib, keys, archive.Manual)
	took = time.Since(start)
	close(done)
	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}
	return took, longest, e
}

type torrent struct {
	raw  []byte
	meta bencode.Metainfo
}

// torrents is n torrents of Worker w from the i-th on, each with pieces
// random piece hashes, which set its metainfo's size and don't compress.
func torrents(t *testing.T, w, from, n, pieces int) []torrent {
	t.Helper()
	out := make([]torrent, n)
	hashes := make([]byte, 20*pieces)
	for k := range out {
		name := fmt.Sprintf("Worker.%d.Torrent.%d.mkv", w, from+k)
		_, _ = rand.Read(hashes)
		binary.BigEndian.PutUint64(hashes, uint64(w)<<32|uint64(from+k)) //nolint:gosec // both are small and positive
		raw, err := bencode.Encode(map[string]any{
			"announce": "https://tracker.example/announce",
			"info":     map[string]any{"name": name, "length": int64(pieces) << 18, "piece length": int64(1 << 18), "pieces": string(hashes), "private": int64(1)},
		})
		if err != nil {
			t.Fatal(err)
		}
		m, err := bencode.DecodeMetainfo(raw)
		if err != nil {
			t.Fatal(err)
		}
		out[k] = torrent{raw: raw, meta: m}
	}
	return out
}

type quiet struct{}

func (quiet) Publish(context.Context, reconciling.Changed) {}

type roster struct{}

func (roster) Standing(nodespkg.ID) (nodespkg.Standing, time.Time) {
	return nodespkg.Standing(0), time.Time{}
}

func (roster) Unknowns() []nodespkg.Unknown { return nil }

// adopt begins a restore's Reconciling on lib and has every Worker report
// held, a page at a time, and reports how long each took to be adopted
// from its first page.
func adopt(t *testing.T, lib *storage.DB[storage.Library], ids []role.NodeID, held [][]torrent) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	meta := map[bencode.InfoHash][]byte{}
	for _, trs := range held {
		for _, tr := range trs {
			meta[tr.meta.InfoHash] = tr.raw
		}
	}
	var rec entry.Recorder
	rec, _ = entrytest.New("reconciling", reconciling.Entries())
	keeper := reconciling.New(reconciling.Config{
		Library: lib,
		Self:    func() role.NodeID { return role.NodeID{UUID: storage.NewUUID()} },
		History: &rec,
		Touch:   func(...role.NodeID) {},
		Metainfo: func(_ context.Context, _ role.NodeID, h bencode.InfoHash, _ *dc.ClientID) ([]byte, error) {
			if raw, ok := meta[h]; ok {
				return raw, nil
			}
			return nil, rpc.Errorf(rpc.CodeNotFound, "no resume data")
		},
		Page: *holdingsPage, Max: 100_000,
		Rehome: func(context.Context, identity.Admin, roots.ID, roots.Rehome) (roots.View, error) {
			return roots.View{}, nil
		},
		Scan:   func(context.Context, []roots.ID) error { return nil },
		Ready:  func(context.Context) ([]roots.Root, error) { return nil, nil },
		Hub:    live.NewHub([]live.Event{roots.ChangedEvent}),
		Ending: func(context.Context, *storage.Tx[storage.Library]) error { return nil },
		Ended:  func(context.Context) {},
		RootWords: func(context.Context, storage.Reader[storage.Library]) (map[roots.ID]string, error) {
			return nil, nil
		},
		Roster: roster{},
		Look: func(_ context.Context, _ role.NodeID, _ string, files []dc.LookFile) ([]bool, error) {
			return make([]bool, len(files)), nil
		},
		Forgot:  func() {},
		Changed: quiet{},
		Log:     slog.New(slog.DiscardHandler),
	})
	clients := dc.New(dc.Config{Library: lib, Keeps: keeper, HoldingsPage: *holdingsPage, HoldingsMax: 100_000, Touch: func(...role.NodeID) {}})
	go func() { _ = keeper.Run(ctx) }()
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return reconciling.Begin(ctx, tx, reconciling.BeginArgs{ID: "scale", Cause: reconciling.Restore, Backup: "scale", TakenAt: time.Now()})
	})

	var mu sync.Mutex
	took := make([]time.Duration, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Go(func() {
			start := time.Now()
			round := roundOf(t, lib, keeper, id)
			pages := (len(held[i]) + *holdingsPage - 1) / *holdingsPage
			for p := range pages {
				entries := make([]*dlv1.Holding, 0, *holdingsPage)
				for _, tr := range held[i][p**holdingsPage : min((p+1)**holdingsPage, len(held[i]))] {
					entries = append(entries, &dlv1.Holding{
						InfoHash: tr.meta.InfoHash[:], Content: &dlv1.HoldingContent{Name: tr.meta.Info.Name, Size: tr.meta.Info.Files[0].Length}, Done: 1, SeedSeconds: 3600,
						Hosts: []string{"tracker.example"}, Private: true, SaveFolder: "/downloads",
					})
				}
				rep := &dlv1.Report{Kind: &dlv1.Report_Holdings{Holdings: &dlv1.Holdings{Round: round, Page: uint32(p), Pages: uint32(pages), Entries: entries}}} //nolint:gosec // pages are few
				write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
					return clients.Apply(ctx, tx, id, rep)
				})
				keeper.Applied(id, string(role.Downloading))
			}
			for pending(t, lib)[id] {
				time.Sleep(20 * time.Millisecond)
			}
			mu.Lock()
			took[i] = time.Since(start)
			mu.Unlock()
		})
	}
	wg.Wait()
	for i, d := range took {
		t.Logf("adoption of worker-%d: %d torrents held, in %s from its first page", i+1, len(held[i]), d.Round(time.Millisecond))
	}
	t.Logf("adoption: slowest Worker %s", slices.Max(took).Round(time.Millisecond))
}

func roundOf(t *testing.T, lib *storage.DB[storage.Library], keeper *reconciling.Service, id role.NodeID) uint64 {
	t.Helper()
	var round uint64
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		in, err := dc.Derive{Keeps: keeper, HoldingsPage: *holdingsPage}.Intent(ctx, r, id)
		round = in.GetHoldings().GetRound()
		return err
	})
	if err != nil || round == 0 {
		t.Fatalf("worker's intent asks for Holdings round %d, %v", round, err)
	}
	return round
}

func pending(t *testing.T, lib *storage.DB[storage.Library]) map[nodespkg.ID]bool {
	t.Helper()
	var out map[nodespkg.ID]bool
	if err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		out, err = reconciling.Pending(ctx, r)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return out
}
