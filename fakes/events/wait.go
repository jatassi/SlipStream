package events

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const tick = 25 * time.Millisecond

// Wait returns the first record of the run in runDir that q matches,
// already logged or appended later. It is the end-to-end tier's polling
// wait on the log: it reads what was appended every 25 ms until ctx ends,
// and fails at once when the run stopped with no match.
func Wait(ctx context.Context, runDir string, q Query) (Record, error) {
	f, err := os.Open(filepath.Join(runDir, File))
	if err != nil {
		return Record{}, fmt.Errorf("read the event log: %w", err)
	}
	defer func() { _ = f.Close() }()
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	var pending []byte
	lines := 0
	for {
		more, err := io.ReadAll(f)
		if err != nil {
			return Record{}, fmt.Errorf("read the event log: %w", err)
		}
		pending = append(pending, more...)
		recs, used, err := parse(pending, lines)
		if err != nil {
			return Record{}, err
		}
		pending, lines = pending[used:], lines+len(recs)
		for _, r := range recs {
			if q.Matches(r) {
				return r, nil
			}
			if r.Kind == RunStopped {
				return Record{}, fmt.Errorf("no record matches %s: the run stopped at seq %d", q, r.Seq)
			}
		}
		select {
		case <-ctx.Done():
			return Record{}, fmt.Errorf("no record matches %s: %w", q, ctx.Err())
		case <-ticker.C:
		}
	}
}
