package sim_test

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
)

func seedboxNAS(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("seedbox-nas")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// parked holds seedbox's sender at the first chunk it sends until
// release closes, so a test can act while the Transfer is mid-copy.
type parked struct {
	reached, release chan struct{}
	held             atomic.Bool
}

func newParked() *parked { return &parked{reached: make(chan struct{}), release: make(chan struct{})} }

func (p *parked) sent(transfers.ID, int, int, []byte) {
	if p.held.CompareAndSwap(false, true) {
		close(p.reached)
		<-p.release
	}
}

func transferRecords(t *testing.T, w *sim.World) int64 {
	t.Helper()
	var n int64
	if err := w.Node("home").Half().Library.QueryRowContext(t.Context(), "SELECT count(*) FROM transfers_record").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// staging is what the movies root's staging folder holds.
func staging(t *testing.T, w *sim.World) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(w.Manifest().MediaDir, "movies", storing.StagingDir))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func expectFromSeedbox(t *testing.T, w *sim.World, g grabView) {
	t.Helper()
	if box := w.API().Node("seedbox").NodeID; g.NodeID != box {
		t.Fatalf("the Grab's Node is %s, want seedbox %s, which downloads", g.NodeID, box)
	}
}

// TestAGrabOnSeedboxLandsInNassRootThroughATransfer grabs on seedbox-nas:
// seedbox downloads, the Transfer copies the file to nas's root, and once
// it is placed the Transfer's record and staging are gone.
func TestAGrabOnSeedboxLandsInNassRootThroughATransfer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, seedboxNAS(t), sim.WithWalk())
		search(t, w, addOrchard(t, w))
		g := await(t, w)
		expectImported(t, g)
		expectFromSeedbox(t, w, g)
		expectPlaced(t, w)
		if n := transferRecords(t, w); n != 0 {
			t.Fatalf("home keeps %d Transfer records once the file is placed, want none", n)
		}
	})
}

// TestATransferBetweenWorkersSurvivesASeedboxKillAndAControllerRestart
// stops seedbox mid-Transfer and restarts home while it is down: once
// seedbox is back the Transfer resumes and the file arrives whole.
func TestATransferBetweenWorkersSurvivesASeedboxKillAndAControllerRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newParked()
		w := sim.Launch(t, seedboxNAS(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Test.Transfers.Sent = p.sent }))
		search(t, w, addOrchard(t, w))
		<-p.reached
		box := w.Node("seedbox")
		stopped := make(chan struct{})
		go func() {
			box.Stop()
			close(stopped)
		}()
		synctest.Wait()
		close(p.release)
		<-stopped
		h := w.Node("home")
		link := h.Ready().Listeners.Link
		h.Stop()
		h.Start(func(e *node.Env) { e.Environ = append(e.Environ, "SLIPSTREAM_LINK_LISTEN="+link.String()) })
		box.Start()
		g := await(t, w)
		expectImported(t, g)
		expectFromSeedbox(t, w, g)
		expectPlaced(t, w)
	})
}

// TestATransferBetweenWorkersResumesAcrossCutLanes cuts every connection
// between nas and seedbox after 6 MiB: each new lane asks only for the
// chunks nas hasn't verified, so no chunk crosses more than twice, and the
// file arrives whole.
func TestATransferBetweenWorkersResumesAcrossCutLanes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		counted := &sends{by: map[int]int{}, flip: -1}
		w := sim.Launch(t, seedboxNAS(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Test.Transfers.Sent = counted.sent }))
		flip(t, w.Path("nas", "seedbox"), fault.Cut(6<<20))
		search(t, w, addOrchard(t, w))
		expectImported(t, await(t, w))
		got := counted.counts()
		if len(got) != 5 {
			t.Fatalf("seedbox sent chunks %v, want each of the 5", got)
		}
		for i, n := range got {
			if n > 2 {
				t.Fatalf("seedbox sent chunk %d %d times (%v), want at most twice: a new lane resumes from what nas verified", i, n, got)
			}
		}
		expectPlaced(t, w)
	})
}

// TestAWriteOffMidTransferFailsTheGrabAndCancelsTheTransfer writes off
// Downloading on seedbox while it sends: the Grab fails node_written_off
// at transferring, the Transfer's record goes in the same Write, and nas
// deletes what it staged.
func TestAWriteOffMidTransferFailsTheGrabAndCancelsTheTransfer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newParked()
		w := sim.Launch(t, seedboxNAS(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Test.Transfers.Sent = p.sent }))
		search(t, w, addOrchard(t, w))
		<-p.reached
		a := w.API()
		writeOff(t, a, seedbox(t, a).ID)
		g := await(t, w)
		close(p.release)
		expectFailed(t, g, "transferring", "node_written_off", "The Admin wrote off what this Grab's Node downloads")
		if n := transferRecords(t, w); n != 0 {
			t.Fatalf("home keeps %d Transfer records once the Grab failed, want none", n)
		}
		eventually(t, "nas deletes the cancelled Transfer's staging", func() bool { return len(staging(t, w)) == 0 })
	})
}

func moviesRoot(t *testing.T, w *sim.World) rootRow {
	t.Helper()
	roots := decode[struct {
		Roots []rootRow `json:"roots"`
	}](t, "listRoots", w.API().Admin("home", http.MethodGet, "/api/v1/roots", "")).Roots
	if len(roots) != 1 {
		t.Fatalf("listRoots = %+v, want the movies root", roots)
	}
	return roots[0]
}

func rehome(t *testing.T, w *sim.World, root rootRow, to, path string) {
	t.Helper()
	body := `{"nodeId":"` + w.API().Node(to).NodeID + `","path":` + strconv.Quote(path) + `}`
	if r := w.API().Admin("home", http.MethodPatch, "/api/v1/roots/"+root.ID, body); r.Status != http.StatusOK {
		t.Fatalf("rehomeRoot onto %s = %d %s, want 200", to, r.Status, r.Body)
	}
}

// TestARootRehomedMidTransferTakesTheFileWhereItIsNow re-homes nas's root
// while seedbox sends: onto home at a copy of its folder, the Transfer
// starts again at home and the file lands there; onto seedbox, which
// downloaded it, the Transfer stops and the file is linked to the
// download, with no copy.
func TestARootRehomedMidTransferTakesTheFileWhereItIsNow(t *testing.T) {
	t.Run("onto home", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := seedboxNAS(t)
			s.Nodes[0].Roles = append(s.Nodes[0].Roles, "storage")
			p := newParked()
			w := sim.Launch(t, s, sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Test.Transfers.Sent = p.sent }))
			search(t, w, addOrchard(t, w))
			<-p.reached
			root := moviesRoot(t, w)
			moved := filepath.Join(t.TempDir(), "movies")
			if err := os.CopyFS(moved, os.DirFS(root.Path)); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(filepath.Join(moved, storing.StagingDir)); err != nil {
				t.Fatal(err)
			}
			rehome(t, w, root, "home", moved)
			close(p.release)
			g := await(t, w)
			expectImported(t, g)
			rel, err := filepath.Rel("movies", filepath.FromSlash(placed))
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(moved, rel))
			if err != nil || blake3.Sum256(data) != blake3.Sum256(sim.Payload(release)) {
				t.Fatalf("home's root holds %d bytes (%v), want the release payload", len(data), err)
			}
			eventually(t, "nas deletes what it staged", func() bool { return len(staging(t, w)) == 0 })
		})
	})
	t.Run("onto seedbox", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := seedboxNAS(t)
			s.Nodes[1].Roles = append(s.Nodes[1].Roles, "storage")
			p := newParked()
			w := sim.Launch(t, s, sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Test.Transfers.Sent = p.sent }))
			search(t, w, addOrchard(t, w))
			<-p.reached
			root := moviesRoot(t, w)
			rehome(t, w, root, "seedbox", root.Path)
			g := await(t, w)
			close(p.release)
			expectImported(t, g)
			expectPlaced(t, w)
			linked, err := os.Stat(filepath.Join(w.Manifest().MediaDir, filepath.FromSlash(placed)))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(downloads(t, w), func(d os.FileInfo) bool { return os.SameFile(d, linked) }) {
				t.Fatal("the placed file isn't seedbox's download linked: the Transfer copied it")
			}
		})
	})
}

// downloads is every file in seedbox's downloads folder.
func downloads(t *testing.T, w *sim.World) []os.FileInfo {
	t.Helper()
	var out []os.FileInfo
	err := filepath.WalkDir(w.Node("seedbox").DataDir().Resolve("downloads"), func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		out = append(out, info)
		return err //nolint:wrapcheck // the test reports it as is
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
