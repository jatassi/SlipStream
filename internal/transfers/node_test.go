package transfers_test

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"log/slog"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

const chunk = 4 << 20

// sizes are two files: five chunks, the last one short, and one chunk and
// a bit.
var sizes = []int{4*chunk + 1000, chunk + 7}

// seeds draw each file's bytes, and name its torrent.
var seeds = []byte{1, 2}

var (
	root  = storage.NewUUID()
	place = int64(3)
)

// end is one Node's Transfers half on a linktest Node, with its own
// local.db and a disk the test can fault.
type end struct {
	t       *testing.T
	name    string
	ln      *linktest.Node
	lanes   *lateLanes
	h       *holder
	local   *storage.DB[storage.Local]
	disk    storing.Disk
	plan    *fault.Plan
	dir     string
	sources *sources
	stager  *stager
	timings transfers.Timings
	hooks   transfers.Hooks

	node *transfers.Node
	stop context.CancelFunc
	done chan struct{}
}

func newEnd(t *testing.T, name string) (*end, linktest.Side) {
	t.Helper()
	base, err := storing.NewOSDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e := &end{
		t: t, name: name, lanes: &lateLanes{}, h: newHolder(),
		local: storagetest.Local(t, transfers.Schema()), plan: fault.NewDiskPlan(), dir: t.TempDir(),
		sources: &sources{paths: map[source.Ref]string{}, absent: map[source.Ref]error{}}, timings: transfers.DefaultTimings(),
	}
	e.disk = e.plan.Disk(base)
	e.stager = &stager{dir: e.dir}
	return e, linktest.Side{
		LaneHandler: e.lanes,
		Streams:     map[link.Kind]func(link.Key, link.Stream){link.KindTransfer: func(k link.Key, s link.Stream) { e.lanes.node().ServeStream(k, s) }},
	}
}

// start runs a Node engine over the end's local.db and disk, as a Node
// process starting does.
func (e *end) start() {
	e.node = transfers.NewNode(transfers.NodeConfig{
		Local: e.local, Disk: e.disk, Link: e.ln.Link, Sources: e.sources, Roots: e.stager,
		Timings: e.timings, Hooks: e.hooks, Log: slog.New(slog.DiscardHandler),
	})
	e.lanes.set(e.node)
	ctx, cancel := context.WithCancel(e.t.Context())
	e.stop, e.done = cancel, make(chan struct{})
	go func() {
		defer close(e.done)
		if err := e.node.Run(ctx, e.h); err != nil {
			e.t.Errorf("%s: Run = %v", e.name, err)
		}
	}()
	e.t.Cleanup(e.halt)
}

// halt stops the engine as a Node process stopping does.
func (e *end) halt() {
	e.stop()
	<-e.done
}

func (e *end) addrs() []string {
	var out []string
	for _, a := range e.ln.Target().Addresses.All() {
		out = append(out, a.String())
	}
	return out
}

// payload writes a source file of n bytes drawn from seed and names it
// torrent i's file 0.
func (e *end) payload(i, n int, seed byte) *tv1.File {
	e.t.Helper()
	b := make([]byte, n)
	_, _ = rand.NewChaCha8([32]byte{seed}).Read(b)
	path := e.source(i)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		e.t.Fatal(err)
	}
	hash := bencode.InfoHash{seed}
	e.sources.put(source.Torrent{Hash: hash}, path, nil)
	return &tv1.File{Position: uint32(i), Size: int64(n), Source: &tv1.Source{Ref: &tv1.Source_Torrent{Torrent: &tv1.TorrentPayload{InfoHash: hash[:]}}}} //nolint:gosec // a test file
}

func (e *end) source(i int) string { return filepath.Join(e.dir, "payload", strconv.Itoa(i)+".mkv") }

func (e *end) staged(id storage.UUID, position int) string {
	return filepath.Join(e.dir, storing.StagingDir, id.String(), strconv.Itoa(position)+".part")
}

// transfer is one Transfer between a sender and a receiver.
type transfer struct {
	t                *testing.T
	id               storage.UUID
	sender, receiver *end
	files            []*tv1.File
	sent             *sent
}

func newTransfer(t *testing.T, sender, receiver *end) *transfer {
	t.Helper()
	return newTransferOf(t, sender, receiver, sizes)
}

// newTransferOf is a Transfer of two files of the sizes given.
func newTransferOf(t *testing.T, sender, receiver *end, sizes []int) *transfer {
	t.Helper()
	tr := &transfer{t: t, id: storage.NewUUID(), sender: sender, receiver: receiver, sent: &sent{n: map[[2]int]int{}}}
	for i, n := range sizes {
		tr.files = append(tr.files, sender.payload(i, n, seeds[i]))
	}
	sender.hooks.Sent = tr.sent.see
	return tr
}

// route gives both ends an intent that runs the Transfer on route, with
// sendLimit on the sender.
func (tr *transfer) route(r tv1.Route, sendLimit int64) {
	tr.sender.h.set(&tv1.Intent{SendLimit: sendLimit, Active: []*tv1.Active{tr.active(tv1.Side_SIDE_SEND, r)}}, role.Live)
	tr.receiver.h.set(&tv1.Intent{Active: []*tv1.Active{tr.active(tv1.Side_SIDE_RECEIVE, r)}, Keep: [][]byte{tr.id.Bytes()}}, role.Live)
}

func (tr *transfer) active(side tv1.Side, r tv1.Route) *tv1.Active {
	a := &tv1.Active{TransferId: tr.id.Bytes(), Side: side, Route: r, Files: tr.files}
	peer := tr.receiver
	if side == tv1.Side_SIDE_RECEIVE {
		peer = tr.sender
		a.Destination = &tv1.Destination{Kind: &tv1.Destination_Root{Root: &tv1.RootDestination{RootId: root.Bytes(), Place: place}}}
	}
	a.Peer = &tv1.Peer{Key: peer.ln.Key[:], Addresses: peer.addrs()}
	return a
}

// expectArrived waits for every file's FileVerified and checks each
// staged file is its source, by BLAKE3.
func (tr *transfer) expectArrived() {
	tr.t.Helper()
	tr.receiver.h.await(tr.t, "every file verified", func(rs []*tv1.Result, _ []*tv1.Facts) bool {
		return len(verified(rs, tr.id)) == len(tr.files)
	})
	if got := verified(tr.receiver.h.results(), tr.id); !slices.Equal(got, []int{0, 1}) {
		tr.t.Fatalf("FileVerified named positions %v, want [0 1]", got)
	}
	for i := range tr.files {
		got, want := sum(tr.t, tr.receiver.staged(tr.id, i)), sum(tr.t, tr.sender.source(i))
		if got != want {
			tr.t.Fatalf("staged file %d's BLAKE3 is %x, want its source's %x", i, got, want)
		}
	}
}

func verified(rs []*tv1.Result, id storage.UUID) []int {
	var out []int
	for _, r := range rs {
		if v := r.GetVerified(); v != nil && bytes.Equal(v.GetTransferId(), id.Bytes()) && v.GetPlace() == place {
			out = append(out, int(v.GetPosition()))
		}
	}
	return out
}

// trio is two Workers' ends and the Controller's, on one fault.Network.
func trio(t *testing.T) (controller *end, workers [2]*end, nw *fault.Network) {
	t.Helper()
	c, cs := newEnd(t, "home")
	a, as := newEnd(t, "seedbox")
	b, bs := newEnd(t, "nas")
	p := linktest.NewTrio(t, cs, [2]linktest.Side{as, bs})
	c.ln, a.ln, b.ln = p.Controller, p.Workers[0], p.Workers[1]
	p.Connect(t)
	return c, [2]*end{a, b}, p.Net
}

func TestATransferArrivesWholeOnEachRoute(t *testing.T) {
	for _, c := range []struct {
		name  string
		route tv1.Route
		ends  func(controller *end, workers [2]*end) (sender, receiver *end)
	}{
		{"the receiver dials", tv1.Route_ROUTE_RECEIVER_DIALS, func(_ *end, w [2]*end) (*end, *end) { return w[0], w[1] }},
		{"the sender dials", tv1.Route_ROUTE_SENDER_DIALS, func(_ *end, w [2]*end) (*end, *end) { return w[0], w[1] }},
		{"either dials", tv1.Route_ROUTE_BOTH_DIAL, func(_ *end, w [2]*end) (*end, *end) { return w[0], w[1] }},
		{"the Node link to the Controller", tv1.Route_ROUTE_NODE_LINK, func(c *end, w [2]*end) (*end, *end) { return w[0], c }},
		{"the Node link from the Controller", tv1.Route_ROUTE_NODE_LINK, func(c *end, w [2]*end) (*end, *end) { return c, w[1] }},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				controller, workers, _ := trio(t)
				sender, receiver := c.ends(controller, workers)
				tr := newTransfer(t, sender, receiver)
				sender.start()
				receiver.start()
				tr.route(c.route, 0)
				tr.expectArrived()
				if moved := receiver.h.moving(tr.id); !slices.Contains(moved, c.route) {
					t.Fatalf("the receiver's facts said bytes moved on %v, want %s", moved, c.route)
				}
			})
		})
	}
}

func TestACutLaneResumesFromTheVerifiedChunks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, nw := trio(t)
		tr := newTransfer(t, w[0], w[1])
		w[1].timings.SyncBytes = chunk
		if _, err := nw.Path(linktest.WorkerAddr, linktest.OtherWorkerAddr).Set(fault.Rule{ID: "cut", Fault: fault.Cut(2*chunk + chunk/2), Times: 1}); err != nil {
			t.Fatal(err)
		}
		w[0].start()
		w[1].start()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		tr.expectArrived()
		for k, n := range tr.sent.counts() {
			if n > 2 {
				t.Fatalf("chunk %d of file %d was sent %d times across one cut", k[1], k[0], n)
			}
		}
		if tr.sent.count(0, 0) != 1 {
			t.Fatal("the chunk verified before the cut was sent again")
		}
	})
}

// interrupt runs tr until the receiver recorded file 0's first three
// chunks, then stops the receiver's engine while the sender holds chunk 3.
func interrupt(t *testing.T) (*transfer, [2]*end) {
	t.Helper()
	_, w, _ := trio(t)
	tr := newTransfer(t, w[0], w[1])
	w[1].timings.SyncBytes = chunk
	held := make(chan struct{})
	tr.sent.setHold(func(position, index int) {
		if position == 0 && index == 3 {
			<-held
		}
	})
	w[0].start()
	w[1].start()
	tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
	w[1].h.await(t, "three chunks verified", func(_ []*tv1.Result, fs []*tv1.Facts) bool {
		return len(fs) > 0 && receiving(fs[len(fs)-1], tr.id).GetVerifiedBytes() >= 3*chunk
	})
	w[1].halt()
	tr.sent.setHold(nil)
	close(held)
	return tr, w
}

func TestAReceiverRestartedMidFileResumes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tr, w := interrupt(t)
		w[1].start()
		tr.expectArrived()
		for i := range 3 {
			if n := tr.sent.count(0, i); n != 1 {
				t.Fatalf("chunk %d, verified before the restart, was sent %d times", i, n)
			}
		}
	})
}

func TestAReceiverRestartedReportsTheBytesItKeepsWhileItConnects(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tr, w := interrupt(t)
		w[0].halt()
		before := len(w[1].h.facts())
		w[1].start()
		w[1].h.await(t, "the kept chunks reported while connecting", func(_ []*tv1.Result, fs []*tv1.Facts) bool {
			if len(fs) == before {
				return false
			}
			r := receiving(fs[len(fs)-1], tr.id)
			return r.GetWait().GetKind() == tv1.Wait_KIND_CONNECTING && r.GetVerifiedBytes() >= 3*chunk
		})
		w[0].start()
		tr.expectArrived()
	})
}

func TestATornStagedFileRestartsFromZero(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tr, w := interrupt(t)
		if err := os.Truncate(w[1].staged(tr.id, 0), chunk); err != nil {
			t.Fatal(err)
		}
		w[1].start()
		tr.expectArrived()
		if n := tr.sent.count(0, 0); n != 2 {
			t.Fatalf("chunk 0 of a torn file was sent %d times, want again after the restart", n)
		}
	})
}

func TestASourceChangedWhileStoppedRestartsItsFile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tr, w := interrupt(t)
		changed := make([]byte, sizes[0])
		_, _ = rand.NewChaCha8([32]byte{9}).Read(changed)
		if err := os.WriteFile(w[0].source(0), changed, 0o600); err != nil {
			t.Fatal(err)
		}
		later := time.Now().Add(time.Hour)
		if err := os.Chtimes(w[0].source(0), later, later); err != nil {
			t.Fatal(err)
		}
		w[1].start()
		tr.expectArrived()
		if n := tr.sent.count(0, 0); n != 2 {
			t.Fatalf("chunk 0 of a changed source was sent %d times, want again: a file of two versions must not arrive", n)
		}
	})
}

func TestALostResumeRecordRestartsItsFile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tr, w := interrupt(t)
		if err := w[1].local.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
			_, err := tx.ExecContext(ctx, "DELETE FROM transfers_resume")
			return err
		}); err != nil {
			t.Fatal(err)
		}
		w[1].start()
		tr.expectArrived()
		if n := tr.sent.count(0, 0); n != 2 {
			t.Fatalf("chunk 0 was sent %d times after its record was lost, want again: nothing unverified is trusted", n)
		}
	})
}

func TestAChunkDamagedAfterHashingIsAskedAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		tr := newTransfer(t, w[0], w[1])
		tr.sent.damage = func(position, index, nth int) bool { return position == 0 && index == 1 && nth == 1 }
		w[0].start()
		w[1].start()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		tr.expectArrived()
		if n := tr.sent.count(0, 1); n != 2 {
			t.Fatalf("a chunk damaged once was sent %d times, want 2", n)
		}
	})
}

func TestAChunkThatNeverVerifiesFailsDamaged(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		tr := newTransfer(t, w[0], w[1])
		tr.sent.damage = func(position, index, _ int) bool { return position == 1 && index == 0 }
		w[0].start()
		w[1].start()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		start := time.Now()
		f := w[1].h.awaitFailed(t, tr.id)
		if f.GetCode() != tv1.Failed_CODE_DAMAGED || f.GetPosition() != 1 || f.GetDetail() == "read_back" {
			t.Fatalf("a chunk damaged in every copy failed %v", f)
		}
		if waited, backoff := time.Since(start), 7*time.Second; waited < backoff {
			t.Fatalf("the Transfer failed after %v, before the 1 s, 2 s and 4 s backoffs", waited)
		}
		if n := tr.sent.count(1, 0); n != 4 {
			t.Fatalf("the damaged chunk was sent %d times, want 4", n)
		}
	})
}

func TestAStaleRouteLaneIsReset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		tr := newTransfer(t, w[0], w[1])
		lanes := [2]*laneLog{newLaneLog(0), newLaneLog(0)}
		w[0].hooks.Lane, w[1].hooks.Lane = lanes[0].wrap, lanes[1].wrap
		held, reached := make(chan struct{}), make(chan struct{})
		var once sync.Once
		tr.sent.setHold(func(position, index int) {
			if position == 0 && index == 2 {
				once.Do(func() { close(reached) })
				<-held
			}
		})
		w[0].start()
		w[1].start()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		<-reached
		synctest.Wait()
		if moved := w[1].h.moving(tr.id); !slices.Equal(moved, []tv1.Route{tv1.Route_ROUTE_RECEIVER_DIALS}) {
			t.Fatalf("bytes moved on %v before the route changed, want route 1", moved)
		}
		if n := lanes[1].openOn(tv1.Route_ROUTE_RECEIVER_DIALS); n < 2 {
			t.Fatalf("the receiver had %d lanes open on route 1, want several before the route changed", n)
		}
		tr.route(tv1.Route_ROUTE_SENDER_DIALS, 0)
		synctest.Wait()
		for i, l := range lanes {
			if n := l.openOn(tv1.Route_ROUTE_RECEIVER_DIALS); n != 0 {
				t.Fatalf("%s had %d route 1 lanes open after the route changed, want none", w[i].name, n)
			}
		}
		tr.sent.setHold(nil)
		close(held)
		tr.expectArrived()
		if n := tr.sent.count(0, 2); n != 2 {
			t.Fatalf("the chunk held on the route 1 lane was sent %d times, want again on route 2", n)
		}
		if moved := w[1].h.moving(tr.id); !slices.Contains(moved, tv1.Route_ROUTE_SENDER_DIALS) {
			t.Fatalf("bytes moved on %v after the route changed, want route 2", moved)
		}
	})
}

func TestTheSendLimitPacesChunks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		tr := newTransfer(t, w[0], w[1])
		w[0].start()
		w[1].start()
		const limit = 4 << 20
		start := time.Now()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, limit)
		tr.expectArrived()
		total := int64(sizes[0] + sizes[1])
		least := time.Duration(float64(total-limit) / limit * float64(time.Second))
		if took := time.Since(start); took < least || took > least+2*time.Second {
			t.Fatalf("%d bytes at %d a second took %v, want about %v after a one-second burst", total, limit, took, least)
		}
		if !slices.ContainsFunc(w[0].h.facts(), func(f *tv1.Facts) bool { return sending(f, tr.id).GetBinding() }) {
			t.Fatal("the sender's facts never said the send limit held a chunk back")
		}
	})
}

func TestASendLimitTooSlowForAChunkPerLaneWithinTheChunkTimeoutStillArrives(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		tr := newTransfer(t, w[0], w[1])
		w[0].start()
		w[1].start()
		const limit = 64 << 10
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, limit)
		tr.expectArrived()
		for k, n := range tr.sent.counts() {
			if n > 1 {
				t.Fatalf("chunk %d of file %d was sent %d times, want once: a slow lane that keeps moving isn't stalled", k[1], k[0], n)
			}
		}
	})
}

func TestALaneStalledPastTheChunkTimeoutIsRedialed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		tr := newTransfer(t, w[0], w[1])
		tr.sent.setHold(func(position, index int) {
			if position == 0 && index == 1 && tr.sent.count(0, 1) == 1 {
				time.Sleep(w[1].timings.Chunk + 15*time.Second)
			}
		})
		w[0].start()
		w[1].start()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		tr.expectArrived()
		if n := tr.sent.count(0, 1); n != 2 {
			t.Fatalf("the stalled chunk was sent %d times, want twice: once on the stalled lane, once after the redial", n)
		}
	})
}

func TestARouteOneEndCantDialIsGivenUp(t *testing.T) {
	for _, c := range []struct {
		name   string
		refuse func(nw *fault.Network, sender *end)
	}{
		{"the network refuses the dial", func(nw *fault.Network, _ *end) {
			if _, err := nw.Path(linktest.WorkerAddr, linktest.OtherWorkerAddr).Set(fault.Rule{ID: "refuse", Origin: linktest.OtherWorkerAddr.String(), Fault: fault.Refuse{}}); err != nil {
				panic(err)
			}
		}},
		{"the sender doesn't admit the lane", func(_ *fault.Network, sender *end) { sender.lanes.setRefuse(true) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				_, w, nw := trio(t)
				tr := newTransfer(t, w[0], w[1])
				c.refuse(nw, w[0])
				w[0].start()
				w[1].start()
				start := time.Now()
				tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
				w[1].h.await(t, "route 1 given up", func(_ []*tv1.Result, fs []*tv1.Facts) bool {
					return len(fs) > 0 && slices.Contains(receiving(fs[len(fs)-1], tr.id).GetGaveUp(), tv1.Route_ROUTE_RECEIVER_DIALS)
				})
				if waited := time.Since(start); waited < 30*time.Second {
					t.Fatalf("the receiver gave route 1 up after %v, before timings.transfers.dial", waited)
				}
				if w[1].h.lastWait(tr.id) != tv1.Wait_KIND_CONNECTING {
					t.Fatalf("a receiver with no lane waits %s, want connecting", w[1].h.lastWait(tr.id))
				}
				w[0].lanes.setRefuse(false)
				tr.route(tv1.Route_ROUTE_SENDER_DIALS, 0)
				tr.expectArrived()
			})
		})
	}
}

func TestASourceNotReadyWaitsThenGoneFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		tr := newTransfer(t, w[0], w[1])
		ref := source.Torrent{Hash: bencode.InfoHash{1}}
		w[0].sources.put(ref, "", &source.Absent{Reason: source.NotReady, Detail: "the torrent is incomplete"})
		w[0].start()
		w[1].start()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		w[1].h.await(t, "the source isn't ready", func(_ []*tv1.Result, fs []*tv1.Facts) bool {
			return lastWait(fs, tr.id) == tv1.Wait_KIND_SOURCE_NOT_READY
		})
		w[0].sources.put(ref, "", &source.Absent{Reason: source.Gone})
		if f := w[1].h.awaitFailed(t, tr.id); f.GetCode() != tv1.Failed_CODE_SOURCE_GONE || f.GetPosition() != 0 {
			t.Fatalf("a source gone failed %v, want source_gone at file 0", f)
		}
	})
}

func TestADiskThatFillsWaitsAndGoesOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		tr := newTransfer(t, w[0], w[1])
		if _, err := w[1].plan.Set(fault.Rule{ID: "full", Target: fault.Target(w[1].dir), Fault: fault.Full{}}); err != nil {
			t.Fatal(err)
		}
		w[0].start()
		w[1].start()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		w[1].h.await(t, "a full disk", func(_ []*tv1.Result, fs []*tv1.Facts) bool {
			return lastWait(fs, tr.id) == tv1.Wait_KIND_DISK_FULL
		})
		w[1].plan.Clear("full")
		tr.expectArrived()
	})
}

func TestATransferLeavingIntentTakesItsStaging(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		tr := newTransfer(t, w[0], w[1])
		w[0].start()
		w[1].start()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		tr.expectArrived()
		folder := filepath.Dir(w[1].staged(tr.id, 0))
		w[1].h.set(&tv1.Intent{Keep: [][]byte{tr.id.Bytes()}}, role.Live)
		synctest.Wait()
		if _, err := os.Stat(folder); err != nil {
			t.Fatalf("a kept Transfer's staging folder: %v", err)
		}
		w[1].h.set(&tv1.Intent{}, role.Live)
		synctest.Wait()
		if _, err := os.Stat(folder); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("the staging folder of a Transfer the intent dropped: %v", err)
		}
	})
}

func TestALaneFromNoActivePeerIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, _ := trio(t)
		w[1].start()
		w[1].h.set(&tv1.Intent{}, role.Live)
		synctest.Wait()
		if w[1].node.Admit(w[0].ln.Key) {
			t.Fatal("a Node with no active Transfer admitted a lane")
		}
		tr := newTransfer(t, w[0], w[1])
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		synctest.Wait()
		if !w[1].node.Admit(w[0].ln.Key) {
			t.Fatal("a receiver refused its sender's lane")
		}
		w[1].h.set(&tv1.Intent{Active: []*tv1.Active{tr.active(tv1.Side_SIDE_RECEIVE, tv1.Route_ROUTE_RECEIVER_DIALS)}}, role.Unreachable)
		synctest.Wait()
		if w[1].node.Admit(w[0].ln.Key) {
			t.Fatal("a Node out of step with its Controller admitted a lane")
		}
	})
}

func receiving(f *tv1.Facts, id storage.UUID) *tv1.Receiving {
	for _, r := range f.GetReceiving() {
		if bytes.Equal(r.GetTransferId(), id.Bytes()) {
			return r
		}
	}
	return nil
}

func sending(f *tv1.Facts, id storage.UUID) *tv1.Sending {
	for _, s := range f.GetSending() {
		if bytes.Equal(s.GetTransferId(), id.Bytes()) {
			return s
		}
	}
	return nil
}

// holder is the transfers channel's holder half as a Node's supervisor
// runs it: the latest intent and standing, the facts it was given and the
// results it queued.
type holder struct {
	mu       sync.Mutex
	snap     *role.Snapshot[*tv1.Intent]
	changed  chan struct{}
	seen     []*tv1.Facts
	reported []*tv1.Result
	moved    chan struct{}
}

func newHolder() *holder { return &holder{changed: make(chan struct{}), moved: make(chan struct{})} }

func (h *holder) set(i *tv1.Intent, s role.Standing) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.snap = &role.Snapshot[*tv1.Intent]{Intent: i, Standing: s}
	close(h.changed)
	h.changed = make(chan struct{})
}

func (h *holder) Intent(ctx context.Context) iter.Seq[role.Snapshot[*tv1.Intent]] {
	return func(yield func(role.Snapshot[*tv1.Intent]) bool) {
		for {
			h.mu.Lock()
			snap, changed := h.snap, h.changed
			h.mu.Unlock()
			if snap != nil && !yield(*snap) {
				return
			}
			select {
			case <-changed:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (h *holder) SetFacts(f *tv1.Facts) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seen = append(h.seen, proto.CloneOf(f))
	h.notifyLocked()
}

func (h *holder) Report(_ context.Context, _ *storage.Tx[storage.Local], r *tv1.Result) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reported = append(h.reported, r)
	h.notifyLocked()
	return nil
}

func (h *holder) notifyLocked() {
	close(h.moved)
	h.moved = make(chan struct{})
}

func (h *holder) facts() []*tv1.Facts {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.seen)
}

func (h *holder) results() []*tv1.Result {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.reported)
}

// await waits until cond holds of the results and facts, failing after an
// hour of the bubble's time.
func (h *holder) await(t *testing.T, what string, cond func([]*tv1.Result, []*tv1.Facts) bool) {
	t.Helper()
	deadline := time.After(time.Hour)
	for {
		h.mu.Lock()
		ok, moved := cond(h.reported, h.seen), h.moved
		h.mu.Unlock()
		if ok {
			return
		}
		select {
		case <-moved:
		case <-deadline:
			t.Fatalf("waited an hour for %s", what)
		}
	}
}

func (h *holder) awaitFailed(t *testing.T, id storage.UUID) *tv1.Failed {
	t.Helper()
	var failed *tv1.Failed
	h.await(t, "a failure", func(rs []*tv1.Result, _ []*tv1.Facts) bool {
		for _, r := range rs {
			if f := r.GetFailed(); f != nil && bytes.Equal(f.GetTransferId(), id.Bytes()) {
				failed = f
			}
		}
		return failed != nil
	})
	return failed
}

func (h *holder) lastWait(id storage.UUID) tv1.Wait_Kind { return lastWait(h.facts(), id) }

func lastWait(fs []*tv1.Facts, id storage.UUID) tv1.Wait_Kind {
	if len(fs) == 0 {
		return tv1.Wait_KIND_UNSPECIFIED
	}
	return receiving(fs[len(fs)-1], id).GetWait().GetKind()
}

// moving is each route the receiver's facts said bytes moved on.
func (h *holder) moving(id storage.UUID) []tv1.Route {
	var out []tv1.Route
	for _, f := range h.facts() {
		if r := receiving(f, id); r.GetMoving() && !slices.Contains(out, r.GetRoute()) {
			out = append(out, r.GetRoute())
		}
	}
	return out
}

// lateLanes is an end's lane handler, bound to its engine once the link
// it needs is up. refuse refuses every lane, as a peer with no Transfer
// does.
type lateLanes struct {
	mu     sync.Mutex
	n      *transfers.Node
	refuse bool
}

func (l *lateLanes) set(n *transfers.Node) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.n = n
}

func (l *lateLanes) setRefuse(refuse bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refuse = refuse
}

func (l *lateLanes) node() *transfers.Node {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.n
}

// Admit refuses every lane until the end's engine is up, as a Node with no
// Transfer does.
func (l *lateLanes) Admit(key link.Key) bool {
	l.mu.Lock()
	n, refuse := l.n, l.refuse
	l.mu.Unlock()
	return n != nil && !refuse && n.Admit(key)
}

func (l *lateLanes) Serve(ctx context.Context, key link.Key, c net.Conn) { l.node().Serve(ctx, key, c) }

// sources is a sender's torrent client: a path per source, or why it is
// absent.
type sources struct {
	mu     sync.Mutex
	paths  map[source.Ref]string
	absent map[source.Ref]error
}

func (s *sources) put(ref source.Ref, path string, absent error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if absent != nil {
		s.absent[ref] = absent
		return
	}
	s.paths[ref] = path
}

func (s *sources) Open(ref source.Ref) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.absent[ref]; err != nil {
		return "", err
	}
	if p, ok := s.paths[ref]; ok {
		return p, nil
	}
	return "", &source.Absent{Reason: source.Gone}
}

// stager is a receiver's Storage holder: it stages each Transfer under the
// end's folder, for the one root at its one place.
type stager struct{ dir string }

func (s *stager) Staging(_ context.Context, r storage.UUID, p int64, id storage.UUID) (string, error) {
	if r != root || p != place {
		return "", &storing.NotReady{Unknown: true}
	}
	folder := filepath.Join(s.dir, storing.StagingDir, id.String())
	if err := os.MkdirAll(folder, 0o750); err != nil {
		return "", err //nolint:wrapcheck // a test's gate
	}
	return folder, nil
}

// sent counts each chunk the sender hashed, and may hold or damage it.
type sent struct {
	mu     sync.Mutex
	n      map[[2]int]int
	hold   func(position, index int)
	damage func(position, index, nth int) bool
}

func (s *sent) see(_ transfers.ID, position, index int, data []byte) {
	s.mu.Lock()
	s.n[[2]int{position, index}]++
	nth, hold, damage := s.n[[2]int{position, index}], s.hold, s.damage
	s.mu.Unlock()
	if hold != nil {
		hold(position, index)
	}
	if damage != nil && damage(position, index, nth) {
		data[100] ^= 0xff
	}
}

func (s *sent) setHold(hold func(position, index int)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hold = hold
}

func (s *sent) count(position, index int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n[[2]int{position, index}]
}

func (s *sent) counts() map[[2]int]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[[2]int]int{}
	for k, v := range s.n {
		out[k] = v
	}
	return out
}

func sum(t *testing.T, path string) [32]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return blake3.Sum256(b)
}
