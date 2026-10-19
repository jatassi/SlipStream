package sim_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// engines follows the torrents in a Node's engine, the last one it
// opened, and how many its engines added.
type engines struct {
	mu    sync.Mutex
	held  map[engine.Handle]bool
	added int
}

func (e *engines) wrap(open func(engine.Config) (engine.Engine, error)) func(engine.Config) (engine.Engine, error) {
	return func(cfg engine.Config) (engine.Engine, error) {
		eng, err := open(cfg)
		e.mu.Lock()
		e.held = map[engine.Handle]bool{}
		e.mu.Unlock()
		return followed{eng, e}, err
	}
}

func (e *engines) torrents() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.held)
}

func (e *engines) adds() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.added
}

type followed struct {
	engine.Engine
	e *engines
}

func (f followed) Add(p engine.AddParams) (engine.Handle, error) {
	h, err := f.Engine.Add(p)
	if err == nil {
		f.e.mu.Lock()
		f.e.held[h] = true
		f.e.added++
		f.e.mu.Unlock()
	}
	return h, err
}

func (f followed) Remove(h engine.Handle) error {
	err := f.Engine.Remove(h)
	if err == nil {
		f.e.mu.Lock()
		delete(f.e.held, h)
		f.e.mu.Unlock()
	}
	return err
}

func stateOf(r worldtest.Row, role string) string {
	for _, v := range r.Roles {
		if v.Role == role {
			return v.State
		}
	}
	return "none"
}

func forfeits(t *testing.T, w *sim.World) int64 {
	t.Helper()
	var n int64
	if err := w.Node("home").Half().Library.QueryRowContext(t.Context(), "SELECT count(*) FROM nodes_forfeit").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func writeOff(t *testing.T, a *worldtest.API, id string) worldtest.Outcome {
	t.Helper()
	r := a.Admin("home", http.MethodPost, "/api/v1/nodes/"+id+"/write-offs", `{"roles":["downloading"]}`)
	return decode[worldtest.Outcome](t, "writeOff", r)
}

// eventually waits in fake time until ok, failing with what.
func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for range settled / time.Second {
		if ok() {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("not within %s: %s", settled, what)
}

// expectDownloadKept checks seedbox's downloads folder still holds the
// release's file, whole.
func expectDownloadKept(t *testing.T, w *sim.World) {
	t.Helper()
	dir := w.Node("seedbox").DataDir().Resolve("downloads")
	var kept []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err == nil && bytes.Equal(data, sim.Payload(release)) {
			kept = append(kept, path)
		}
		return err //nolint:wrapcheck // the test reports it as is
	})
	if err != nil || len(kept) != 1 {
		t.Fatalf("seedbox's downloads hold %v whole copies of the release, %v; want one", kept, err)
	}
}

// TestAnUntickedDownloadingRoleDrainsUntilItIsWrittenOff unticks
// Downloading on seedbox once its Grab imported: home takes Downloading
// over, and seedbox keeps seeding with Downloading draining in its facts.
// Writing it off empties seedbox's intent, so the torrent leaves its
// engine with its file kept, and the forfeit goes once nothing is held.
func TestAnUntickedDownloadingRoleDrainsUntilItIsWrittenOff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		box := &engines{}
		w := sim.Launch(t, home(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Engine = box.wrap(e.Engine) }))
		search(t, w, addOrchard(t, w))
		expectImported(t, await(t, w))
		a := w.API()
		id := seedbox(t, a).ID

		out := a.SetRoles("home", id, "searching")
		if words := out.Words(); !slices.Contains(words, "home takes over Downloading, since SlipStream always needs a Node that downloads.") {
			t.Fatalf("unticking seedbox's Downloading says %q, want home taking it over", words)
		}
		synctest.Wait()
		row := seedbox(t, a)
		if st := stateOf(row, "downloading"); st != "draining" || row.Facts == nil || !slices.Contains(row.Facts.Running, "downloading") || slices.Contains(row.Facts.Assigned, "downloading") {
			t.Fatalf("unticked, seedbox stands %s in Downloading with facts %+v; want draining, running and not assigned", st, row.Facts)
		}
		if n := box.torrents(); n != 1 {
			t.Fatalf("draining, seedbox's engine holds %d torrents, want the 1 it seeds", n)
		}

		wrote := writeOff(t, a, id)
		if kinds := wrote.Effects; len(kinds) != 1 || kinds[0].Kind != "writes-off" {
			t.Fatalf("the write-off answers %+v, want one writes-off effect", wrote.Effects)
		}
		eventually(t, "seedbox's engine lets the torrent go and the forfeit settles", func() bool {
			row := seedbox(t, a)
			return box.torrents() == 0 && forfeits(t, w) == 0 && stateOf(row, "downloading") == "none" &&
				row.Facts != nil && !slices.Contains(row.Facts.Running, "downloading")
		})
		expectDownloadKept(t, w)
		expectPlaced(t, w)
	})
}

// TestAWriteOffOnAnOfflineSeedboxSettlesWhenItReturns writes off
// Downloading on seedbox while it is stopped and its Grab waits to probe:
// the Grab fails node_written_off and the forfeit goes at once, and once
// seedbox is back its engine holds no torrent and its file is kept.
func TestAWriteOffOnAnOfflineSeedboxSettlesWhenItReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		box := &engines{}
		reached := make(chan struct{})
		var held atomic.Bool
		w := sim.Launch(t, home(t), sim.WithWalk(), sim.Edit(func(name string, e *node.Env) {
			switch name {
			case "seedbox":
				e.Engine = box.wrap(e.Engine)
			case "home":
				e.Test.BeforeStep = func(ctx context.Context, s grabs.Stage) {
					if s == grabs.Probing && held.CompareAndSwap(false, true) {
						close(reached)
						<-ctx.Done()
					}
				}
			}
		}))
		search(t, w, addOrchard(t, w))
		<-reached
		a := w.API()
		id := seedbox(t, a).ID
		added := box.adds()
		w.Node("seedbox").Stop()

		writeOff(t, a, id)
		var failed grabView
		eventually(t, "the Grab fails and the forfeit settles", func() bool {
			gs := decode[struct {
				Grabs []grabView `json:"grabs"`
			}](t, "listGrabs", a.Admin("home", http.MethodGet, "/api/v1/grabs", "")).Grabs
			if len(gs) == 1 {
				failed = gs[0]
			}
			return failed.Stage == "failed" && forfeits(t, w) == 0
		})
		expectFailed(t, failed, "probing", "node_written_off", "the Admin wrote off what this Grab's Node downloads")

		w.Node("seedbox").Start()
		a = w.API()
		eventually(t, "seedbox is back in step and its engine holds no torrent", func() bool {
			row := seedbox(t, a)
			return row.Link.State == "up" && row.Facts != nil && !slices.Contains(row.Facts.Assigned, "downloading") &&
				!slices.Contains(row.Facts.Running, "downloading") && box.torrents() == 0
		})
		if n := box.adds() - added; n != 0 {
			t.Errorf("back in step, seedbox's engine added %d torrents it was written off, want none", n)
		}
		expectDownloadKept(t, w)
	})
}

// failsOnce fails the first probe it is asked for.
type failsOnce struct {
	media.Prober
	failed atomic.Bool
}

func (p *failsOnce) Probe(ctx context.Context, path string) (media.Probe, error) {
	if p.failed.CompareAndSwap(false, true) {
		return media.Probe{}, errors.New("the probe failed once")
	}
	return p.Prober.Probe(ctx, path)
}

// TestANewGrabAttachesToTheTorrentADrainingSeedboxSeeds fails seedbox's
// first Grab at probing, so its torrent keeps seeding, then unticks
// Downloading on seedbox: seedbox drains. The next search's Grab is
// fetched through seedbox, which still searches, and though the router
// picks home, it attaches to the Torrent seedbox already holds instead of
// downloading the release again.
func TestANewGrabAttachesToTheTorrentADrainingSeedboxSeeds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		box := &engines{}
		w := sim.Launch(t, home(t), sim.WithWalk(), onSeedbox(func(e *node.Env) {
			e.Engine = box.wrap(e.Engine)
			e.Media = &failsOnce{Prober: e.Media}
		}))
		work := addOrchard(t, w)
		search(t, w, work)
		first := await(t, w)
		if first.Stage != "failed" || first.Failure == nil || first.Failure.Stage != "probing" {
			t.Fatalf("seedbox's first Grab ended %s with %+v, want failed at probing", first.Stage, first.Failure)
		}
		a := w.API()
		box1 := seedbox(t, a)
		var home string
		for _, r := range a.Nodes("home") {
			if r.Name == "home" {
				home = r.ID
			}
		}
		a.SetRoles("home", box1.ID, "searching")
		synctest.Wait()
		if st := stateOf(seedbox(t, a), "downloading"); st != "draining" || box.torrents() != 1 {
			t.Fatalf("unticked, seedbox stands %s in Downloading with %d torrents, want draining the 1 it seeds", st, box.torrents())
		}

		search(t, w, work)
		var next grabView
		eventually(t, "the second Grab imports", func() bool {
			gs := grabsListed(t, w)
			if len(gs) == 2 {
				next = gs[0]
			}
			return next.Stage == "imported"
		})
		if next.NodeID != box1.ID || next.FetchNodeID != box1.ID || next.InfoHash != first.InfoHash {
			t.Fatalf("the second Grab was fetched by %s and downloaded by %s with %s, want both seedbox %s with the first Grab's torrent %s, not home %s",
				next.FetchNodeID, next.NodeID, next.InfoHash, box1.ID, first.InfoHash, home)
		}
		if st := stateOf(seedbox(t, a), "downloading"); st != "draining" || box.torrents() != 1 {
			t.Fatalf("after home's Grab, seedbox stands %s in Downloading with %d torrents, want still draining the 1 it seeds", st, box.torrents())
		}
	})
}
