package logs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/system"
)

// maxBatch is the most records one Read returns.
const maxBatch = 1000

// ErrClosed is every Read's answer once EndReads has run.
var ErrClosed = errors.New("the log serves no more reads")

const subsystemKey = "subsystem"

// ring keeps the newest records at the log's level for the viewer. Seqs
// count up from base, when the log opened in Unix microseconds, so a seq
// below base came from an earlier process.
type ring struct {
	opened time.Time
	base   uint64
	ended  chan struct{}
	end    sync.Once

	mu sync.Mutex
	// head is the newest record's seq, base while there is none.
	head uint64
	// recs is circular: seq s sits at s % len(recs). held counts the
	// records it holds.
	recs []system.LogRecord
	held uint64
	// changed is closed and replaced at each record.
	changed chan struct{}
	// encoded receives the JSON of one record's attributes.
	encoded bytes.Buffer
}

func newRing(size int) *ring {
	now := time.Now()
	base := uint64(now.UnixMicro())
	return &ring{
		opened:  now,
		base:    base,
		ended:   make(chan struct{}),
		head:    base,
		recs:    make([]system.LogRecord, size),
		changed: make(chan struct{}),
	}
}

// add appends rec as the next seq. The caller holds mu.
func (r *ring) add(rec system.LogRecord) {
	r.head++
	rec.Seq = r.head
	r.recs[r.head%uint64(len(r.recs))] = rec
	r.held = min(r.held+1, uint64(len(r.recs)))
	close(r.changed)
	r.changed = make(chan struct{})
}

// resize keeps the newest size records.
func (r *ring) resize(size int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if size == len(r.recs) {
		return
	}
	recs := make([]system.LogRecord, size)
	keep := min(r.held, uint64(len(recs)))
	for s := r.head - keep + 1; s <= r.head; s++ {
		recs[s%uint64(len(recs))] = r.recs[s%uint64(len(r.recs))]
	}
	r.recs, r.held = recs, keep
}

// batch is what a reader whose last record was after reads now, false
// when it has to wait for one. after 0 reads the whole ring. The caller
// holds mu.
func (r *ring) batch(after uint64) (system.LogBatch, bool) {
	oldest := r.head - r.held + 1
	from := after + 1
	var b system.LogBatch
	switch {
	case after == 0:
		from = oldest
	case after < r.base || after > r.head:
		b.Reset = &system.LogReset{Reason: system.Restarted, StartedAt: r.opened, After: oldest - 1}
		from = oldest
	case from < oldest:
		b.Reset = &system.LogReset{Reason: system.Lapped, StartedAt: r.opened, After: oldest - 1}
		from = oldest
	}
	for s := from; s <= min(r.head, from+maxBatch-1); s++ {
		b.Records = append(b.Records, r.recs[s%uint64(len(r.recs))])
	}
	return b, b.Reset != nil || len(b.Records) > 0
}

// Read returns the records after after, waiting up to wait for the first.
// It returns an empty batch when none came in time, and starts over with
// a reset when after was lapped or came from an earlier process.
func (l *Log) Read(ctx context.Context, after uint64, wait time.Duration) (system.LogBatch, error) {
	r := l.ring
	timer := time.NewTimer(wait)
	defer timer.Stop()
	for {
		r.mu.Lock()
		b, ready := r.batch(after)
		changed := r.changed
		r.mu.Unlock()
		select {
		case <-r.ended:
			return system.LogBatch{}, ErrClosed
		default:
		}
		if ready {
			return b, nil
		}
		select {
		case <-changed:
		case <-timer.C:
			return system.LogBatch{}, nil
		case <-r.ended:
			return system.LogBatch{}, ErrClosed
		case <-ctx.Done():
			return system.LogBatch{}, fmt.Errorf("read the log: %w", context.Cause(ctx))
		}
	}
}

// EndReads ends every Read, now and later, with ErrClosed. Records still
// reach the file and the ring.
func (l *Log) EndReads() {
	l.ring.end.Do(func() { close(l.ring.ended) })
}

// ringHandler feeds the ring. Its "subsystem" attribute becomes the
// record's Subsystem, and enc writes the rest as JSON into the ring.
type ringHandler struct {
	ring      *ring
	level     slog.Leveler
	enc       slog.Handler
	subsystem string
	grouped   bool
}

func newRingHandler(r *ring, level slog.Leveler) *ringHandler {
	enc := slog.NewJSONHandler(&r.encoded, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && (a.Key == slog.TimeKey || a.Key == slog.LevelKey || a.Key == slog.MessageKey) {
				return slog.Attr{}
			}
			return a
		},
	})
	return &ringHandler{ring: r, level: level, enc: enc}
}

func (h *ringHandler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.level.Level() }

func (h *ringHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	var rest []slog.Attr
	for _, a := range attrs {
		if !h.grouped && a.Key == subsystemKey {
			c.subsystem = a.Value.Resolve().String()
			continue
		}
		rest = append(rest, a)
	}
	c.enc = h.enc.WithAttrs(rest)
	return &c
}

func (h *ringHandler) WithGroup(name string) slog.Handler {
	c := *h
	c.enc = h.enc.WithGroup(name)
	c.grouped = c.grouped || name != ""
	return &c
}

func (h *ringHandler) Handle(ctx context.Context, rec slog.Record) error {
	subsystem := h.subsystem
	rest := slog.NewRecord(rec.Time, rec.Level, rec.Message, rec.PC)
	rec.Attrs(func(a slog.Attr) bool {
		if !h.grouped && a.Key == subsystemKey {
			subsystem = a.Value.Resolve().String()
		} else {
			rest.AddAttrs(a)
		}
		return true
	})
	r := h.ring
	r.mu.Lock()
	defer r.mu.Unlock()
	r.encoded.Reset()
	if err := h.enc.Handle(ctx, rest); err != nil {
		return fmt.Errorf("encode a log record: %w", err)
	}
	r.add(system.LogRecord{
		Time:      rec.Time,
		Level:     string(levelOf(rec.Level)),
		Subsystem: subsystem,
		Message:   rec.Message,
		Attrs:     bytes.Clone(bytes.TrimSuffix(r.encoded.Bytes(), []byte("\n"))),
	})
	return nil
}

// levelOf is the Level a record's slog level falls in.
func levelOf(l slog.Level) Level {
	switch {
	case l < slog.LevelInfo:
		return Debug
	case l < slog.LevelWarn:
		return Info
	case l < slog.LevelError:
		return Warn
	}
	return Error
}
