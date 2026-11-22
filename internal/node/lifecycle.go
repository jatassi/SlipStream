package node

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/jatassi/slipstream/internal/atomicfile"
	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/lifecycle"
)

// lifecycle is d's record and swap. retry is timings.backups.windows_retry,
// how long a rename or delete retries a sharing violation on Windows.
func (d DataDir) lifecycle(retry time.Duration) lifecycle.Dir {
	return lifecycle.Dir{Path: d.path, Retry: retry}
}

// RestoreStage is a kind of step a restore's finish at start takes.
type RestoreStage string

const (
	// RestoreRename is one rename of the swap.
	RestoreRename RestoreStage = "rename"
	// RestoreSwapped is the record advancing to swapped.
	RestoreSwapped RestoreStage = "swapped"
	// RestoreWritten is the record advancing to written.
	RestoreWritten RestoreStage = "written"
	// RestoreAside is deleting the files the swap moved aside.
	RestoreAside RestoreStage = "aside"
	// RestoreFinish is deleting the record.
	RestoreFinish RestoreStage = "finish"
)

// RestoreStep is one step of a restore's finish, as
// Test.BeforeRestoreStage sees it. Rename is set for a RestoreRename.
type RestoreStep struct {
	Stage  RestoreStage
	Rename lifecycle.Step
}

// resumeLifecycle runs before library.db opens. It finishes the operation
// lifecycle.json names, or, with no record, deletes the files a staging
// left without writing one. A record this binary can't resume refuses the
// start with RecordRefused.
func (n *Node) resumeLifecycle(ctx context.Context) error {
	d := n.env.DataDir.lifecycle(n.cfg.Timings.Backups.WindowsRetry)
	op, ok, err := readLifecycle(d)
	if err != nil {
		return err
	}
	if !ok {
		return errors.Join(d.Leftovers(), atomicfile.Remove(filepath.Join(d.Path, backups.UploadFile), d.Retry))
	}
	switch op := op.(type) {
	case *lifecycle.Restore:
		return n.finishRestore(ctx, d, op)
	case *lifecycle.Wipe:
		return n.finishWipe(d, op)
	}
	return nil
}

// readLifecycle is d's record, with one this binary can't resume as a
// *RecordError.
func readLifecycle(d lifecycle.Dir) (lifecycle.Op, bool, error) {
	op, ok, err := d.Read()
	if _, refused := errors.AsType[*lifecycle.RefusedError](err); refused {
		return nil, false, &RecordError{Err: err}
	}
	return op, ok, err
}

// finishRestore takes a restore from its record's stage to the end. Each
// step is derived from the files and the stage, so a start stopped before
// any of them finishes the rest at the next start.
func (n *Node) finishRestore(ctx context.Context, d lifecycle.Dir, op *lifecycle.Restore) error {
	n.setPhase(PhaseRestoring)
	defer n.setPhase(PhaseStarting)
	before := n.env.Test.beforeRestoreStage
	advance := func(stage RestoreStage, to lifecycle.Stage) error {
		if err := before(RestoreStep{Stage: stage}); err != nil {
			return err
		}
		op.Stage = to
		return d.Advance(op)
	}
	if op.Stage == lifecycle.Staged {
		if err := d.Swap(func(s lifecycle.Step) error { return before(RestoreStep{Stage: RestoreRename, Rename: s}) }); err != nil {
			return err
		}
		if err := advance(RestoreSwapped, lifecycle.Swapped); err != nil {
			return err
		}
	}
	if op.Stage == lifecycle.Swapped {
		if err := n.writeRestored(ctx, op); err != nil {
			return err
		}
		if err := advance(RestoreWritten, lifecycle.Written); err != nil {
			return err
		}
	}
	if err := before(RestoreStep{Stage: RestoreAside}); err != nil {
		return err
	}
	if err := d.Aside(); err != nil {
		return err
	}
	if err := before(RestoreStep{Stage: RestoreFinish}); err != nil {
		return err
	}
	return d.Finish()
}

// lifecycleIdle is lifecycle.ErrBusy while lifecycle.json exists: a
// restore staged before a restart, or one this Node is committing. A
// claim or a join then would make a library the next start replaces.
func (n *Node) lifecycleIdle() error {
	_, busy, err := readLifecycle(n.env.DataDir.lifecycle(n.cfg.Timings.Backups.WindowsRetry))
	if err != nil {
		return err
	}
	if busy {
		return lifecycle.ErrBusy
	}
	return nil
}
