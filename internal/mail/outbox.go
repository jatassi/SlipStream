package mail

import (
	"context"
	"log/slog"
	"sync"
)

// Outbox sends the messages nobody waits for, after the response. Each
// Post sends on its own goroutine and logs any outcome but Sent.
type Outbox struct {
	send   Sender
	log    *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc

	mu      sync.Mutex
	closed  bool
	flights sync.WaitGroup
}

func NewOutbox(s Sender, log *slog.Logger) *Outbox {
	ctx, cancel := context.WithCancel(context.Background())
	return &Outbox{send: s, log: log, ctx: ctx, cancel: cancel}
}

func (o *Outbox) Post(m Message) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		o.log.Warn("mail not sent", "to", m.To.Email, "subject", m.Subject, "reason", "shutting down")
		return
	}
	o.flights.Go(func() {
		out, err := o.send.Send(o.ctx, m)
		switch {
		case err != nil:
			o.log.Error("mail not sent", "to", m.To.Email, "subject", m.Subject, "err", err.Error())
		case out.Kind != Sent:
			o.log.Warn("mail not sent", "to", m.To.Email, "subject", m.Subject, "outcome", string(out.Kind), "reply", out.Reply)
		}
	})
}

// Close takes no more messages and waits for those in flight. When ctx
// ends first it cancels them, waits for them to return and answers ctx's
// error.
func (o *Outbox) Close(ctx context.Context) error {
	o.mu.Lock()
	o.closed = true
	o.mu.Unlock()
	done := make(chan struct{})
	go func() {
		o.flights.Wait()
		close(done)
	}()
	defer o.cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		o.cancel()
		<-done
		return ctx.Err() //nolint:wrapcheck // the caller's own deadline
	}
}
