package roots

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// Place puts the file from at to in root, on the Node that holds root now,
// answering the placed file. The Node may refuse with a *WaitingError,
// another file at to is storing.ErrOccupied, and a place that needs a copy
// without allowCopy is *storing.NeedsCopyError.
func (r *Roots) Place(ctx context.Context, root ID, to storing.Rel, from storing.Source, allowCopy bool) (storing.Placed, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return storing.Placed{}, err
	}
	placed, err := r.cfg.Remote.Place(ctx, at, from, to, allowCopy)
	return placed, r.waiting(ctx, at.Node, err)
}

// PlaceSidecar is Place for a Sidecar its source's ledger marks
// SlipStream's: the Node records the placed file in root's ledger.
func (r *Roots) PlaceSidecar(ctx context.Context, root ID, to storing.Rel, from storing.Source, allowCopy bool) (storing.Placed, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return storing.Placed{}, err
	}
	placed, err := r.cfg.Remote.PlaceSidecar(ctx, at, from, to, allowCopy)
	return placed, r.waiting(ctx, at.Node, err)
}

// MoveSidecar renames the Sidecar from to to within root on its Node,
// and records it at to in root's ledger.
func (r *Roots) MoveSidecar(ctx context.Context, root ID, from, to storing.Rel) (storing.Placed, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return storing.Placed{}, err
	}
	videos, err := r.cfg.Videos(ctx, r.cfg.Library)
	if err != nil {
		return storing.Placed{}, err
	}
	moved, err := r.cfg.Remote.MoveSidecar(ctx, at, from, to, videos)
	return moved, r.waiting(ctx, at.Node, err)
}

// Move renames from to to within root on its Node, and with carry from's
// companions after it, answering the moved file and the companions left;
// moving again converges. The Node may refuse with a *WaitingError.
func (r *Roots) Move(ctx context.Context, root ID, from, to storing.Rel, carry bool) (storing.Placed, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return storing.Placed{}, err
	}
	videos, err := r.cfg.Videos(ctx, r.cfg.Library)
	if err != nil {
		return storing.Placed{}, err
	}
	moved, err := r.cfg.Remote.Move(ctx, at, from, to, carry, videos)
	return moved, r.waiting(ctx, at.Node, err)
}

// Survey answers each planned rename within root on its Node: the files
// at its target, compared without case, and with companions its source's
// companions. A root its Node can't read now is *storing.NotReady.
func (r *Roots) Survey(ctx context.Context, root ID, planned []storing.Planned, companions bool) ([]storing.Surveyed, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return nil, err
	}
	videos, err := r.cfg.Videos(ctx, r.cfg.Library)
	if err != nil {
		return nil, err
	}
	return r.cfg.Remote.Survey(ctx, at, planned, companions, videos)
}

// Prune removes folder from root on its Node, and each parent while it is
// empty, never a held folder, a kept folder or a parent of one. It
// answers the folders removed, innermost first.
func (r *Roots) Prune(ctx context.Context, root ID, folder storing.Rel, keep []storing.Rel) ([]storing.Rel, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return nil, err
	}
	return r.cfg.Remote.Prune(ctx, at, folder, keep)
}

// WaitingError is a root whose Node takes no new bytes into it now, so an
// import waits for its next report. Why names the Node and the cause in
// words.
type WaitingError struct {
	Why string
	Err *storing.NotReady
}

func (e *WaitingError) Error() string { return e.Why }
func (e *WaitingError) Unwrap() error { return e.Err }

func (r *Roots) waiting(ctx context.Context, node role.NodeID, err error) error {
	refused, ok := errors.AsType[*storing.NotReady](err)
	if !ok {
		return err
	}
	var name string
	if err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		name, err = nameOf(ctx, rd, node)
		return err
	}); err != nil {
		return errors.Join(refused, err)
	}
	return &WaitingError{Why: waitingWords(refused, name), Err: refused}
}

// waitingWords is why SlipStream can't put new files in the library root
// on node while refused holds.
func waitingWords(refused *storing.NotReady, node string) string {
	const wait = " New files wait until it's ready."
	switch {
	case refused.Draining:
		return node + " is finishing its Storage work, so SlipStream can't put new files in its library root."
	case refused.Unknown:
		return node + " hasn't confirmed the library root's folder yet. New files wait until it does."
	}
	switch refused.State {
	case storing.Missing:
		return "The library root's folder on " + node + " is missing." + wait
	case storing.Unmounted:
		return "The drive of the library root on " + node + " isn't mounted." + wait
	case storing.Foreign:
		return "The library root's folder on " + node + " belongs to another SlipStream." + wait
	case storing.Unwritable:
		return "SlipStream can't write to the library root's folder on " + node + "." + wait
	case storing.Unresponsive:
		return "The library root's folder on " + node + " isn't responding." + wait
	case storing.Ready:
	}
	return "SlipStream is still checking the library root on " + node + "." + wait
}

func nameOf(ctx context.Context, rd storage.Reader[file], node role.NodeID) (string, error) {
	named, err := nodes.Names(ctx, rd)
	if err != nil {
		return "", err
	}
	for _, n := range named {
		if n.ID == node {
			return n.Name, nil
		}
	}
	return "Its Node", nil
}

// Remove removes rel from root on its Node; a file already gone wraps
// fs.ErrNotExist.
func (r *Roots) Remove(ctx context.Context, root ID, rel storing.Rel) error {
	at, err := r.at(ctx, root)
	if err != nil {
		return err
	}
	return r.cfg.Remote.Remove(ctx, at, rel)
}

// RemoveAll removes the folder rel and everything in it from root on its
// Node; a folder already gone is removed.
func (r *Roots) RemoveAll(ctx context.Context, root ID, rel storing.Rel) error {
	at, err := r.at(ctx, root)
	if err != nil {
		return err
	}
	return r.cfg.Remote.RemoveAll(ctx, at, rel)
}

// RemoveIf removes rel from root on its Node only while its size and
// modified time, to the millisecond, are the ones asked, answering
// whether it was removed, gone or changed.
func (r *Roots) RemoveIf(ctx context.Context, root ID, rel storing.Rel, size int64, modified time.Time) (storing.Removal, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return 0, err
	}
	return r.cfg.Remote.RemoveIf(ctx, at, rel, size, modified)
}

// ProbeMove reads one page of a Move plan on node, its companions by the
// Admin's video file extensions.
func (r *Roots) ProbeMove(ctx context.Context, node role.NodeID, p storing.ProbeMove) (storing.Probed, error) {
	videos, err := r.cfg.Videos(ctx, r.cfg.Library)
	if err != nil {
		return storing.Probed{}, err
	}
	p.Videos = videos
	return r.cfg.Remote.ProbeMove(ctx, node, p)
}

func (r *Roots) ListHeld(ctx context.Context, root ID, folder storing.HeldFolder) ([]string, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return nil, err
	}
	return r.cfg.Remote.List(ctx, at, folder)
}

func (r *Roots) at(ctx context.Context, id ID) (storing.At, error) {
	var root Root
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		root, err = Get(ctx, rd, id)
		return err
	})
	return storing.At{Node: root.Node, Root: root.ID.UUID}, err
}

// Watch is closed at the next change of any root's report: its state or
// volume, or the place its Node judged it at.
func (r *Roots) Watch() <-chan struct{} {
	r.watch.Lock()
	defer r.watch.Unlock()
	return r.watch.ch
}

func (r *Roots) moved() {
	r.watch.Lock()
	defer r.watch.Unlock()
	close(r.watch.ch)
	r.watch.ch = make(chan struct{})
}

// UnreachableError is a root whose Node can't take its Works' new
// downloads: offline, or its Storage written off.
type UnreachableError struct {
	State State
	// Node is the Node's name.
	Node string
	// Why names the Node and what keeps it, in words.
	Why string
}

func (e *UnreachableError) Error() string { return e.Why }

// Reachable is nil when root's Node may take new downloads into it, else
// an *UnreachableError, read in rd and the roster. A root its Node finds
// unmounted, unwritable, unresponsive or draining is reachable: its
// imports wait until it is ready.
func (r *Roots) Reachable(ctx context.Context, rd storage.Reader[file], root Root) error {
	lost, err := LostIn(ctx, rd)
	if err != nil {
		return err
	}
	var state State
	switch {
	case lost[root.ID]:
		state = WrittenOff
	case !r.cfg.Roster.InStep(root.Node):
		state = Offline
	default:
		return nil
	}
	name, err := nameOf(ctx, rd, root.Node)
	if err != nil {
		return err
	}
	if state == WrittenOff {
		return &UnreachableError{State: state, Node: name, Why: "SlipStream stopped waiting for " + name + "'s Storage, so it can't put new files in its library root."}
	}
	return &UnreachableError{State: state, Node: name, Why: name + " is offline, so SlipStream can't put new files in its library root."}
}
