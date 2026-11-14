package mediaserverlink

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

// watch is one server's supervisor: it keeps the server's watch running,
// starting it again with backoff when it fails, and reports each change in
// how the server responds.
type watch struct {
	l     *Link
	local *storage.DB[storage.Local]
	h     Holder
	t     target
	// version is the settings version of the credential the watch holds.
	version  int64
	reported contactState
	wait     time.Duration
}

type contactState struct {
	state link.Contact
	found string
}

func (w *watch) supervise(ctx context.Context) {
	tm := w.l.cfg.Timings
	w.wait = tm.ReconnectMin
	for {
		err := w.once(ctx)
		if ctx.Err() != nil {
			return
		}
		w.l.cfg.Log.Warn("the watch of a Media Server stopped", "server", w.t.name, "err", errString(err), "retry_in", w.wait.String())
		timer := time.NewTimer(w.wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		w.wait = min(2*w.wait, tm.ReconnectMax)
	}
}

func (w *watch) once(ctx context.Context) error {
	h, err := w.l.credential(ctx, w.t.ref)
	if err != nil {
		return err
	}
	w.version = h.version
	err = w.l.dial(w.t, h).Watch(ctx, w.t.every, func(s link.Seen) { w.note(ctx, contactOf(s.Problem), s.At) })
	if p, ok := link.ServerProblem(err); ok {
		w.note(ctx, contactOf(&p), time.Now())
	}
	return err
}

// contactOf is how a server responds when a look at it failed with p, or
// succeeded for nil.
func contactOf(p *link.Problem) contactState {
	if p == nil {
		return contactState{state: link.ContactResponding}
	}
	return contactState{state: contactsOf[p.Reason], found: p.Found}
}

var contactsOf = map[link.Reason]link.Contact{
	link.ReasonUnreachable: link.ContactNotResponding,
	link.ReasonRefused:     link.ContactSignInRefused,
	link.ReasonOtherServer: link.ContactOtherServer,
	link.ReasonUnsupported: link.ContactNotResponding,
	link.ReasonFailed:      link.ContactNotResponding,
}

// note reports c when it differs from what the watch last reported. A
// report that fails to queue goes again with the next look.
func (w *watch) note(ctx context.Context, c contactState, at time.Time) {
	if c.state == link.ContactResponding {
		w.wait = w.l.cfg.Timings.ReconnectMin
	}
	if c == w.reported {
		return
	}
	r := &msv1.Report{ServerId: w.t.ref.ID.Bytes(), Report: &msv1.Report_Contact{Contact: &msv1.Contact{
		State: string(c.state), SettingsVersion: w.version, AtUnixMs: at.UnixMilli(), Found: c.found,
	}}}
	err := w.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return w.h.Report(ctx, tx, r)
	})
	if err != nil {
		if ctx.Err() == nil {
			w.l.cfg.Log.Error("queue a Media Server's contact report", "server", w.t.name, "err", err.Error())
		}
		return
	}
	w.reported = c
}

func errString(err error) string {
	if err == nil {
		return "the watch returned"
	}
	return err.Error()
}
