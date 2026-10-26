package roots

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// Place puts payload from at to in root, on the Node that holds root now,
// answering the file's size. The Node may refuse with a *WaitingError,
// and another file at to is storing.ErrOccupied.
func (r *Roots) Place(ctx context.Context, root ID, to storing.Rel, from storing.Payload) (int64, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return 0, err
	}
	size, err := r.cfg.Remote.Place(ctx, at, from, to)
	return size, r.waiting(ctx, at.Node, err)
}

// Move renames from to to within root on its Node, answering the file's
// size; moving again converges. The Node may refuse with a *WaitingError.
func (r *Roots) Move(ctx context.Context, root ID, from, to storing.Rel) (int64, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return 0, err
	}
	size, err := r.cfg.Remote.Move(ctx, at, from, to)
	return size, r.waiting(ctx, at.Node, err)
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
	var why string
	switch {
	case refused.Draining:
		why = name + " is draining Storage, so the library root there takes no new files."
	case refused.Unknown:
		why = name + " hasn't taken up the library root yet, so it takes no new files."
	default:
		why = "The library root on " + name + " reads " + refused.State.String() + ", so it takes no new files until it is ready."
	}
	return &WaitingError{Why: why, Err: refused}
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
		return &UnreachableError{State: state, Node: name, Why: name + "'s Storage was written off, so the library root there takes no new files."}
	}
	return &UnreachableError{State: state, Node: name, Why: name + " is offline, so the library root there takes no new files."}
}
