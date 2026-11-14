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
	// watching is whether this watch has sent its Watching report.
	watching bool
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
	err = w.l.dial(w.t, h).Watch(ctx, w.t.every, func(s link.Seen) { w.look(ctx, s) })
	if p, ok := link.ServerProblem(err); ok {
		w.look(ctx, link.Seen{At: time.Now(), Problem: &p})
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

// look queues, in one Write, the reports s causes: a change in how the
// server responds, the plays it starts, changes and ends, the items a
// push said changed, and after the watch's first poll its Watching. The
// facts and the plays move on only once the reports are queued, so a look
// that fails to queue is reported again with the next.
func (w *watch) look(ctx context.Context, s link.Seen) {
	c := contactOf(s.Problem)
	if c.state == link.ContactResponding {
		w.wait = w.l.cfg.Timings.ReconnectMin
	}
	diff, reports, facts := w.l.diff(w.t.ref.ID).Next(s, storage.NewUUID)
	if c != w.reported {
		reports = append([]*msv1.Report{{ServerId: w.t.ref.ID.Bytes(), Report: &msv1.Report_Contact{Contact: &msv1.Contact{
			State: string(c.state), SettingsVersion: w.version, AtUnixMs: s.At.UnixMilli(), Found: c.found,
		}}}}, reports...)
	}
	if len(s.Items) > 0 {
		reports = append(reports, &msv1.Report{ServerId: w.t.ref.ID.Bytes(), Report: &msv1.Report_IndexHint{IndexHint: &msv1.IndexHint{ItemKeys: s.Items}}})
	}
	watching := s.Polled && s.Problem == nil
	if watching && !w.watching {
		reports = append(reports, diff.Watching(s.At))
	}
	if len(reports) > 0 {
		err := w.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
			for _, r := range reports {
				if err := w.h.Report(ctx, tx, r); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			if ctx.Err() == nil {
				w.l.cfg.Log.Error("queue a Media Server's reports", "server", w.t.name, "err", err.Error())
			}
			return
		}
	}
	w.reported, w.watching = c, w.watching || watching
	w.l.keep(w.h, w.t.ref.ID, diff, facts)
}

func errString(err error) string {
	if err == nil {
		return "the watch returned"
	}
	return err.Error()
}
