// Package engine is a Node's half of Transfers. It sends and receives the
// Transfers its intent makes active, on lanes that grow while they add
// speed, checks each chunk against the hash the sender took as it read
// it, and keeps what a receiver verified across lane breaks and restarts.
package engine

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"maps"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers/internal/db"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// ChunkSize is every chunk's size but a file's last.
const ChunkSize = 4 << 20

type Holder = role.Holder[*tv1.Intent, *tv1.Facts, *tv1.Result]

// ID is a Transfer's.
type ID = db.TransferID

// Opener resolves a source on this Node: a path, or *source.Absent.
type Opener interface {
	Open(ref source.Ref) (string, error)
}

// Stager is the gate of a root destination on this Node: the folder a
// Transfer stages in at place, or *storing.NotReady while the root takes
// no bytes.
type Stager interface {
	Staging(ctx context.Context, root storage.UUID, place int64, transfer storage.UUID) (string, error)
}

// Link opens lanes: TLS lanes for routes 1 and 2, and kind-2 streams on
// the Node link for route 3.
type Link interface {
	DialLane(ctx context.Context, key link.Key, addrs link.Addresses) (net.Conn, error)
	OpenStream(ctx context.Context, key link.Key, kind link.Kind) (link.Stream, error)
}

type Timings struct {
	Chunk      time.Duration
	Report     time.Duration
	Dial       time.Duration
	Estimate   time.Duration
	LanesStart int
	LanesGrow  time.Duration
	// LanesGain is a percentage.
	LanesGain int
	LanesMax  int
	Ahead     int
	SyncBytes int64
	SyncEvery time.Duration
	RetryMin  time.Duration
	RetryMax  time.Duration
	Damaged   int
	Piece     int64
}

type Hooks struct {
	// Read wraps each file a sender reads chunks from.
	Read func(io.ReaderAt) io.ReaderAt
	// Sent sees each chunk after the sender hashed it, and may change it.
	Sent func(transfer ID, position, index int, data []byte)
	// Lane wraps each lane a run holds, to cut, stall or throttle it.
	Lane func(transfer ID, route tv1.Route, c net.Conn) net.Conn
}

type Config struct {
	Local   *storage.DB[storage.Local]
	Disk    storing.Disk
	Link    Link
	Sources Opener
	Roots   Stager
	Timings Timings
	Hooks   Hooks
	Log     *slog.Logger
}

// Engine follows one Node's transfers intent.
type Engine struct {
	cfg   Config
	limit *limiter
	facts *reporter

	mu   sync.Mutex
	live bool
	runs map[ID]*run
	// kept is the last wait of each kept Transfer this Node received and
	// no longer runs.
	kept map[ID]*tv1.Receiving
}

func New(cfg Config) *Engine {
	return &Engine{cfg: cfg, limit: newLimiter(cfg.Timings.Piece), facts: newReporter(), runs: map[ID]*run{}, kept: map[ID]*tv1.Receiving{}}
}

// Run follows h's intent until ctx ends.
func (e *Engine) Run(ctx context.Context, h Holder) error {
	ctx, cancel := context.WithCancel(ctx)
	var reporting sync.WaitGroup
	reporting.Go(func() { e.facts.run(ctx, e.cfg.Timings.Report, func() { h.SetFacts(e.snapshot()) }) })
	for snap := range h.Intent(ctx) {
		e.follow(ctx, h, snap)
	}
	e.stopAll()
	cancel()
	reporting.Wait()
	return nil
}

// follow applies one intent: in step, it starts each active Transfer it
// lacks, stops each that left or changed shape, hands a changed route to
// its run, and deletes the staging of every Transfer in neither list.
// Out of step, it stops every run and keeps what they verified.
func (e *Engine) follow(ctx context.Context, h Holder, snap role.Snapshot[*tv1.Intent]) {
	defer e.facts.urgent()
	if snap.Standing != role.Live {
		e.stopAll()
		return
	}
	e.limit.apply(snap.Intent.GetSendLimit())
	want := map[ID]spec{}
	for _, a := range snap.Intent.GetActive() {
		sp, err := parseActive(a)
		if err != nil {
			e.cfg.Log.Error("parse an active Transfer in the intent", "err", err)
			continue
		}
		want[sp.id] = sp
	}
	keep := map[ID]bool{}
	for _, b := range snap.Intent.GetKeep() {
		if u, err := storage.UUIDFromBytes(b); err == nil {
			keep[ID{UUID: u}] = true
		}
	}

	e.mu.Lock()
	e.live = true
	var stopping []*run
	for id, r := range e.runs {
		sp, ok := want[id]
		switch {
		case !ok || !r.spec.sameRun(sp):
			stopping = append(stopping, r)
			delete(e.runs, id)
		case sp.route != r.route():
			r.setRoute(sp.route)
		}
	}
	e.mu.Unlock()
	e.stop(stopping)

	e.mu.Lock()
	for id, sp := range want {
		if e.runs[id] == nil {
			delete(e.kept, id)
			e.runs[id] = e.start(ctx, h, sp)
		}
	}
	for id := range e.kept {
		if !keep[id] {
			delete(e.kept, id)
		}
	}
	e.mu.Unlock()
	e.collect(ctx, want, keep)
}

func (e *Engine) stopAll() {
	e.mu.Lock()
	e.live = false
	stopping := slices.Collect(maps.Values(e.runs))
	clear(e.runs)
	e.mu.Unlock()
	e.stop(stopping)
}

// stop ends each run and waits for it, keeping a receiver's last wait.
func (e *Engine) stop(runs []*run) {
	for _, r := range runs {
		r.cancel()
	}
	for _, r := range runs {
		r.wg.Wait()
		if f := r.keptFact(); f != nil {
			e.mu.Lock()
			e.kept[r.spec.id] = f
			e.mu.Unlock()
		}
	}
}

// collect deletes the staging folder and resume records of each Transfer
// the intent neither runs nor keeps. A folder that won't go yet keeps its
// record, so the next intent tries again.
func (e *Engine) collect(ctx context.Context, active map[ID]spec, keep map[ID]bool) {
	rows, err := db.New(e.cfg.Local).ListStaging(ctx)
	if err != nil {
		e.cfg.Log.Error("list staged Transfers", "err", err)
		return
	}
	for _, row := range rows {
		if _, ok := active[row.TransferID]; ok || keep[row.TransferID] {
			continue
		}
		if err := e.unstage(ctx, row.TransferID, row.Folder); err != nil {
			e.cfg.Log.Warn("delete a Transfer's staging folder", "transfer", row.TransferID, "folder", row.Folder, "err", err)
		}
	}
}

// unstage removes folder before its record, so a crash between leaves a
// record the next pass finds rather than a folder nothing names.
func (e *Engine) unstage(ctx context.Context, id ID, folder string) error {
	if err := e.cfg.Disk.RemoveAll(ctx, folder); err != nil {
		return err
	}
	return e.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).DeleteStaging(ctx, id)
	})
}

// Release deletes every staging folder and resume record: the Node was
// removed.
func (e *Engine) Release(ctx context.Context) error {
	rows, err := db.New(e.cfg.Local).ListStaging(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := e.unstage(ctx, row.TransferID, row.Folder); err != nil {
			return err
		}
	}
	return nil
}

// Admit admits a lane from key while this Node is in step and runs a
// Transfer with key as its peer. Admission is by peer, not route, so a
// route flip needs no ordering between the two Nodes' intents.
func (e *Engine) Admit(key link.Key) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.live {
		return false
	}
	for _, r := range e.runs {
		if r.spec.peer == key {
			return true
		}
	}
	return false
}

// Serve takes a lane key opened: it reads the lane's Lane frame and hands
// it to that Transfer's run, or closes it unless the Transfer is active
// with key on a route that takes it.
func (e *Engine) Serve(ctx context.Context, key link.Key, c net.Conn) {
	e.accept(ctx, key, c, false)
}

// ServeStream is route 3: a kind-2 stream on the Node link, taken as Serve
// takes a lane.
func (e *Engine) ServeStream(key link.Key, s link.Stream) {
	e.accept(context.Background(), key, s, true)
}

func (e *Engine) accept(ctx context.Context, key link.Key, c net.Conn, stream bool) {
	defer func() { _ = c.Close() }()
	l := newLane(c, tv1.Route_ROUTE_UNSPECIFIED, e.cfg.Timings.Chunk)
	fr, err := l.read(e.cfg.Timings.Chunk)
	if err != nil {
		return
	}
	lf := fr.GetLane()
	u, err := storage.UUIDFromBytes(lf.GetTransferId())
	if err != nil || (lf.GetRoute() == tv1.Route_ROUTE_NODE_LINK) != stream {
		return
	}
	l.route = lf.GetRoute()
	e.mu.Lock()
	r := e.runs[ID{UUID: u}]
	e.mu.Unlock()
	if r == nil || r.spec.peer != key {
		return
	}
	r.take(ctx, l)
}

func (e *Engine) start(ctx context.Context, h Holder, sp spec) *run {
	r := newRun(e, h, sp)
	r.ctx, r.cancel = context.WithCancel(ctx)
	if sp.side == tv1.Side_SIDE_RECEIVE {
		r.rx = &receiving{inbound: make(chan *lane), changed: make(chan struct{})}
		r.wg.Go(r.receive)
	} else {
		r.wg.Go(func() {
			r.runLanes(laneEnd{serve: r.serveSends})
		})
	}
	return r
}

// snapshot is this Node's facts now.
func (e *Engine) snapshot() *tv1.Facts {
	now := time.Now()
	e.mu.Lock()
	runs := slices.Collect(maps.Values(e.runs))
	f := &tv1.Facts{}
	for _, k := range e.kept {
		f.Receiving = append(f.Receiving, k)
	}
	e.mu.Unlock()
	for _, r := range runs {
		switch rf := r.fact(now).(type) {
		case *tv1.Receiving:
			f.Receiving = append(f.Receiving, rf)
		case *tv1.Sending:
			f.Sending = append(f.Sending, rf)
		}
	}
	slices.SortFunc(f.Receiving, func(a, b *tv1.Receiving) int { return bytes.Compare(a.GetTransferId(), b.GetTransferId()) })
	slices.SortFunc(f.Sending, func(a, b *tv1.Sending) int { return bytes.Compare(a.GetTransferId(), b.GetTransferId()) })
	return f
}
