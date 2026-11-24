package storing

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/media/speech"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing/internal/db"
)

const mapExt = ".ssmp"

// builds are the map builds this Node runs and the ones that wait, one per
// file. It never runs more than the newest call's slots; the rest wait in
// rank order, then call order.
type builds struct {
	ctx context.Context
	s   *Sidecars

	mu      sync.Mutex
	slots   int
	running int
	calls   uint64
	jobs    map[storage.UUID]*build
	waiting []*build
}

type build struct {
	file, root storage.UUID
	path       string
	stream     int
	identity   Identity
	rank       int
	call       uint64
	started    bool
	done       chan struct{}
	answer     *storagev1.BuildSpeechMapResponse
}

func newBuilds(ctx context.Context, s *Sidecars) *builds {
	return &builds{ctx: ctx, s: s, jobs: map[storage.UUID]*build{}}
}

func (s *Sidecars) BuildSpeechMap(ctx context.Context, req *storagev1.BuildSpeechMapRequest) (*storagev1.BuildSpeechMapResponse, error) {
	file, err := fileID(req.GetFileId())
	if err != nil {
		return nil, err
	}
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	rel, err := relOf(req.GetPath())
	if err != nil {
		return nil, err
	}
	if req.GetSlots() < 1 {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "a Node runs at least one build at once, not %d", req.GetSlots())
	}
	dir, err := s.store.resolve(ctx, root, false)
	if err != nil {
		return nil, WireError(err)
	}
	info, err := s.store.stat(ctx, dir, rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, WireError(s.store.folderError(err))
	}
	if err != nil {
		return nil, WireError(err)
	}
	id := identityOfInfo(info)
	row, ok, err := s.mapRow(ctx, file)
	if err != nil {
		return nil, err
	}
	if ok && current(row, id) {
		return builtAnswer(row, id), nil
	}
	b, err := s.builds.join(&build{
		file: file, root: root, path: s.store.host.Style.Under(dir, rel), stream: int(req.GetStream()), identity: id, rank: int(req.GetRank()),
	}, int(req.GetSlots()))
	if err != nil {
		return nil, err
	}
	wait := time.NewTimer(s.t.BuildWait)
	defer wait.Stop()
	select {
	case <-b.done:
		return b.answer, nil
	case <-wait.C:
	case <-ctx.Done():
		return nil, fmt.Errorf("await the speech map: %w", ctx.Err())
	}
	s.builds.mu.Lock()
	defer s.builds.mu.Unlock()
	switch {
	case b.answer != nil:
		return b.answer, nil
	case b.started:
		return &storagev1.BuildSpeechMapResponse{State: storagev1.SpeechMapState_SPEECH_MAP_STATE_BUILDING}, nil
	}
	return &storagev1.BuildSpeechMapResponse{State: storagev1.SpeechMapState_SPEECH_MAP_STATE_QUEUED}, nil
}

// current is whether row is a map of the file as it is now.
func current(row db.SubtitlesSpeechmap, id Identity) bool {
	return row.Version == speech.FormatVersion && bytes.Equal(row.Identity, id.encode())
}

func builtAnswer(row db.SubtitlesSpeechmap, id Identity) *storagev1.BuildSpeechMapResponse {
	return &storagev1.BuildSpeechMapResponse{
		State: storagev1.SpeechMapState_SPEECH_MAP_STATE_BUILT, Identity: id.wire(), Size: id.Size,
		Frames: row.Frames, Density: row.Density, Stream: int32(row.Stream), //nolint:gosec // an audio ordinal
	}
}

// join is the build of b's file: the one that runs or waits, given b's
// rank, or b, which waits. slots is how many may run at once from now on.
func (q *builds) join(b *build, slots int) (*build, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.slots = slots
	defer q.start()
	if j, ok := q.jobs[b.file]; ok {
		j.rank = b.rank
		return j, nil
	}
	if len(q.waiting) >= q.s.t.Queue {
		return nil, rpc.Errorf(rpc.CodeResourceExhausted, "%d speech maps already wait on this Node", len(q.waiting))
	}
	q.calls++
	b.call, b.done = q.calls, make(chan struct{})
	q.jobs[b.file] = b
	q.waiting = append(q.waiting, b)
	return b, nil
}

// start runs the first waiting builds while slots are free. q.mu is held.
func (q *builds) start() {
	slices.SortFunc(q.waiting, func(a, b *build) int { return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(a.call, b.call)) })
	for q.running < q.slots && len(q.waiting) > 0 {
		b := q.waiting[0]
		q.waiting = q.waiting[1:]
		b.started = true
		q.running++
		go q.run(b)
	}
}

func (q *builds) run(b *build) {
	answer := q.s.build(q.ctx, b)
	q.mu.Lock()
	defer q.mu.Unlock()
	q.running--
	delete(q.jobs, b.file)
	b.answer = answer
	close(b.done)
	q.start()
}

// build maps b's file, keeps the map and answers what came of it.
func (s *Sidecars) build(ctx context.Context, b *build) *storagev1.BuildSpeechMapResponse {
	m, err := s.mapper.Map(ctx, b.path, b.stream)
	if err == nil {
		m.Header.Identity = b.identity.encode()
		err = s.keep(ctx, b, m)
	}
	if d, ok := errors.AsType[*speech.DecodeError](err); ok {
		return failed(storagev1.SpeechMapFault_SPEECH_MAP_FAULT_DECODE_FAILED, d.First)
	}
	switch {
	case errors.Is(err, speech.ErrNoAudio):
		return failed(storagev1.SpeechMapFault_SPEECH_MAP_FAULT_NO_AUDIO, "")
	case err != nil:
		return failed(storagev1.SpeechMapFault_SPEECH_MAP_FAULT_NODE_FAULT, err.Error())
	}
	return &storagev1.BuildSpeechMapResponse{
		State: storagev1.SpeechMapState_SPEECH_MAP_STATE_BUILT, Identity: b.identity.wire(), Size: b.identity.Size,
		Frames: int64(m.Header.Frames), Density: m.Density(), Stream: int32(m.Header.Stream), //nolint:gosec // an audio ordinal
	}
}

func failed(f storagev1.SpeechMapFault, words string) *storagev1.BuildSpeechMapResponse {
	return &storagev1.BuildSpeechMapResponse{State: storagev1.SpeechMapState_SPEECH_MAP_STATE_FAILED, Fault: f, Words: words}
}

// keep writes m's bytes beside the others, then its index row.
func (s *Sidecars) keep(ctx context.Context, b *build, m speech.Map) error {
	enc := m.Encode()
	if len(enc) > s.t.MapCap {
		return &speech.DecodeError{First: fmt.Sprintf("the speech map is %d bytes, more than the %d a map may hold", len(enc), s.t.MapCap)}
	}
	p := s.mapPath(b.file)
	part := p + ".part"
	if err := writeSynced(part, enc); err != nil {
		return fmt.Errorf("keep the speech map: %w", err)
	}
	if err := os.Rename(part, p); err != nil {
		return fmt.Errorf("keep the speech map: %w", err)
	}
	err := s.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).PutSpeechMap(ctx, db.PutSpeechMapParams{
			FileID: b.file, RootID: b.root, Identity: b.identity.encode(), Version: speech.FormatVersion, Stream: int64(m.Header.Stream),
			Frames: int64(m.Header.Frames), Density: m.Density(), Size: b.identity.Size, BuiltAt: storage.Now(),
		})
	})
	if err != nil {
		return fmt.Errorf("index the speech map: %w", err)
	}
	return nil
}

func writeSynced(p string, b []byte) error {
	f, err := os.Create(p) //nolint:gosec // a map's path under the data dir
	if err != nil {
		return err //nolint:wrapcheck // keep wraps it, and the error names p
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(p)
	}
	return err
}

func (s *Sidecars) mapPath(file storage.UUID) string {
	return filepath.Join(s.maps, file.String()+mapExt)
}

func (s *Sidecars) mapRow(ctx context.Context, file storage.UUID) (db.SubtitlesSpeechmap, bool, error) {
	var row db.SubtitlesSpeechmap
	err := s.local.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Local]) error {
		var err error
		row, err = db.New(r).GetSpeechMap(ctx, file)
		return err
	})
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return row, false, nil
	case err != nil:
		return row, false, fmt.Errorf("read the speech map index: %w", err)
	}
	return row, true, nil
}

func (s *Sidecars) ReadSpeechMap(ctx context.Context, req *storagev1.ReadSpeechMapRequest) (*storagev1.ReadSpeechMapResponse, error) {
	file, err := fileID(req.GetFileId())
	if err != nil {
		return nil, err
	}
	root, err := rootID(req.GetRootId())
	if err != nil {
		return nil, err
	}
	rel, err := relOf(req.GetPath())
	if err != nil {
		return nil, err
	}
	stale := &storagev1.ReadSpeechMapResponse{Stale: true}
	dir, err := s.store.resolve(ctx, root, false)
	if err != nil {
		return nil, WireError(err)
	}
	info, err := s.store.stat(ctx, dir, rel)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return stale, nil
	case err != nil:
		return nil, WireError(err)
	}
	now := identityOfInfo(info)
	if !now.Same(Identity{Seen: seenOf(req.GetIdentity()), Size: req.GetSize()}) {
		return stale, nil
	}
	row, ok, err := s.mapRow(ctx, file)
	if err != nil || !ok || !current(row, now) {
		return stale, err
	}
	b, err := os.ReadFile(s.mapPath(file))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return stale, nil
	case err != nil:
		return nil, fmt.Errorf("read the speech map: %w", err)
	}
	return &storagev1.ReadSpeechMapResponse{Map: b}, nil
}

func (s *Sidecars) PruneSpeechMaps(ctx context.Context, req *storagev1.PruneSpeechMapsRequest) (*storagev1.PruneSpeechMapsResponse, error) {
	after, err := boundOf(req.GetAfter())
	if err != nil {
		return nil, err
	}
	before, err := boundOf(req.GetBefore())
	if err != nil {
		return nil, err
	}
	keep := map[storage.UUID]bool{}
	for _, k := range req.GetKeep() {
		id, err := fileID(k)
		if err != nil {
			return nil, err
		}
		keep[id] = true
	}
	var gone []storage.UUID
	err = s.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		q := db.New(tx)
		ids, err := mapsIn(ctx, q, after, before)
		if err != nil {
			return err
		}
		gone = gone[:0]
		for _, id := range ids {
			if keep[id] {
				continue
			}
			if err := q.DeleteSpeechMap(ctx, id); err != nil {
				return err
			}
			gone = append(gone, id)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("prune the speech maps: %w", err)
	}
	for _, id := range gone {
		if err := os.Remove(s.mapPath(id)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("prune the speech maps: %w", err)
		}
	}
	return &storagev1.PruneSpeechMapsResponse{Removed: int32(len(gone))}, nil //nolint:gosec // one Node's maps, far under 2³¹
}

// boundOf is a range's bound, the zero UUID when b is empty: no bound.
func boundOf(b []byte) (storage.UUID, error) {
	if len(b) == 0 {
		return storage.UUID{}, nil
	}
	return fileID(b)
}

// mapsIn are the files in [after, before) with a map; a zero bound is
// none, since the zero UUID binds as NULL.
func mapsIn(ctx context.Context, q *db.Queries, after, before storage.UUID) ([]storage.UUID, error) {
	switch {
	case after.IsZero() && before.IsZero():
		return q.SpeechMapIDs(ctx)
	case after.IsZero():
		return q.SpeechMapsBefore(ctx, before)
	case before.IsZero():
		return q.SpeechMapsFrom(ctx, after)
	}
	return q.SpeechMapsBetween(ctx, db.SpeechMapsBetweenParams{After: after, Before: before})
}

// sweep removes what a crash left in the maps folder, a part file or a map
// no row names, and the rows whose map is gone.
func (s *Sidecars) sweep(ctx context.Context) error {
	if err := os.MkdirAll(s.maps, 0o700); err != nil {
		return fmt.Errorf("make the folder: %w", err)
	}
	var ids []storage.UUID
	err := s.local.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Local]) error {
		var err error
		ids, err = db.New(r).SpeechMapIDs(ctx)
		return err
	})
	if err != nil {
		return err
	}
	indexed := map[string]bool{}
	for _, id := range ids {
		indexed[id.String()+mapExt] = true
	}
	entries, err := os.ReadDir(s.maps)
	if err != nil {
		return fmt.Errorf("list the folder: %w", err)
	}
	held := map[string]bool{}
	for _, e := range entries {
		if indexed[e.Name()] && e.Type().IsRegular() {
			held[e.Name()] = true
			continue
		}
		if err := os.RemoveAll(filepath.Join(s.maps, e.Name())); err != nil {
			return fmt.Errorf("remove what a stopped build left: %w", err)
		}
	}
	return s.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		q := db.New(tx)
		for _, id := range ids {
			if !held[id.String()+mapExt] {
				if err := q.DeleteSpeechMap(ctx, id); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func fileID(b []byte) (storage.UUID, error) {
	id, err := storage.UUIDFromBytes(b)
	if err != nil {
		return id, rpc.Errorf(rpc.CodeInvalidArgument, "file id: %v", err)
	}
	return id, nil
}
