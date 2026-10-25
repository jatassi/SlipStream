package transfers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	transferv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/db"
)

// Job is one file to bring from Source into Staging.
type Job struct {
	ID        ID
	Source    role.NodeID
	InfoHash  bencode.InfoHash
	FileIndex int
	// Staging is <root>/.slipstream-staging/<ID>.part, on the root's
	// filesystem so placement is a rename.
	Staging string
}

// Keys finds a Node's link key: the roster.
type Keys interface {
	Key(node role.NodeID) (link.Key, bool)
}

type Config struct {
	Local   *storage.DB[file]
	Link    *link.Link
	Keys    Keys
	Timings Timings
}

// reasks is how many more copies of a chunk the receiver asks for after a
// copy that fails its hash.
const reasks = 3

type Receiver struct {
	cfg Config

	mu      sync.Mutex
	running map[ID]progress
}

type progress struct{ verified, size int64 }

func NewReceiver(cfg Config) *Receiver { return &Receiver{cfg: cfg, running: map[ID]progress{}} }

// Receive fills j.Staging with the source's file, verified chunk by chunk,
// resuming from the chunks local.db records. It returns nil once every
// chunk is verified, at once when they already are. A source that holds
// no live, matched session fails before sending as rpc.NotSent reads it,
// a stream that breaks with ErrBroken, a source without the file with
// ErrNotPresent, and a Transfer that can't finish with *FailedError.
func (r *Receiver) Receive(ctx context.Context, j Job) error {
	rec, err := r.record(ctx, j)
	if err != nil {
		return err
	}
	if rec.done(j.Staging) {
		return nil
	}
	key, ok := r.cfg.Keys.Key(j.Source)
	if !ok {
		return fmt.Errorf("transfers: node %s is not in the roster: %w", j.Source, link.ErrDisconnected)
	}
	st, err := r.cfg.Link.OpenStream(ctx, key, link.KindTransfer)
	if err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { _ = st.Reset() })
	defer stop()
	if err := r.receive(ctx, j, rec, newFrames(st)); err != nil {
		_ = st.Reset()
		return err
	}
	_ = st.Close()
	return nil
}

// record is a Transfer's resume record: the manifest, once known, and the
// chunks verified against it.
type record struct {
	manifest *Manifest
	verified map[int]bool
}

func (rec record) done(staging string) bool {
	if rec.manifest == nil || len(rec.verified) < len(rec.manifest.Hashes) {
		return false
	}
	info, err := os.Stat(staging)
	return err == nil && info.Size() == rec.manifest.Size
}

func (r *Receiver) record(ctx context.Context, j Job) (record, error) {
	var rec record
	err := r.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		err := q.InsertTransfer(ctx, db.InsertTransferParams{
			ID: j.ID, SourceNode: j.Source, InfoHash: j.InfoHash[:], FileIndex: int64(j.FileIndex),
			StagingPath: j.Staging, CreatedAt: storage.Now(),
		})
		if err != nil {
			return err
		}
		row, err := q.Transfer(ctx, j.ID)
		if err != nil {
			return err
		}
		idxs, err := q.Chunks(ctx, j.ID)
		if err != nil {
			return err
		}
		rec = record{manifest: storedManifest(row), verified: map[int]bool{}}
		for _, i := range idxs {
			rec.verified[int(i)] = true
		}
		return nil
	})
	return rec, err
}

func storedManifest(row db.TransfersTransfer) *Manifest {
	if row.Size == nil {
		return nil
	}
	m := &Manifest{Size: *row.Size, Hashes: make([]Hash, len(row.ChunkHashes)/len(Hash{}))}
	for i := range m.Hashes {
		m.Hashes[i] = Hash(row.ChunkHashes[i*len(Hash{}):])
	}
	return m
}

func (r *Receiver) receive(ctx context.Context, j Job, rec record, f *frames) error {
	m, err := r.open(f, j)
	if err != nil {
		return err
	}
	verified, err := r.adopt(ctx, j, rec, m)
	if err != nil {
		return err
	}
	out, err := openStaging(j.Staging, m.Size)
	if err != nil {
		return err
	}
	r.track(j.ID, m, verified)
	defer r.untrack(j.ID)
	if err := r.fill(ctx, j.ID, f, out, m, verified); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("transfers: close the staging file: %w", err)
	}
	return nil
}

// open sends Open and reads the source's Manifest.
func (r *Receiver) open(f *frames, j Job) (Manifest, error) {
	if err := f.send(openFrame(j)); err != nil {
		return Manifest{}, broken(err)
	}
	fr, err := r.next(f)
	if err != nil {
		return Manifest{}, err
	}
	switch v := fr.GetFrame().(type) {
	case *transferv1.Frame_Manifest:
		return parseManifest(v.Manifest)
	case *transferv1.Frame_Absent:
		return Manifest{}, ErrNotPresent
	}
	return Manifest{}, &FailedError{Reason: fmt.Sprintf("the source responded to Open with %T", fr.GetFrame())}
}

// adopt is the chunks already verified against m. A manifest that changed,
// or a staging file that's gone, restarts the file.
func (r *Receiver) adopt(ctx context.Context, j Job, rec record, m Manifest) (map[int]bool, error) {
	_, err := os.Stat(j.Staging)
	gone := errors.Is(err, fs.ErrNotExist)
	if err != nil && !gone {
		return nil, fmt.Errorf("transfers: read the staging file: %w", err)
	}
	if rec.manifest != nil && rec.manifest.equal(m) && !gone {
		return rec.verified, nil
	}
	hashes := make([]byte, 0, len(m.Hashes)*len(Hash{}))
	for _, h := range m.Hashes {
		hashes = append(hashes, h[:]...)
	}
	size, chunkSize := m.Size, int64(ChunkSize)
	err = r.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if err := q.SetManifest(ctx, db.SetManifestParams{ID: j.ID, Size: &size, ChunkSize: &chunkSize, ChunkHashes: hashes}); err != nil {
			return err
		}
		return q.DeleteChunks(ctx, j.ID)
	})
	return map[int]bool{}, err
}

func openStaging(path string, size int64) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // inside a library root, which media servers read
		return nil, fmt.Errorf("transfers: make the staging folder: %w", err)
	}
	out, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644) //nolint:gosec // the pipeline's staging path, inside a library root that media servers read
	if err != nil {
		return nil, fmt.Errorf("transfers: open the staging file: %w", err)
	}
	if err := out.Truncate(size); err != nil {
		_ = out.Close()
		return nil, fmt.Errorf("transfers: size the staging file: %w", err)
	}
	return out, nil
}

// fill asks for every chunk not verified and keeps each copy that matches
// its hash, asking again for one that doesn't.
func (r *Receiver) fill(ctx context.Context, id ID, f *frames, out *os.File, m Manifest, verified map[int]bool) error {
	want := plan(len(m.Hashes), verified)
	if len(want) == 0 {
		return nil
	}
	if err := f.send(wantFrame(want)); err != nil {
		return broken(err)
	}
	bad := map[int]int{}
	for len(verified) < len(m.Hashes) {
		i, data, err := r.chunk(f, m)
		if err != nil {
			return err
		}
		if verified[i] {
			continue
		}
		if blake3.Sum256(data) != m.Hashes[i] {
			bad[i]++
			if bad[i] > reasks {
				return &FailedError{Reason: fmt.Sprintf("chunk %d failed verification %d times", i, bad[i])}
			}
			if err := f.send(wantFrame([]int{i})); err != nil {
				return broken(err)
			}
			continue
		}
		if err := r.keep(ctx, id, out, m.offset(i), i, data); err != nil {
			return err
		}
		verified[i] = true
		r.verify(id, m.chunkLen(i))
	}
	return nil
}

func (r *Receiver) chunk(f *frames, m Manifest) (i int, data []byte, err error) {
	fr, err := r.next(f)
	if err != nil {
		return 0, nil, err
	}
	c := fr.GetChunk()
	switch {
	case c == nil:
		return 0, nil, &FailedError{Reason: fmt.Sprintf("the source sent %T, not a chunk", fr.GetFrame())}
	case int(c.GetIndex()) >= len(m.Hashes):
		return 0, nil, &FailedError{Reason: fmt.Sprintf("the source sent chunk %d of %d", c.GetIndex(), len(m.Hashes))}
	}
	return int(c.GetIndex()), c.GetData(), nil
}

// keep writes a verified chunk and makes it durable before recording it,
// so the record never names a chunk the file lacks.
func (r *Receiver) keep(ctx context.Context, id ID, out *os.File, offset int64, i int, data []byte) error {
	if _, err := out.WriteAt(data, offset); err != nil {
		return fmt.Errorf("transfers: write chunk %d: %w", i, err)
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("transfers: sync chunk %d: %w", i, err)
	}
	return r.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).InsertChunk(ctx, db.InsertChunkParams{TransferID: id, Idx: int64(i)})
	})
}

// next reads the source's next frame, waiting at most Timings.Chunk.
func (r *Receiver) next(f *frames) (*transferv1.Frame, error) {
	if err := f.s.SetReadDeadline(time.Now().Add(r.cfg.Timings.Chunk)); err != nil {
		return nil, broken(err)
	}
	fr, err := f.read()
	if failed, ok := errors.AsType[*FailedError](err); ok {
		return nil, failed
	}
	if err != nil {
		return nil, broken(err)
	}
	return fr, nil
}

func broken(err error) error { return fmt.Errorf("%w: %w", ErrBroken, err) }

func (r *Receiver) track(id ID, m Manifest, verified map[int]bool) {
	p := progress{size: m.Size}
	for i := range verified {
		p.verified += m.chunkLen(i)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running[id] = p
}

func (r *Receiver) verify(id ID, n int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.running[id]
	p.verified += n
	r.running[id] = p
}

func (r *Receiver) untrack(id ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.running, id)
}

// Progress is a running Transfer's verified bytes and size, false when
// none runs for id.
func (r *Receiver) Progress(id ID) (verified, size int64, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.running[id]
	return p.verified, p.size, ok
}

// Forget drops id's record and its staging file, if any is left.
func (r *Receiver) Forget(ctx context.Context, id ID) error {
	row, err := db.New(r.cfg.Local).Transfer(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.drop(ctx, id, row.StagingPath)
}

// Retain drops every record not in keep, and its staging file.
func (r *Receiver) Retain(ctx context.Context, keep []ID) error {
	rows, err := db.New(r.cfg.Local).ListTransfers(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if slices.Contains(keep, row.ID) {
			continue
		}
		if err := r.drop(ctx, row.ID, row.StagingPath); err != nil {
			return err
		}
	}
	return nil
}

// drop removes the staging file before the record, so a crash between
// leaves a record Retain finds rather than a file nothing names.
func (r *Receiver) drop(ctx context.Context, id ID, staging string) error {
	if err := os.Remove(staging); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("transfers: remove the staging file: %w", err)
	}
	return r.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).DeleteTransfer(ctx, id)
	})
}
