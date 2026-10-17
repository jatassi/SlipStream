package link

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/joincode"
)

// await is one AwaitPair: the Controller its code names may dial in, and
// this Node pairs over that session as the client.
type await struct {
	code joincode.Code
	name string
	// done is closed once paired or err is set, under the link's lock.
	done   chan struct{}
	paired Paired
	err    error
}

// AwaitPair waits for the Controller code names to dial this Node, then
// sends Pair{secret, name} and Hello on that session as Pair does, and
// returns once it is live. Only that Controller's key passes TLS for an
// await, whatever Trust answers, and a session that fails before a
// Welcome leaves the wait running for the next dial. A Controller's
// refusal ends it with a *RefusedError, and ctx ending or Close with
// ctx's error.
func (l *Link) AwaitPair(ctx context.Context, code joincode.Code, name string) (Paired, error) {
	a := &await{code: code, name: name, done: make(chan struct{})}
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return Paired{}, errClosed
	}
	l.awaits[a] = struct{}{}
	l.mu.Unlock()
	defer func() {
		l.mu.Lock()
		delete(l.awaits, a)
		l.mu.Unlock()
	}()
	select {
	case <-a.done:
		return a.paired, a.err
	case <-ctx.Done():
		return Paired{}, fmt.Errorf("await the Controller's dial: %w", ctx.Err())
	case <-l.ctx.Done():
		return Paired{}, errClosed
	}
}

// awaiting is the open await whose code names key. Callers hold l.mu.
func (l *Link) awaiting(key Key) *await {
	for a := range l.awaits {
		if a.code.Names(key[:]) {
			select {
			case <-a.done:
			default:
				return a
			}
		}
	}
	return nil
}

func (l *Link) isAwaited(key Key) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.awaiting(key) != nil
}

// pairAwaited pairs over s, accepted from an awaited Controller, within
// ctx. It is false when no await names s's key, so s says Hello itself.
func (l *Link) pairAwaited(ctx context.Context, s *session) bool {
	l.mu.Lock()
	a := l.awaiting(s.key)
	l.mu.Unlock()
	if a == nil {
		return false
	}
	paired, err := l.pairOn(ctx, s, a.code, a.name)
	var refused *RefusedError
	if err != nil && !errors.As(err, &refused) {
		l.cfg.Log.Debug("pair over the Controller's dial failed", "peer", s.key.Fingerprint(), "err", err)
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-a.done:
	default:
		a.paired, a.err = paired, err
		close(a.done)
	}
	return true
}
