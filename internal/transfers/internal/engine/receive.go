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
	"sync"
	"time"

	"github.com/zeebo/blake3"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers/internal/db"
)

// receiving is the receiver's state. mu guards the files and the queue,
// which every lane's carrier shares. The facts are under run.mu, taken
// after mu.
type receiving struct {
	inbound chan *lane

	mu sync.Mutex
	// changed closes when a lane may find work it didn't before.
	changed chan struct{}
	folder  string
	files   []*staged
	buf     []byte
	// batchBytes and batchSince are the chunks written since the last
	// sync.
	batchBytes int64
	batchSince time.Time
	tries      int
	failed     bool
	damaged    *failure

	// The facts, under run.mu.
	wait          *tv1.Wait
	moving        tv1.Route
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
	// kept is the bytes the resume record holds verified until the run
	// takes m and decides whether they resume.
	kept int64
	// written is the chunks written since the last sync, by their hash.
	written map[int][32]byte
	// asked is the chunks some lane asked for and hasn't taken.
	asked map[int]bool
	fails map[int]*fails
	// gen counts restarts. A lane holds the file open at the gen it
	// opened it, and takes no chunk of another.
	gen       int
	finishing bool
	out       storing.WriteAtFile
	in        storing.ReadAtCloser
	reported  bool
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

// receive runs the receiver's lanes until the run ends, each filling the
// files from the one queue of missing chunks.
func (r *run) receive() {
	defer r.closeFiles()
	r.runLanes(laneEnd{inbound: r.rx.inbound, ready: r.ready, serve: r.carry, grows: true})
}

// ready gates a first lane on the Storage holder's gate for the root,
// with a backoff while it fails. A receiver whose files are all reported,
// or that failed, takes no lane.
func (r *run) ready() (time.Duration, bool) {
	rx := r.rx
	rx.mu.Lock()
	defer rx.mu.Unlock()
	if rx.failed || rx.folder != "" && rx.current() == nil {
		r.setWait(tv1.Wait_KIND_UNSPECIFIED, "")
		return 0, false
	}
	if err := r.stage(); err != nil {
		r.setWait(tv1.Wait_KIND_ROOT_NOT_READY, err.Error())
		rx.tries++
		return retryDelay(rx.tries, r.e.cfg.Timings), false
	}
	rx.tries = 0
	r.setWait(tv1.Wait_KIND_CONNECTING, "")
	return 0, true
}

// carry fills the files on l until it ends, puts the chunks it still had
// asked back in the queue, and reports whether the sender answered on it.
func (r *run) carry(l *lane) (heard bool) {
	rx := r.rx
	heard, err := r.fill(l)
	rx.mu.Lock()
	if ferr := r.flush(); ferr != nil && err == nil {
		err = ferr
	}
	if f := rx.file(l.open); f != nil && l.gen == f.gen {
		for _, i := range l.asked {
			delete(f.asked, i)
		}
	}
	r.note(func() { rx.retried += len(l.asked) })
	l.asked = nil
	rx.poke()
	rx.mu.Unlock()
	if f, ok := errors.AsType[*failure](err); ok {
		r.fail(f)
	}
	return heard
}

// retire lets l go once the chunks it asked for arrive. Its last Want
// tells a sender that dials how many lanes to keep.
func (r *run) retire(l *lane) {
	l.retiring.Store(true)
	r.rx.mu.Lock()
	r.rx.poke()
	r.rx.mu.Unlock()
}

// poke wakes every idle lane. Call it with mu held.
func (rx *receiving) poke() {
	close(rx.changed)
	rx.changed = make(chan struct{})
}

// backoff waits out the receiver's next retry of a source or a full disk,
// with mu let go. The run's end cuts it short, which the caller's loop
// sees.
func (r *run) backoff() {
	r.rx.tries++
	d := retryDelay(r.rx.tries, r.e.cfg.Timings)
	r.rx.mu.Unlock()
	defer r.rx.mu.Lock()
	r.pause(d)
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
		s := &staged{
			file: f, path: style.Join(folder, strconv.Itoa(f.position)+".part"),
			written: map[int][32]byte{}, asked: map[int]bool{}, fails: map[int]*fails{},
		}
		row, err := q.Resume(r.ctx, r.resumeKey(f.position))
		if err != nil && !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		if err == nil {
			s.reported = row.Done == 1
			s.kept = bitset(row.Verified).bytes(row.Size)
		}
		rx.files = append(rx.files, s)
	}
	rx.folder = folder
	r.countVerified()
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

func (rx *receiving) file(position int) *staged {
	for _, f := range rx.files {
		if f.position == position {
			return f
		}
	}
	return nil
}

// pick is the file l takes further next: the one it has chunks asked of,
// else the first not yet reported that is still to open, has a chunk to
// ask for, or is whole and next to finish. Lanes move on to the next file
// while another's last chunks are in flight, and files still finish in
// order.
func (rx *receiving) pick(l *lane, now time.Time) *staged {
	if len(l.asked) > 0 {
		return rx.file(l.open)
	}
	for _, f := range rx.files {
		switch {
		case f.reported || f.finishing:
			continue
		case f.m == nil, f.whole() && f == rx.current():
			return f
		case f.whole():
			continue
		}
		if _, ok := f.next(now); ok {
			return f
		}
	}
	return nil
}

// fill takes the files one exchange at a time on l, beside the other
// lanes, until l breaks or retires, the run ends, every file is reported,
// or the Transfer can't finish. It reports whether the sender answered at
// all.
func (r *run) fill(l *lane) (heard bool, err error) {
	rx := r.rx
	rx.mu.Lock()
	defer rx.mu.Unlock()
	for r.ctx.Err() == nil {
		switch {
		case rx.damaged != nil:
			return heard, rx.damaged
		case rx.current() == nil:
			return heard, nil
		case l.retiring.Load() && len(l.asked) == 0:
			return heard, r.wantOn(l, nil)
		}
		f := rx.pick(l, time.Now())
		if f == nil {
			if err := r.idle(); err != nil {
				return heard, err
			}
			continue
		}
		stepped, err := r.step(l, f)
		heard = heard || stepped
		if err != nil && !r.full(err) {
			return heard, err
		}
	}
	return heard, nil
}

// step takes f one exchange further on l: it finishes f, opens it, or
// asks for and takes a chunk. mu is let go while l waits on the sender.
func (r *run) step(l *lane, f *staged) (heard bool, err error) {
	if len(l.asked) == 0 {
		switch {
		case f.whole():
			err := r.finish(l, f)
			return answered(err), r.notReady(err)
		case l.open != f.position || l.gen != f.gen:
			return r.openFile(l, f)
		}
	}
	if err := r.want(l, f); err != nil {
		return false, err
	}
	if len(l.asked) == 0 {
		return false, nil
	}
	var fr *tv1.Frame
	err = r.rx.unlocked(func() (err error) {
		fr, err = l.read(r.e.cfg.Timings.Chunk)
		return err
	})
	if err != nil {
		return false, err
	}
	c := fr.GetChunk()
	at := slices.Index(l.asked, int(c.GetIndex()))
	if c == nil || at < 0 {
		return true, fmt.Errorf("the sender sent %T, not a chunk it was asked for", fr.GetFrame())
	}
	l.asked = slices.Delete(l.asked, at, at+1)
	if l.gen != f.gen {
		return true, nil
	}
	delete(f.asked, int(c.GetIndex()))
	if err := r.keep(f, c); err != nil {
		return true, err
	}
	t := r.e.cfg.Timings
	if r.rx.batchBytes >= t.SyncBytes || time.Since(r.rx.batchSince) >= t.SyncEvery || f.allWritten() {
		return true, r.flush()
	}
	return true, nil
}

// openFile opens f on l and takes its Manifest as f's source.
func (r *run) openFile(l *lane, f *staged) (heard bool, err error) {
	m, err := r.request(l, f)
	if err != nil {
		return answered(err), r.notReady(err)
	}
	r.rx.tries = 0
	if !f.reported {
		if err := r.adopt(f, m); err != nil {
			return true, err
		}
	}
	l.open, l.gen = f.position, f.gen
	return true, nil
}

// unlocked runs io with mu let go.
func (rx *receiving) unlocked(io func() error) error {
	rx.mu.Unlock()
	defer rx.mu.Lock()
	return io()
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

// want asks for chunks of f until l has Ahead of them asked, while l
// holds f open at its current gen and isn't retiring.
func (r *run) want(l *lane, f *staged) error {
	if l.open != f.position || l.gen != f.gen || l.retiring.Load() {
		return nil
	}
	now := time.Now()
	var want []uint32
	for len(l.asked) < r.e.cfg.Timings.Ahead {
		i, ok := f.next(now)
		if !ok {
			break
		}
		l.asked = append(l.asked, i)
		f.asked[i] = true
		want = append(want, uint32(i)) //nolint:gosec // under the file's chunk count
	}
	if len(want) == 0 {
		return nil
	}
	return r.wantOn(l, want)
}

// wantOn sends a Want of chunks on l, with mu let go, and the lanes the
// receiver wants now.
func (r *run) wantOn(l *lane, chunks []uint32) error {
	w := &tv1.Want{Chunks: chunks, Lanes: uint32(r.lanesWanted.Load())} //nolint:gosec // at most LanesMax
	return r.rx.unlocked(func() error { return l.send(&tv1.Frame{Frame: &tv1.Frame_Want{Want: w}}) })
}

// idle is a lane with nothing to take: it records the chunks written, or
// waits with mu let go until a lane may find work or the first backoff
// ends.
func (r *run) idle() error {
	rx := r.rx
	if rx.batchBytes > 0 {
		return r.flush()
	}
	var retry <-chan time.Time
	if at := rx.retryAt(time.Now()); !at.IsZero() {
		retry = after(at)
	}
	changed := rx.changed
	rx.mu.Unlock()
	defer rx.mu.Lock()
	select {
	case <-changed:
	case <-retry:
	case <-r.ctx.Done():
	}
	return nil
}

var errNotReady = errors.New("the source isn't ready")

// request opens f on l, with mu let go while it waits: its Manifest,
// errNotReady while the sender can't read it yet, or a *failure when it
// never will.
func (r *run) request(l *lane, f *staged) (*tv1.Manifest, error) {
	start := time.Now()
	open := &tv1.Frame{Frame: &tv1.Frame_Open{Open: &tv1.Open{Position: uint32(f.position), Source: f.wire}}} //nolint:gosec // a position the intent names
	var fr *tv1.Frame
	err := r.rx.unlocked(func() (err error) {
		if err := l.send(open); err != nil {
			return err
		}
		fr, err = l.read(r.e.cfg.Timings.Chunk)
		return err
	})
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
		r.movesOn(l)
		return m, nil
	case *tv1.Frame_Absent:
		return nil, r.absent(f, v.Absent)
	}
	return nil, fmt.Errorf("the sender responded to an Open with %T", fr.GetFrame())
}

// movesOn marks bytes moving on l's route while the intent's route holds
// it: the sender answered on l, so its chunks follow however slowly.
func (r *run) movesOn(l *lane) {
	r.note(func() {
		if r.rx.moving != l.route && holds(r.r, l.route) {
			r.rx.moving = l.route
			r.e.facts.urgent()
		}
	})
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
	f.gen++
	clear(f.written)
	clear(f.asked)
	clear(f.fails)
	r.countVerified()
	r.rx.poke()
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
func (r *run) keep(f *staged, c *tv1.Chunk) error {
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
		r.win.binding = r.win.binding || c.GetBinding()
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
	rx.poke()
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
	r.rx.poke()
	if n >= r.e.cfg.Timings.Damaged {
		detail := fmt.Sprintf("chunk %d failed its check %d times", i, n)
		if c.transit == 0 {
			detail = "read_back"
		}
		r.rx.damaged = &failure{code: tv1.Failed_CODE_DAMAGED, position: f.position, detail: detail}
	}
}

// finish opens f once more on l, the one lane that finishes it: a changed
// identity restarts it. Otherwise it syncs and closes the file, then
// records it whole and queues FileVerified in one local write, so
// placement never finds a torn file or an open handle.
func (r *run) finish(l *lane, f *staged) error {
	f.finishing = true
	defer func() {
		f.finishing = false
		r.rx.poke()
	}()
	gen := f.gen
	m, err := r.request(l, f)
	if err != nil {
		return err
	}
	if f.gen != gen || !sameIdentity(f.m.GetIdentity(), m.GetIdentity()) {
		err := r.adopt(f, m)
		l.open, l.gen = f.position, f.gen
		return err
	}
	l.open, l.gen = f.position, f.gen
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

// fail reports f once and stops the receiver: the Controller decides what
// follows.
func (r *run) fail(f *failure) {
	r.rx.mu.Lock()
	defer r.rx.mu.Unlock()
	if r.rx.failed {
		return
	}
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
	r.rx.mu.Lock()
	defer r.rx.mu.Unlock()
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

// next is the first chunk neither verified, written nor asked on any
// lane, whose backoff is over.
func (f *staged) next(now time.Time) (int, bool) {
	for i := range chunks(f.m.GetSize()) {
		if c := f.fails[i]; f.missing(i) && (c == nil || !now.Before(c.next)) {
			return i, true
		}
	}
	return 0, false
}

func (f *staged) missing(i int) bool {
	_, written := f.written[i]
	return !written && !f.verified.has(i) && !f.asked[i]
}

// retryAt is when the first missing chunk's backoff ends, zero when no
// missing chunk waits one out.
func (rx *receiving) retryAt(now time.Time) time.Time {
	var at time.Time
	for _, f := range rx.files {
		for i, c := range f.fails {
			if f.reported || !f.missing(i) || !now.Before(c.next) {
				continue
			}
			if at.IsZero() || c.next.Before(at) {
				at = c.next
			}
		}
	}
	return at
}

func (r *run) countVerified() {
	var n int64
	for _, f := range r.rx.files {
		switch {
		case f.reported:
		case f.m == nil:
			n += f.kept
		default:
			n += f.verified.bytes(f.m.GetSize())
		}
	}
	r.note(func() { r.rx.verifiedBytes = n })
}

// bitset is one bit per chunk, chunk i at bit i%8 of byte i/8.
type bitset []byte

func bitsetLen(n int) int { return (n + 7) / 8 }

func (b bitset) set(i int)      { b[i/8] |= 1 << (i % 8) }
func (b bitset) has(i int) bool { return i/8 < len(b) && b[i/8]&(1<<(i%8)) != 0 }

// bytes is what the set chunks of a file of size bytes hold.
func (b bitset) bytes(size int64) int64 {
	var n int64
	for i := range chunks(size) {
		if b.has(i) {
			n += chunkLen(size, i)
		}
	}
	return n
}

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
