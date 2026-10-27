package sim_test

import (
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
	transferv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
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

// parked holds seedbox's sender at the at-th chunk it sends until release
// closes, so a test can act while the Transfer is mid-copy.
type parked struct {
	reached, release chan struct{}
	at               int32
	sends            atomic.Int32
}

// newParked parks at the second chunk, with the first free to arrive.
func newParked() *parked { return parkedAt(2) }

func parkedAt(at int32) *parked {
	return &parked{reached: make(chan struct{}), release: make(chan struct{}), at: at}
}

func (p *parked) sent(transfers.ID, int, int, []byte) {
	if p.sends.Add(1) == p.at {
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

type transferRow struct {
	State    string `json:"state"`
	Reason   string `json:"reason"`
	Verified int64  `json:"verified"`
}

// theTransfer is the world's one Transfer as listTransfers reads it, the
// zero row while it lists none.
func theTransfer(t *testing.T, a *worldtest.API) transferRow {
	t.Helper()
	listed := decode[struct {
		Transfers []transferRow `json:"transfers"`
	}](t, "listTransfers", a.Admin("home", http.MethodGet, "/api/v1/transfers", "")).Transfers
	if len(listed) != 1 {
		return transferRow{}
	}
	return listed[0]
}

// TestATransferNeedingSpaceKeepsTheBytesNasVerified fills nas's root once
// nas verified a chunk: the Transfer needs space, and listTransfers still
// counts the bytes nas keeps for resume.
func TestATransferNeedingSpaceKeepsTheBytesNasVerified(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newParked()
		w := sim.Launch(t, seedboxNAS(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Test.Transfers.Sent = p.sent }))
		search(t, w, addOrchard(t, w))
		<-p.reached
		a := w.API()
		eventually(t, "nas verifies a chunk", func() bool { return theTransfer(t, a).Verified > 0 })
		kept := theTransfer(t, a).Verified
		lift := w.Disk("nas", filepath.Join(w.Manifest().MediaDir, "movies"), fault.Full{})
		close(p.release)
		eventually(t, "the Transfer needs space", func() bool { return theTransfer(t, a).State == "needs_space" })
		if got := theTransfer(t, a); got.Verified < kept {
			t.Fatalf("listTransfers = %+v while it needs space, want at least the %d bytes nas verified", got, kept)
		}
		lift()
		expectImported(t, await(t, w))
	})
}

// TestATransferWhoseLaneCarriesItsFirstChunkIsTransferring holds
// seedbox's first chunk on its lane, as a slow send limit does: the lane
// is up and the sender answered, so the Transfer copies rather than
// connects.
func TestATransferWhoseLaneCarriesItsFirstChunkIsTransferring(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := parkedAt(1)
		w := sim.Launch(t, seedboxNAS(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Test.Transfers.Sent = p.sent }))
		search(t, w, addOrchard(t, w))
		<-p.reached
		a := w.API()
		eventually(t, "the Transfer copies while its first chunk is on the way", func() bool {
			got := theTransfer(t, a)
			return got.State == "copying" && got.Reason == "" && got.Verified == 0
		})
		close(p.release)
		expectImported(t, await(t, w))
	})
}

// cutLanes counts the lanes a fault cut while seedbox sent on them.
type cutLanes struct{ n atomic.Int32 }

func (c *cutLanes) wrap(_ transfers.ID, _ transferv1.Route, conn net.Conn) net.Conn {
	return &cutLane{Conn: conn, cuts: c}
}

type cutLane struct {
	net.Conn
	cuts *cutLanes
	once sync.Once
}

func (l *cutLane) Write(b []byte) (int, error) {
	n, err := l.Conn.Write(b)
	if errors.Is(err, fault.ErrCut) {
		l.once.Do(func() { l.cuts.n.Add(1) })
	}
	return n, err //nolint:wrapcheck // the lane's own error, as the engine expects
}

// TestATransferBetweenWorkersResumesAcrossCutLanes cuts every connection
// between nas and seedbox after 6 MiB, mid-chunk on any lane that carries
// two. Each cut loses the one chunk in flight on its lane, and a lane asks
// only for chunks nas hasn't verified and no other lane has asked for: so
// seedbox sends each chunk once, and once more for each lane cut, and the
// file arrives whole.
func TestATransferBetweenWorkersResumesAcrossCutLanes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		counted := &sends{by: map[int]int{}, flip: -1}
		cut := &cutLanes{}
		w := sim.Launch(t, seedboxNAS(t), sim.WithWalk(), onSeedbox(func(e *node.Env) {
			e.Test.Transfers = transfers.Hooks{Sent: counted.sent, Lane: cut.wrap}
		}))
		flip(t, w.Path("nas", "seedbox"), fault.Cut(6<<20))
		search(t, w, addOrchard(t, w))
		expectImported(t, await(t, w))
		got := counted.counts()
		if len(got) != 5 {
			t.Fatalf("seedbox sent chunks %v, want each of the 5", got)
		}
		cuts := int(cut.n.Load())
		if cuts == 0 {
			t.Fatal("no lane was cut, want the 6 MiB cut to break the lanes that carry two chunks")
		}
		sent := 0
		for _, n := range got {
			sent += n
		}
		if sent > len(got)+cuts {
			t.Fatalf("seedbox sent %d chunks (%v) over %d cut lanes, want at most %d: only a chunk lost to a cut crosses again", sent, got, cuts, len(got)+cuts)
		}
		expectPlaced(t, w)
	})
}

// TestAWriteOffMidTransferFailsTheGrabAndCancelsTheTransfer writes off
// Downloading on seedbox while it sends, the Grab reads its Transfer
// copying and the Transfer is named for its Work: the Grab fails
// node_written_off at transferring, the Transfer's record goes in the
// same Write, and nas deletes what it staged.
func TestAWriteOffMidTransferFailsTheGrabAndCancelsTheTransfer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newParked()
		w := sim.Launch(t, seedboxNAS(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Test.Transfers.Sent = p.sent }))
		search(t, w, addOrchard(t, w))
		<-p.reached
		eventually(t, "the transferring Grab reads its Transfer's bytes", func() bool {
			gs := grabsListed(t, w)
			return len(gs) == 1 && gs[0].Stage == "transferring" && gs[0].Progress != nil &&
				gs[0].Progress.Total == int64(len(sim.Payload(release)))
		})
		listed := decode[struct {
			Transfers []struct {
				JobName string `json:"jobName"`
				State   string `json:"state"`
			} `json:"transfers"`
		}](t, "listTransfers", w.API().Admin("home", http.MethodGet, "/api/v1/transfers", "")).Transfers
		if len(listed) != 1 || listed[0].JobName != "The Quiet Orchard" || listed[0].State != "copying" {
			t.Fatalf("listTransfers = %+v, want the Grab's Transfer named The Quiet Orchard, copying", listed)
		}
		a := w.API()
		writeOff(t, a, seedbox(t, a).ID)
		g := await(t, w)
		close(p.release)
		expectFailed(t, g, "transferring", "node_written_off", "The Admin wrote off what this Grab's Node downloads")
		if n := transferRecords(t, w); n != 0 {
			t.Fatalf("home keeps %d Transfer records once the Grab failed, want none", n)
		}
		eventually(t, "nas deletes the canceled Transfer's staging", func() bool { return len(staging(t, w)) == 0 })
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
