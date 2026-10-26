package link

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
)

// errNotWelcomed is an invited session the Node didn't pair over within
// handshake.
var errNotWelcomed = errors.New("the Node didn't pair over the session")

// Invite dials t pinning its full key, as a Controller dials a Node that
// waits in AwaitPair, and returns once the Node's Pair and Hello made the
// session live. It dials again with backoff after every failure, and
// returns at once while t's key already has a live session. ctx ending
// or Close returns ctx's error with the last attempt's.
func (l *Link) Invite(ctx context.Context, t Target) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(l.ctx, cancel)
	defer stop()
	var last error
	for n := 0; ; n++ {
		if _, live := l.State(t.Key).(Live); live {
			return nil
		}
		err := l.invite(ctx, t)
		if err == nil {
			return nil
		}
		last = err
		wait := time.NewTimer(backoff.Delay(l.cfg.Timings.RedialMin, l.cfg.Timings.RedialMax, n, l.cfg.Rand))
		select {
		case <-wait.C:
		case <-ctx.Done():
			wait.Stop()
			return fmt.Errorf("invite %s: %w", t.Key.Fingerprint(), errors.Join(ctx.Err(), last))
		}
	}
}

// invite is one dial of Invite, bounded by handshake.
func (l *Link) invite(ctx context.Context, t Target) error {
	ctx, cancel := context.WithTimeout(ctx, l.cfg.Timings.Handshake)
	defer cancel()
	s, err := l.dialSession(ctx, t.Addresses, pinned(t.Key))
	if err != nil {
		return err
	}
	select {
	case <-s.welcomed:
		return nil
	case <-s.mux.Done():
		return errNotWelcomed
	case <-ctx.Done():
		_ = s.mux.Close()
		return errNotWelcomed
	}
}

// pinned is the dialer's check that the peer holds key.
func pinned(key Key) func(Key) error {
	return func(k Key) error {
		if k != key {
			return fmt.Errorf("peer key %s is not the pinned key %s", k.Fingerprint(), key.Fingerprint())
		}
		return nil
	}
}
