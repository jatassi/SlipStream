package link

import (
	"context"
	"sync"
	"time"
)

// Watcher is how a kind looks at a server's sessions. Poll reads them now.
// Socket, when the kind has one, holds the server's push socket until ctx
// ends or the socket fails: it calls opened once the socket is open, push
// for what a message carries, and poll for a message that means the
// sessions changed without listing them.
type Watcher struct {
	Poll   func(ctx context.Context) Seen
	Socket func(ctx context.Context, opened func(), push func(Seen), poll func()) error
}

// Run polls every every, and at once after a socket message asks for it,
// while the socket is held open beside the polls and opened again after
// t.ReconnectMin, doubling to t.ReconnectMax. It calls seen from one
// goroutine only, and returns nil once ctx ends.
func (w Watcher) Run(ctx context.Context, t Timings, every time.Duration, seen func(Seen)) error {
	pushed := make(chan Seen)
	pollNow := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	defer wg.Wait()
	defer cancel()
	if w.Socket != nil {
		wg.Go(func() { w.hold(ctx, t, pushed, pollNow) })
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		s := w.Poll(ctx)
		if ctx.Err() != nil {
			return nil
		}
		seen(s)
		if !waitForPoll(ctx, tick.C, pollNow, pushed, seen) {
			return nil
		}
	}
}

// waitForPoll passes each push to seen until the next poll is due, false
// once ctx ends.
func waitForPoll(ctx context.Context, tick <-chan time.Time, pollNow <-chan struct{}, pushed <-chan Seen, seen func(Seen)) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case <-tick:
			return true
		case <-pollNow:
			return true
		case s := <-pushed:
			seen(s)
		}
	}
}

func (w Watcher) hold(ctx context.Context, t Timings, pushed chan<- Seen, pollNow chan<- struct{}) {
	wait := t.ReconnectMin
	opened := func() { wait = t.ReconnectMin }
	push := func(s Seen) {
		select {
		case pushed <- s:
		case <-ctx.Done():
		}
	}
	poll := func() {
		select {
		case pollNow <- struct{}{}:
		default:
		}
	}
	for {
		_ = w.Socket(ctx, opened, push, poll)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		wait = min(2*wait, t.ReconnectMax)
	}
}
