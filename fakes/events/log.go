package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// Log is a run's one writer of events.jsonl.
type Log struct {
	observe func(Record) []Record

	mu  sync.Mutex
	f   *os.File
	seq uint64
}

// Create makes events.jsonl in runDir, refusing one that exists. observe
// hears each record once it is written, under the writer's lock, and
// returns the records it causes, which are appended before any other.
func Create(runDir string, observe func(Record) []Record) (*Log, error) {
	f, err := os.OpenFile(filepath.Join(runDir, File), os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_EXCL, 0o644) //nolint:gosec // the event log is evidence anyone on the machine may read
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", File, err)
	}
	return &Log{observe: observe, f: f}, nil
}

// Append writes rec as the next record and returns it with its Seq and At.
func (l *Log) Append(rec Record) (Record, error) {
	return l.Apply(func() (Record, []Record, error) { return rec, nil, nil })
}

// Apply runs change under the writer's lock and, unless it fails, appends
// the record it returns and then the records that follow from it, so no
// other record lands between a change and its record.
func (l *Log) Apply(change func() (rec Record, follow []Record, err error)) (Record, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, follow, err := change()
	if err != nil {
		return Record{}, err
	}
	written, err := l.writeLocked(rec)
	if err != nil {
		return Record{}, err
	}
	queue := slices.Concat(follow, l.observe(written))
	for len(queue) > 0 {
		next, err := l.writeLocked(queue[0])
		if err != nil {
			return written, err
		}
		queue = append(queue[1:], l.observe(next)...)
	}
	return written, nil
}

func (l *Log) writeLocked(rec Record) (Record, error) {
	rec.Seq, rec.At = l.seq+1, time.Now().UTC()
	line, err := json.Marshal(rec)
	if err != nil {
		return Record{}, fmt.Errorf("%s record: %w", rec.Kind, err)
	}
	if _, err := l.f.Write(append(line, '\n')); err != nil {
		return Record{}, fmt.Errorf("append to %s: %w", File, err)
	}
	l.seq = rec.Seq
	return rec, nil
}

// Close ends the log. Appends after it fail.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.f.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		return fmt.Errorf("close %s: %w", File, err)
	}
	return nil
}
