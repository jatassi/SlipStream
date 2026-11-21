package node

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/lifecycle"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/supersededapi"
)

// The steps of a Wipe and join's finish beside the deletes, as
// Test.BeforeWipeStep sees them.
const (
	WipeFinish = "finish"
	WipeJoin   = "join"
)

// superseded is whether this Node is a Superseded Controller.
func (n *Node) superseded() bool {
	h := n.half.Load()
	return h != nil && h.nodes != nil && h.nodes.Superseded()
}

func (s surfaceNode) Superseded() bool { return s.n.superseded() }

func (s surfaceNode) Fence() (supersededapi.Fence, bool) {
	h := s.n.half.Load()
	if h == nil || h.nodes == nil {
		return nil, false
	}
	return fence{s.n, h}, true
}

// fence is the Controller half's fence as the Superseded page reaches it.
type fence struct {
	n *Node
	h *half
}

func (f fence) Fenced(ctx context.Context) (nodes.Fence, bool, error) { return f.h.nodes.Fenced(ctx) }

func (f fence) Name(node nodes.ID) string { return f.h.nodes.Name(node) }

func (f fence) Stop(ctx context.Context) error { return f.h.nodes.Stop(ctx) }

func (f fence) Retire(ctx context.Context) error { return f.h.nodes.Retire(ctx) }

func (f fence) Resume(ctx context.Context) error {
	return f.h.nodes.Resume(ctx, f.h.backups.RegenerateFence)
}

// Wipe takes the superseded Backup and waits for it, then writes the wipe
// record and restarts. Until the record is written nothing changes, so a
// failed Backup leaves the Controller as it was.
func (f fence) Wipe(ctx context.Context, code joincode.Code) error {
	n := f.n
	n.claimMu.Lock()
	defer n.claimMu.Unlock()
	if !f.h.nodes.Superseded() {
		return nodes.ErrNotSuperseded
	}
	if err := n.lifecycleIdle(); err != nil {
		return err
	}
	run, err := f.h.backups.Take(ctx, backups.Request{Trigger: archive.Superseded})
	if err != nil {
		return err
	}
	if _, err := run.Wait(ctx); err != nil {
		n.log.Error("the Backup before Wipe and join failed", "err", err.Error())
		return fmt.Errorf("%w: %w", supersededapi.ErrBackupFailed, err)
	}
	if err := n.env.DataDir.lifecycle(n.cfg.Timings.Backups.WindowsRetry).Begin(&lifecycle.Wipe{Stage: lifecycle.Deleting, JoinCode: code.String()}); err != nil {
		return err
	}
	n.log.Info("restarting to wipe this Controller and join the new one")
	return n.Restart()
}

// finishWipe deletes the library, its keys and the Backups index, each
// only when present, then ends the record. The start goes on Unclaimed
// and joins with the record's Join Code, which a crash after Finish
// loses: the Node then waits Unclaimed with its Setup code, as a refused
// code leaves it.
func (n *Node) finishWipe(d lifecycle.Dir, op *lifecycle.Wipe) error {
	before := n.env.Test.beforeWipeStep
	if err := d.Wipe(before); err != nil {
		return err
	}
	code, err := joincode.Parse(op.JoinCode)
	if err != nil {
		n.log.Warn("the Join Code saved for Wipe and join is damaged, so this Node waits for a Join Code", "err", err.Error())
	}
	if err := before(WipeFinish); err != nil {
		return err
	}
	if err := d.Finish(); err != nil {
		return err
	}
	n.wiped = code
	n.log.Info("wiped this Controller's library, joining the new Controller")
	return nil
}

// localPage is a paired Worker's own page, which takes a Join Code while
// the Worker can't reach its Controller.
type localPage struct{ n *Node }

func (l localPage) Open() {
	code := l.n.gate.OpenForWorker()
	l.n.log.Warn(fmt.Sprintf("To pair this Node again, open %s and enter Setup code %s.", nodeURL(l.n.host.facts.Load().Listeners.HTTP), code))
}

func (l localPage) Shut() { l.n.gate.Shut() }
