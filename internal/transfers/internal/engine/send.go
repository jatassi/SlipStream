package engine

import (
	"errors"
	"fmt"
	"io"

	"github.com/zeebo/blake3"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// dialSends opens the sender's lanes while its route has it dial, one at a
// time, and serves each until it breaks.
func (r *run) dialSends() {
	for r.ctx.Err() == nil {
		route := dials(r.spec.side, r.route())
		if route == tv1.Route_ROUTE_UNSPECIFIED {
			select {
			case <-r.routeChanged():
			case <-r.ctx.Done():
			}
			continue
		}
		l, err := r.open(route)
		if err == nil {
			heard := r.serveSends(l)
			r.drop(l)
			if heard {
				r.heard(route)
				continue
			}
		}
		if !r.pause(r.failed(route)) {
			return
		}
	}
}

// serveSends answers the receiver's frames on l until the lane ends, and
// reports whether the receiver sent any. The receiver drives, so a lane
// waits on it for as long as it takes.
func (r *run) serveSends(l *lane) (heard bool) {
	var open *sending
	defer func() { open.close() }()
	buf := make([]byte, ChunkSize)
	for {
		fr, err := l.read(0)
		if err != nil {
			return heard
		}
		heard = true
		if err := r.answer(l, fr, &open, buf); err != nil {
			r.e.cfg.Log.Warn("a sender's lane ended", "transfer", r.spec.id, "err", err)
			return heard
		}
	}
}

// answer answers one receiver frame on l: an Open with the file's
// Manifest or why it is absent, a Want with its chunks.
func (r *run) answer(l *lane, fr *tv1.Frame, open **sending, buf []byte) error {
	switch v := fr.GetFrame().(type) {
	case *tv1.Frame_Open:
		(*open).close()
		var reply *tv1.Frame
		*open, reply = r.openSource(v.Open)
		if reply == nil {
			return fmt.Errorf("the receiver opened file %d, which its Transfer doesn't send", v.Open.GetPosition())
		}
		return l.send(reply)
	case *tv1.Frame_Want:
		if *open == nil {
			return errors.New("the receiver requested chunks with no file open")
		}
		for _, i := range v.Want.GetChunks() {
			if err := r.sendChunk(l, *open, int(i), buf); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("the receiver sent %T, which a sender doesn't accept", fr.GetFrame())
}

// sending is a file a sender has open.
type sending struct {
	position int
	f        storing.ReadAtCloser
	r        io.ReaderAt
	size     int64
}

func (s *sending) close() {
	if s != nil {
		_ = s.f.Close()
	}
}

// openSource answers an Open: the file's Manifest, or why it is Absent.
// A file the Transfer doesn't name ends the lane, with no reply.
func (r *run) openSource(o *tv1.Open) (*sending, *tv1.Frame) {
	f, ok := r.spec.file(int(o.GetPosition()), o.GetSource())
	if !ok {
		return nil, nil
	}
	path, err := r.e.cfg.Sources.Open(f.ref)
	if err != nil {
		return nil, absentFrame(err)
	}
	ctx := r.ctx
	in, err := r.e.cfg.Disk.OpenAt(ctx, path)
	if err != nil {
		return nil, absentFrame(&source.Absent{Reason: source.NotReady, Detail: err.Error()})
	}
	info, err := r.e.cfg.Disk.Stat(ctx, path)
	if err != nil {
		_ = in.Close()
		return nil, absentFrame(&source.Absent{Reason: source.NotReady, Detail: err.Error()})
	}
	s := &sending{position: f.position, f: in, r: in, size: info.Size()}
	if r.e.cfg.Hooks.Read != nil {
		s.r = r.e.cfg.Hooks.Read(in)
	}
	return s, &tv1.Frame{Frame: &tv1.Frame_Manifest{Manifest: &tv1.Manifest{
		Size:      info.Size(),
		ChunkSize: ChunkSize,
		Identity:  &tv1.Identity{Size: info.Size(), MtimeUnixNs: info.ModTime().UnixNano(), FileId: info.FileID()},
	}}}
}

func absentFrame(err error) *tv1.Frame {
	a := &tv1.Absent{Reason: tv1.Absent_REASON_NOT_READY, Detail: err.Error()}
	if absent, ok := errors.AsType[*source.Absent](err); ok {
		a.Reason, a.Detail = absentReasons[absent.Reason], absent.Detail
	}
	return &tv1.Frame{Frame: &tv1.Frame_Absent{Absent: a}}
}

var absentReasons = map[source.Reason]tv1.Absent_Reason{
	source.Gone:     tv1.Absent_REASON_GONE,
	source.Failed:   tv1.Absent_REASON_FAILED,
	source.NotReady: tv1.Absent_REASON_NOT_READY,
}

// sendChunk reads chunk i of s, hashes it as read and sends it, paced by
// the Node's send limit.
func (r *run) sendChunk(l *lane, s *sending, i int, buf []byte) error {
	if i < 0 || i >= chunks(s.size) {
		return fmt.Errorf("the receiver wants chunk %d of %d", i, chunks(s.size))
	}
	b := buf[:chunkLen(s.size, i)]
	if _, err := s.r.ReadAt(b, offset(i)); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read chunk %d: %w", i, err)
	}
	hash := blake3.Sum256(b)
	if r.e.cfg.Hooks.Sent != nil {
		r.e.cfg.Hooks.Sent(r.spec.id, s.position, i, b)
	}
	binding := r.e.limit.binding(len(b))
	if binding {
		r.mu.Lock()
		r.binding = true
		r.mu.Unlock()
	}
	c := &tv1.Chunk{Index: uint32(i), Hash: hash[:], Binding: binding, Data: b} //nolint:gosec // under the file's chunk count
	if err := l.sendPaced(r.ctx, &tv1.Frame{Frame: &tv1.Frame_Chunk{Chunk: c}}, r.e.limit); err != nil {
		return err
	}
	r.e.facts.progress()
	return nil
}

func chunks(size int64) int { return int((size + ChunkSize - 1) / ChunkSize) }

func offset(i int) int64 { return int64(i) * ChunkSize }

func chunkLen(size int64, i int) int64 { return min(ChunkSize, size-offset(i)) }
