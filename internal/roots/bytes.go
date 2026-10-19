package roots

import (
	"context"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// Place puts payload from at to in root, on the Node that holds root now,
// answering the file's size. The Node may refuse with *storing.NotReady,
// and another file at to is storing.ErrOccupied.
func (r *Roots) Place(ctx context.Context, root ID, to storing.Rel, from storing.Payload) (int64, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return 0, err
	}
	return r.cfg.Remote.Place(ctx, at, from, to)
}

// Move renames from to to within root on its Node, answering the file's
// size; moving again converges.
func (r *Roots) Move(ctx context.Context, root ID, from, to storing.Rel) (int64, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return 0, err
	}
	return r.cfg.Remote.Move(ctx, at, from, to)
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

// ListHeld names what folder holds in root on its Node.
func (r *Roots) ListHeld(ctx context.Context, root ID, folder storing.HeldFolder) ([]string, error) {
	at, err := r.at(ctx, root)
	if err != nil {
		return nil, err
	}
	return r.cfg.Remote.List(ctx, at, folder)
}

// at is where the Storage service reaches root now.
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
	// Why names the Node and what keeps it, in words.
	Why string
}

func (e *UnreachableError) Error() string { return e.Why }

// Reachable is nil when root's Node may take new downloads into it, else
// an *UnreachableError, read in rd and the roster. A root its Node finds
// unmounted, unwritable, unresponsive or draining is reachable: its
// imports wait until it is ready.
func (r *Roots) Reachable(ctx context.Context, rd storage.Reader[file], root Root) error {
	writtenOff, err := nodes.WrittenOffIn(ctx, rd, role.Storage)
	if err != nil {
		return err
	}
	var state State
	switch {
	case writtenOff[root.Node]:
		state = WrittenOff
	case !r.cfg.Roster.InStep(root.Node):
		state = Offline
	default:
		return nil
	}
	named, err := nodes.Names(ctx, rd)
	if err != nil {
		return err
	}
	name := "Its Node"
	for _, n := range named {
		if n.ID == root.Node {
			name = n.Name
		}
	}
	if state == WrittenOff {
		return &UnreachableError{State: state, Why: name + "'s Storage was written off, so the library root there takes no new files."}
	}
	return &UnreachableError{State: state, Why: name + " is offline, so the library root there takes no new files."}
}
