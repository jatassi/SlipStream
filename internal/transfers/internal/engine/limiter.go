package engine

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/time/rate"
)

// limiter is the Node's one send limit, across every lane of every
// Transfer it sends, route 3 included. Writes go in pieces of piece bytes,
// with a burst of the larger of a piece and one second of the limit.
type limiter struct {
	piece int
	l     *rate.Limiter

	mu      sync.Mutex
	applied int64
}

func newLimiter(piece int64) *limiter {
	return &limiter{piece: int(piece), l: rate.NewLimiter(rate.Inf, int(piece))}
}

// apply sets the limit to bytesPerSecond, 0 for none, when it differs from
// the one it applied last.
func (l *limiter) apply(bytesPerSecond int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if bytesPerSecond == l.applied {
		return
	}
	l.applied = bytesPerSecond
	if bytesPerSecond <= 0 {
		l.l.SetLimit(rate.Inf)
		return
	}
	l.l.SetBurst(max(l.piece, int(bytesPerSecond)))
	l.l.SetLimit(rate.Limit(bytesPerSecond))
}

// binding reports whether the limit holds back n bytes sent now.
func (l *limiter) binding(n int) bool {
	return l.l.Limit() != rate.Inf && l.l.Tokens() < float64(n)
}

func (l *limiter) wait(ctx context.Context, n int) error {
	if err := l.l.WaitN(ctx, n); err != nil {
		return fmt.Errorf("wait out the send limit: %w", err)
	}
	return nil
}
