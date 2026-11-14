package sim_test

import (
	"bytes"
	"context"
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
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// engines follows the torrents in a Node's engine, the last one it
// opened, and how many its engines added.
type engines struct {
	mu      sync.Mutex
	held    map[engine.Handle]bool
	added   int
	removed []removal
}

type removal struct {
	at          time.Time
	deleteFiles bool
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

func (f followed) Remove(h engine.Handle, deleteFiles bool) error {
	err := f.Engine.Remove(h, deleteFiles)
	if err == nil {
		f.e.mu.Lock()
		delete(f.e.held, h)
		f.e.removed = append(f.e.removed, removal{at: time.Now(), deleteFiles: deleteFiles})
		f.e.mu.Unlock()
	}
	return err
}

func (e *engines) removals() []removal {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.removed)
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
	return simtest.Decode[worldtest.Outcome](t, "writeOff", r)
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
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		box := &engines{}
		w := sim.Launch(t, simtest.Home(t), sim.WithWalk(), simtest.OnSeedbox(func(e *node.Env) { e.Engine = box.wrap(e.Engine) }))
		simtest.Search(t, w, simtest.AddOrchard(t, w))
		simtest.ExpectImported(t, simtest.Await(t, w))
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
		simtest.Eventually(t, "seedbox's engine lets the torrent go and the forfeit settles", func() bool {
			row := seedbox(t, a)
			return box.torrents() == 0 && forfeits(t, w) == 0 && stateOf(row, "downloading") == "none" &&
				row.Facts != nil && !slices.Contains(row.Facts.Running, "downloading")
		})
		expectDownloadKept(t, w)
		expectPlaced(t, w)
	})
}

// TestAWriteOffOnAnOfflineSeedboxSettlesWhenItReturns writes off
// Downloading on seedbox while it is stopped and its Import waits to probe:
// the Grab fails node_written_off and the forfeit goes at once, and once
// seedbox is back its engine holds no torrent and its file is kept.
func TestAWriteOffOnAnOfflineSeedboxSettlesWhenItReturns(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		box := &engines{}
		reached := make(chan struct{})
		var held atomic.Bool
		w := sim.Launch(t, simtest.Home(t), sim.WithWalk(), sim.Edit(func(name string, e *node.Env) {
			switch name {
			case "seedbox":
				e.Engine = box.wrap(e.Engine)
			case "home":
				e.Test.BeforeImport = func(ctx context.Context, s imports.Step) {
					if s.At == imports.AtProbe && held.CompareAndSwap(false, true) {
						close(reached)
						<-ctx.Done()
					}
				}
			}
		}))
		simtest.Search(t, w, simtest.AddOrchard(t, w))
		<-reached
		a := w.API()
		id := seedbox(t, a).ID
		added := box.adds()
		w.Node("seedbox").Stop()

		writeOff(t, a, id)
		var failed simtest.GrabView
		simtest.Eventually(t, "the Grab fails and the forfeit settles", func() bool {
			gs := simtest.Decode[struct {
				Grabs []simtest.GrabView `json:"grabs"`
			}](t, "listGrabs", a.Admin("home", http.MethodGet, "/api/v1/grabs", "")).Grabs
			if len(gs) == 1 {
				failed = gs[0]
			}
			return failed.Stage == "failed" && forfeits(t, w) == 0
		})
		expectFailed(t, failed, "importing", "node_written_off", "The Admin wrote off what this Grab's Node downloads")

		w.Node("seedbox").Start()
		a = w.API()
		simtest.Eventually(t, "seedbox is back in step and its engine holds no torrent", func() bool {
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

// heldFirst holds the first Grab's file for the Admin: its probe times out
// three times, then reads.
func heldFirst() sim.Option {
	probes := simtest.NewProbes()
	probes.Play(release+".mkv", simtest.ProbeTimeout, simtest.ProbeTimeout, simtest.ProbeTimeout)
	return probes.Option()
}

// declineFirst answers Don't import on the first Grab's held file once its
// hold offers it, so the Grab fails declined, nothing is blocklisted and
// its torrent keeps seeding, and returns the Grab.
func declineFirst(t *testing.T, w *sim.World) simtest.GrabView {
	t.Helper()
	simtest.Eventually(t, "the first Grab's file offers Don't import", func() bool {
		ims := grabImports(t, w)
		return len(ims) == 1 && w.API().Admin("home", http.MethodPost, "/api/v1/imports/"+ims[0].ID+"/files/0", `{"action":"decline"}`).Status == http.StatusOK
	})
	first := simtest.Await(t, w)
	expectFailed(t, first, "importing", "declined", "The Admin declined every file of the download.")
	return first
}

// TestANewGrabAttachesToTheTorrentADrainingSeedboxSeeds fails seedbox's
// first Grab's Import declined, so its torrent keeps seeding, then unticks
// Downloading on seedbox: seedbox drains. The next search's Grab is
// fetched through seedbox, which still searches, and though the router
// picks home, it attaches to the Torrent seedbox already holds instead of
// downloading the release again.
func TestANewGrabAttachesToTheTorrentADrainingSeedboxSeeds(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		box := &engines{}
		w := sim.Launch(t, simtest.Home(t), sim.WithWalk(), heldFirst(), simtest.OnSeedbox(func(e *node.Env) { e.Engine = box.wrap(e.Engine) }))
		work := simtest.AddOrchard(t, w)
		simtest.Search(t, w, work)
		first := declineFirst(t, w)
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

		simtest.Search(t, w, work)
		var next simtest.GrabView
		simtest.Eventually(t, "the second Grab imports", func() bool {
			gs := simtest.GrabsListed(t, w)
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
