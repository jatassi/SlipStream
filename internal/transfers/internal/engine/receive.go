package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/bits"
	"slices"
	"strconv"
	"time"

	"github.com/zeebo/blake3"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers/internal/db"
)

// receiving is the receiver's state. Only its run's receive goroutine
// touches the fields above the facts, which are under run.mu.
type receiving struct {
	inbound chan *lane
	folder  string
	files   []*staged
	buf     []byte
	// batchBytes and batchSince are the chunks written since the last
	// sync.
	batchBytes int64
	batchSince time.Time
	nextDial   time.Time
	tries      int
	failed     bool
	damaged    *failure

	// The facts, under run.mu.
	wait          *tv1.Wait
	moving        tv1.Route
	lanes         int
	rtt           time.Duration
	retried       int
	verifiedBytes int64
	moved         int64
	rate          float64
	rateBytes     int64
	rateAt        time.Time
}

// staged is one file the receiver stages, at <folder>/<position>.part.
type staged struct {
	file
	path string
	// m is the sender's Manifest of the file as this run first opened it.
	m        *tv1.Manifest
	verified bitset
	// written is the chunks written since the last sync, by their hash.
	written  map[int][32]byte
	fails    map[int]*fails
	out      storing.WriteAtFile
	in       storing.ReadAtCloser
	reported bool
}

type fails struct {
	transit, readBack int
	next              time.Time
}

// failure is a Transfer that can't finish: the receiver reports it and
// stops.
type failure struct {
	code     tv1.Failed_Code
	position int
	detail   string
}

func (f *failure) Error() string {
	return fmt.Sprintf("%s at file %d: %s", f.code, f.position, f.detail)
}

// receive runs the receiver until the run ends: it stages, takes a lane on
// its route, and fills its files on it, again after each break.
func (r *run) receive() {
	rx := r.rx
	defer r.closeFiles()
	for r.ctx.Err() == nil {
		if rx.failed || rx.folder != "" && rx.current() == nil {
			r.setWait(tv1.Wait_KIND_UNSPECIFIED, "")
			<-r.ctx.Done()
			return
		}
		if err := r.stage(); err != nil {
			r.setWait(tv1.Wait_KIND_ROOT_NOT_READY, err.Error())
			r.backoff()
			continue
		}
		rx.tries = 0
		if l := r.awaitLane(); l != nil {
			r.carry(l)
		}
	}
}

// carry fills the files on l until it ends, then lets it go. A lane this
// end opened that never heard from the peer is a failed attempt on its
// route.
func (r *run) carry(l *lane) {
	rx := r.rx
	r.note(func() { rx.lanes = 1 })
	heard, err := r.fill(l)
	if ferr := r.flush(); ferr != nil && err == nil {
		err = ferr
	}
	r.note(func() {
		rx.lanes, rx.moving = 0, tv1.Route_ROUTE_UNSPECIFIED
		rx.retried += len(l.asked)
	})
	r.drop(l)
	switch {
	case !l.dialed:
	case heard:
		r.heard(l.route)
	default:
		rx.nextDial = time.Now().Add(r.failed(l.route))
	}
	if f, ok := errors.AsType[*failure](err); ok {
		r.fail(f)
	}
}

// backoff waits out the receiver's next retry of a gate, a source or a
// full disk. The run's end cuts it short, which the caller's loop sees.
func (r *run) backoff() {
	r.rx.tries++
	r.pause(retryDelay(r.rx.tries, r.e.cfg.Timings))
}

// awaitLane is the next lane on the current route: one this end opens, or
// one the sender opened. It is nil once the run ends.
func (r *run) awaitLane() *lane {
	rx := r.rx
	for {
		route := dials(r.spec.side, r.route())
		if route != tv1.Route_ROUTE_UNSPECIFIED && !time.Now().Before(rx.nextDial) {
			l, err := r.open(route)
			if err == nil {
				return l
			}
			rx.nextDial = time.Now().Add(r.failed(route))
		}
		r.setWait(tv1.Wait_KIND_CONNECTING, "")
		if l, ok := r.awaitInbound(route != tv1.Route_ROUTE_UNSPECIFIED); l != nil || !ok {
			return l
		}
	}
}

// awaitInbound waits for a lane the sender opens, until this end may dial
// again when it dials, or the route changes. ok is false once the run
// ended.
func (r *run) awaitInbound(dialing bool) (l *lane, ok bool) {
	var retry <-chan time.Time
	if dialing {
		t := time.NewTimer(time.Until(r.rx.nextDial))
		defer t.Stop()
		retry = t.C
	}
	select {
	case l := <-r.rx.inbound:
		return l, true
	case <-retry:
	case <-r.routeChanged():
	case <-r.ctx.Done():
		return nil, false
	}
	return nil, true
}

// stage passes the Storage holder's gate for the root, and on the first
// pass records the folder and the files reported in an earlier run. A
// folder at another place goes first.
func (r *run) stage() error {
	rx := r.rx
	folder, err := r.e.cfg.Roots.Staging(r.ctx, r.spec.root, r.spec.place, r.spec.id.UUID)
	if err != nil || rx.folder != "" {
		return err
	}
	q := db.New(r.e.cfg.Local)
	old, err := q.Staging(r.ctx, r.spec.id)
	switch {
	case errors.Is(err, storage.ErrNoRows):
	case err != nil:
		return err
	case old != folder:
		if err := r.e.cfg.Disk.RemoveAll(r.ctx, old); err != nil {
			return fmt.Errorf("delete the staging folder at the root's last place: %w", err)
		}
	}
	err = r.e.cfg.Local.Write(r.ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).PutStaging(ctx, db.PutStagingParams{TransferID: r.spec.id, Folder: folder})
	})
	if err != nil {
		return err
	}
	style := r.e.cfg.Disk.Host().Style
	rx.files = nil
	for _, f := range r.spec.files {
		s := &staged{file: f, path: style.Join(folder, strconv.Itoa(f.position)+".part"), written: map[int][32]byte{}, fails: map[int]*fails{}}
		row, err := q.Resume(r.ctx, r.resumeKey(f.position))
		if err != nil && !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		s.reported = err == nil && row.Done == 1
		rx.files = append(rx.files, s)
	}
	rx.folder = folder
	return nil
}

func (r *run) resumeKey(position int) db.ResumeParams {
	return db.ResumeParams{TransferID: r.spec.id, Position: int64(position), Place: r.spec.place}
}

func (rx *receiving) current() *staged {
	for _, f := range rx.files {
		if !f.reported {
			return f
		}
	}
	return nil
}

// fill asks for the files' missing chunks in order on l, keeps each copy
// that matches its hash, and finishes each file once all are verified. It
// returns when l breaks, the run ends, every file is reported, or the
// Transfer can't finish, and reports whether the sender answered at all.
func (r *run) fill(l *lane) (heard bool, err error) {
	for r.ctx.Err() == nil {
		if r.rx.damaged != nil {
			return heard, r.rx.damaged
		}
		f := r.rx.current()
		if f == nil {
			return heard, nil
		}
		stepped, err := r.step(l, f)
		heard = heard || stepped
		if err != nil && !r.full(err) {
			return heard, err
		}
	}
	return heard, nil
}

// step takes f one exchange further on l: it opens f, finishes it, or
// asks for and takes a chunk.
func (r *run) step(l *lane, f *staged) (heard bool, err error) {
	switch {
	case l.open != f.position:
		m, err := r.request(l, f)
		if err != nil {
			return answered(err), r.notReady(err)
		}
		r.rx.tries = 0
		if err := r.adopt(f, m); err != nil {
			return true, err
		}
		l.open = f.position
		return true, nil
	case f.whole():
		err := r.finish(l, f)
		return answered(err), r.notReady(err)
	}
	if err := r.want(l, f); err != nil {
		return false, err
	}
	if len(l.asked) == 0 {
		return false, r.idle(f)
	}
	fr, err := l.read(r.e.cfg.Timings.Chunk)
	if err != nil {
		return false, err
	}
	c := fr.GetChunk()
	at := slices.Index(l.asked, int(c.GetIndex()))
	if c == nil || at < 0 {
		return true, fmt.Errorf("the sender sent %T, not a chunk it was asked for", fr.GetFrame())
	}
	l.asked = slices.Delete(l.asked, at, at+1)
	if err := r.keep(l, f, c); err != nil {
		return true, err
	}
	t := r.e.cfg.Timings
	if r.rx.batchBytes >= t.SyncBytes || time.Since(r.rx.batchSince) >= t.SyncEvery || f.allWritten() {
		return true, r.flush()
	}
	return true, nil
}

// answered reports whether err came after the sender's answer.
func answered(err error) bool {
	_, failed := errors.AsType[*failure](err)
	return err == nil || failed || errors.Is(err, errNotReady)
}

// notReady waits out a source that isn't ready, so the next step opens it
// again. Any other error passes through.
func (r *run) notReady(err error) error {
	if !errors.Is(err, errNotReady) {
		return err
	}
	r.backoff()
	return nil
}

// want asks for chunks of f until l has Ahead of them asked.
func (r *run) want(l *lane, f *staged) error {
	now := time.Now()
	var want []uint32
	for len(l.asked) < r.e.cfg.Timings.Ahead {
		i, ok := f.next(now, l.asked)
		if !ok {
			break
		}
		l.asked = append(l.asked, i)
		want = append(want, uint32(i)) //nolint:gosec // under the file's chunk count
	}
	if len(want) == 0 {
		return nil
	}
	return l.send(&tv1.Frame{Frame: &tv1.Frame_Want{Want: &tv1.Want{Chunks: want}}})
}

// idle is a step with nothing to ask for: it records the chunks written,
// or waits for the first missing chunk's backoff.
func (r *run) idle(f *staged) error {
	if len(f.written) > 0 {
		return r.flush()
	}
	r.pause(time.Until(f.retryAt()))
	return nil
}

var errNotReady = errors.New("the source isn't ready")

// request opens f on l: its Manifest, errNotReady while the sender can't
// read it yet, or a *failure when it never will.
func (r *run) request(l *lane, f *staged) (*tv1.Manifest, error) {
	start := time.Now()
	if err := l.send(&tv1.Frame{Frame: &tv1.Frame_Open{Open: &tv1.Open{Position: uint32(f.position), Source: f.wire}}}); err != nil { //nolint:gosec // a position the intent names
		return nil, err
	}
	fr, err := l.read(r.e.cfg.Timings.Chunk)
	if err != nil {
		return nil, err
	}
	r.note(func() { r.rx.rtt = time.Since(start) })
	switch v := fr.GetFrame().(type) {
	case *tv1.Frame_Manifest:
		m := v.Manifest
		if m.GetChunkSize() != ChunkSize || m.GetSize() < 0 || m.GetIdentity() == nil {
			return nil, fmt.Errorf("the sender's manifest of file %d reads %v", f.position, m)
		}
		r.setWait(tv1.Wait_KIND_UNSPECIFIED, "")
		return m, nil
	case *tv1.Frame_Absent:
		return nil, r.absent(f, v.Absent)
	}
	return nil, fmt.Errorf("the sender responded to an Open with %T", fr.GetFrame())
}

// absent is the error an Absent answer to f's Open makes: a *failure for
// a source gone or failed, errNotReady with the wait set otherwise.
func (r *run) absent(f *staged, a *tv1.Absent) error {
	switch a.GetReason() {
	case tv1.Absent_REASON_GONE:
		return &failure{code: tv1.Failed_CODE_SOURCE_GONE, position: f.position, detail: a.GetDetail()}
	case tv1.Absent_REASON_FAILED:
		return &failure{code: tv1.Failed_CODE_SOURCE_FAILED, position: f.position, detail: a.GetDetail()}
	case tv1.Absent_REASON_UNSPECIFIED, tv1.Absent_REASON_NOT_READY:
	}
	r.setWait(tv1.Wait_KIND_SOURCE_NOT_READY, a.GetDetail())
	return errNotReady
}

// adopt takes m as file f's source. A file this run already opened keeps
// its chunks while the identity holds. On a first open it resumes the
// chunks the resume record verified against the same identity at the same
// place, while the staged file still holds them; anything else restarts
// the file from zero.
func (r *run) adopt(f *staged, m *tv1.Manifest) error {
	if f.m != nil {
		if sameIdentity(f.m.GetIdentity(), m.GetIdentity()) {
			return nil
		}
		return r.restart(f, m)
	}
	row, err := db.New(r.e.cfg.Local).Resume(r.ctx, r.resumeKey(f.position))
	if errors.Is(err, storage.ErrNoRows) {
		return r.restart(f, m)
	}
	if err != nil {
		return err
	}
	verified := bitset(row.Verified)
	resumes := sameIdentity(&tv1.Identity{Size: row.Size, MtimeUnixNs: row.MtimeNs, FileId: row.FileKey}, m.GetIdentity()) &&
		len(verified) == bitsetLen(chunks(m.GetSize())) && r.stagedHolds(f, m.GetSize(), verified)
	if !resumes {
		return r.restart(f, m)
	}
	if err := r.openStaged(f); err != nil {
		return err
	}
	err = r.e.cfg.Local.Write(r.ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).DeleteResumeElsewhere(ctx, db.DeleteResumeElsewhereParams(r.resumeKey(f.position)))
	})
	if err != nil {
		return err
	}
	f.m, f.verified = m, verified
	r.countVerified()
	return nil
}

// stagedHolds reports whether the staged file reaches the end of its
// highest verified chunk.
func (r *run) stagedHolds(f *staged, size int64, verified bitset) bool {
	hi := verified.highest()
	if hi < 0 {
		return true
	}
	info, err := r.e.cfg.Disk.Stat(r.ctx, f.path)
	return err == nil && info.Size() >= offset(hi)+chunkLen(size, hi)
}

// restart truncates f's staged file and records m's identity with no
// chunk verified.
func (r *run) restart(f *staged, m *tv1.Manifest) error {
	if err := r.openStaged(f); err != nil {
		return err
	}
	if err := f.out.Truncate(0); err != nil {
		return fmt.Errorf("truncate the staged file: %w", err)
	}
	verified := make(bitset, bitsetLen(chunks(m.GetSize())))
	id := m.GetIdentity()
	err := r.e.cfg.Local.Write(r.ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		q := db.New(tx)
		if err := q.DeleteResumeElsewhere(ctx, db.DeleteResumeElsewhereParams(r.resumeKey(f.position))); err != nil {
			return err
		}
		return q.PutResume(ctx, db.PutResumeParams{
			TransferID: r.spec.id, Position: int64(f.position), Place: r.spec.place,
			Size: id.GetSize(), MtimeNs: id.GetMtimeUnixNs(), FileKey: id.GetFileId(), Verified: verified,
		})
	})
	if err != nil {
		return err
	}
	f.m, f.verified = m, verified
	clear(f.written)
	clear(f.fails)
	r.countVerified()
	return nil
}

func sameIdentity(a, b *tv1.Identity) bool {
	return a.GetSize() == b.GetSize() && a.GetMtimeUnixNs() == b.GetMtimeUnixNs() && bytes.Equal(a.GetFileId(), b.GetFileId())
}

func (r *run) openStaged(f *staged) error {
	if f.out != nil {
		return nil
	}
	out, err := r.e.cfg.Disk.OpenWrite(r.ctx, f.path)
	if err != nil {
		return fmt.Errorf("open the staged file: %w", err)
	}
	f.out = out
	return nil
}

// keep checks a chunk against the hash the sender took and writes it at
// its offset on a match. A mismatch goes back to the queue.
func (r *run) keep(l *lane, f *staged, c *tv1.Chunk) error {
	i := int(c.GetIndex())
	sum := blake3.Sum256(c.GetData())
	if int64(len(c.GetData())) != chunkLen(f.m.GetSize(), i) || !bytes.Equal(sum[:], c.GetHash()) {
		r.chunkFailed(f, i, false)
		return nil
	}
	if _, err := f.out.WriteAt(c.GetData(), offset(i)); err != nil {
		return fmt.Errorf("write chunk %d: %w", i, err)
	}
	f.written[i] = sum
	rx := r.rx
	if rx.batchBytes == 0 {
		rx.batchSince = time.Now()
	}
	rx.batchBytes += int64(len(c.GetData()))
	r.setWait(tv1.Wait_KIND_UNSPECIFIED, "")
	r.note(func() {
		rx.moved += int64(len(c.GetData()))
		if rx.moving != l.route && holds(r.r, l.route) {
			rx.moving = l.route
			r.e.facts.urgent()
		}
	})
	r.e.facts.progress()
	return nil
}

// flush syncs each file written since the last sync, reads the chunks back
// and hashes them again, and records those that match in one local write.
// A chunk that reads back wrong goes back to the queue.
func (r *run) flush() error {
	rx := r.rx
	var touched []*staged
	for _, f := range rx.files {
		if len(f.written) > 0 {
			touched = append(touched, f)
		}
	}
	if len(touched) == 0 {
		return nil
	}
	for _, f := range touched {
		if err := f.out.Sync(); err != nil {
			return fmt.Errorf("sync the staged file: %w", err)
		}
	}
	for _, f := range touched {
		if err := r.readBack(f); err != nil {
			return err
		}
	}
	err := r.e.cfg.Local.Write(context.WithoutCancel(r.ctx), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		q := db.New(tx)
		for _, f := range touched {
			if err := q.SetVerified(ctx, db.SetVerifiedParams{TransferID: r.spec.id, Position: int64(f.position), Place: r.spec.place, Verified: f.verified}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, f := range touched {
		clear(f.written)
	}
	rx.batchBytes, rx.batchSince = 0, time.Time{}
	r.countVerified()
	r.e.facts.progress()
	return nil
}

// readBack hashes again each chunk of f written since the last sync, as
// the disk holds it: a match is verified, a mismatch goes back to the
// queue.
func (r *run) readBack(f *staged) error {
	if f.in == nil {
		in, err := r.e.cfg.Disk.OpenAt(r.ctx, f.path)
		if err != nil {
			return fmt.Errorf("open the staged file to read it back: %w", err)
		}
		f.in = in
	}
	if r.rx.buf == nil {
		r.rx.buf = make([]byte, ChunkSize)
	}
	for i, want := range f.written {
		b := r.rx.buf[:chunkLen(f.m.GetSize(), i)]
		if _, err := f.in.ReadAt(b, offset(i)); err != nil || blake3.Sum256(b) != want {
			r.chunkFailed(f, i, true)
			continue
		}
		f.verified.set(i)
	}
	return nil
}

// chunkFailed puts chunk i of f back in the queue after a backoff, or
// fails the Transfer once the chunk failed Damaged times.
func (r *run) chunkFailed(f *staged, i int, readBack bool) {
	c := f.fails[i]
	if c == nil {
		c = &fails{}
		f.fails[i] = c
	}
	if readBack {
		c.readBack++
	} else {
		c.transit++
	}
	n := c.transit + c.readBack
	c.next = time.Now().Add(retryDelay(n, r.e.cfg.Timings))
	r.note(func() { r.rx.retried++ })
	if n >= r.e.cfg.Timings.Damaged {
		detail := fmt.Sprintf("chunk %d failed its check %d times", i, n)
		if c.transit == 0 {
			detail = "read_back"
		}
		r.rx.damaged = &failure{code: tv1.Failed_CODE_DAMAGED, position: f.position, detail: detail}
	}
}

// finish opens f once more: a changed identity restarts it. Otherwise it
// syncs and closes the file, then records it whole and queues FileVerified
// in one local write, so placement never finds a torn file or an open
// handle.
func (r *run) finish(l *lane, f *staged) error {
	m, err := r.request(l, f)
	if err != nil {
		return err
	}
	if !sameIdentity(f.m.GetIdentity(), m.GetIdentity()) {
		return r.restart(f, m)
	}
	if err := f.out.Sync(); err != nil {
		return fmt.Errorf("sync the staged file: %w", err)
	}
	f.close()
	err = r.e.cfg.Local.Write(context.WithoutCancel(r.ctx), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		err := db.New(tx).SetVerified(ctx, db.SetVerifiedParams{TransferID: r.spec.id, Position: int64(f.position), Place: r.spec.place, Verified: f.verified, Done: 1})
		if err != nil {
			return err
		}
		return r.h.Report(ctx, tx, &tv1.Result{Result: &tv1.Result_Verified{Verified: &tv1.FileVerified{
			TransferId: r.spec.id.Bytes(), Position: uint32(f.position), Place: r.spec.place, //nolint:gosec // a position the intent names
		}}})
	})
	if err != nil {
		return err
	}
	f.reported = true
	r.countVerified()
	r.e.facts.urgent()
	return nil
}

// fail reports f and stops the receiver: the Controller decides what
// follows.
func (r *run) fail(f *failure) {
	err := r.e.cfg.Local.Write(context.WithoutCancel(r.ctx), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return r.h.Report(ctx, tx, &tv1.Result{Result: &tv1.Result_Failed{Failed: &tv1.Failed{
			TransferId: r.spec.id.Bytes(), Code: f.code, Position: uint32(f.position), Detail: f.detail, //nolint:gosec // a position the intent names
		}}})
	})
	if err != nil {
		r.e.cfg.Log.Error("report a failed Transfer", "transfer", r.spec.id, "err", err)
		return
	}
	r.rx.failed = true
	r.e.facts.urgent()
}

// full reports whether err is a staged write that ran out of space. The
// receiver keeps what it verified, waits with the reason, and tries again
// after a backoff.
func (r *run) full(err error) bool {
	if !noSpace(err) {
		return false
	}
	r.setWait(tv1.Wait_KIND_DISK_FULL, err.Error())
	r.backoff()
	return true
}

func (r *run) closeFiles() {
	if r.rx.folder == "" {
		return
	}
	if err := r.flush(); err != nil && !errors.Is(err, fs.ErrClosed) {
		r.e.cfg.Log.Warn("record a Transfer's last verified chunks", "transfer", r.spec.id, "err", err)
	}
	for _, f := range r.rx.files {
		f.close()
	}
}

func (f *staged) close() {
	if f.out != nil {
		_ = f.out.Close()
		f.out = nil
	}
	if f.in != nil {
		_ = f.in.Close()
		f.in = nil
	}
}

func (f *staged) whole() bool { return f.verified.count() == chunks(f.m.GetSize()) }

func (f *staged) allWritten() bool {
	for i := range chunks(f.m.GetSize()) {
		if _, ok := f.written[i]; !ok && !f.verified.has(i) {
			return false
		}
	}
	return true
}

// next is the first chunk neither verified, written nor asked, whose
// backoff is over.
func (f *staged) next(now time.Time, asked []int) (int, bool) {
	for i := range chunks(f.m.GetSize()) {
		if _, ok := f.written[i]; ok || f.verified.has(i) || slices.Contains(asked, i) {
			continue
		}
		if c := f.fails[i]; c != nil && now.Before(c.next) {
			continue
		}
		return i, true
	}
	return 0, false
}

// retryAt is when the first missing chunk's backoff ends.
func (f *staged) retryAt() time.Time {
	var at time.Time
	for i, c := range f.fails {
		if f.verified.has(i) {
			continue
		}
		if at.IsZero() || c.next.Before(at) {
			at = c.next
		}
	}
	return at
}

func (r *run) countVerified() {
	var n int64
	for _, f := range r.rx.files {
		if f.reported || f.m == nil {
			continue
		}
		for i := range chunks(f.m.GetSize()) {
			if f.verified.has(i) {
				n += chunkLen(f.m.GetSize(), i)
			}
		}
	}
	r.note(func() { r.rx.verifiedBytes = n })
}

// bitset is one bit per chunk, chunk i at bit i%8 of byte i/8.
type bitset []byte

func bitsetLen(n int) int { return (n + 7) / 8 }

func (b bitset) set(i int)      { b[i/8] |= 1 << (i % 8) }
func (b bitset) has(i int) bool { return i/8 < len(b) && b[i/8]&(1<<(i%8)) != 0 }

func (b bitset) count() int {
	n := 0
	for _, x := range b {
		n += bits.OnesCount8(x)
	}
	return n
}

// highest is the highest chunk set, -1 for none.
func (b bitset) highest() int {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			return i*8 + 7 - bits.LeadingZeros8(b[i])
		}
	}
	return -1
}
