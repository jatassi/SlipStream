package roots

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// Rehome is a root's new place. A nil Node or Path keeps the root's own.
type Rehome struct {
	Node *role.NodeID
	Path *string
	// Forget is the count of missing files the Admin confirmed forgetting;
	// nil forgets none.
	Forget *int
}

// Absent is a recorded file a re-home didn't find, as recorded, at the
// new place.
type Absent struct {
	Path   storing.Rel
	Reason storing.MissingReason
}

// MissingError is a re-home refused while recorded files aren't at the new
// place: how many, and the first of them in file order.
type MissingError struct {
	Count int
	First []Absent
}

func (e *MissingError) Error() string {
	return fmt.Sprintf("%d recorded files aren't at the new place", e.Count)
}

const firstMissing = 20

// errStale is a re-home whose root recorded a file its check hasn't seen.
var errStale = errors.New("the root recorded a file the re-home hasn't checked")

// Rehome moves root id to a new place: a folder its Node judges usable
// apart from every other root, holding every file the root records at the
// same path and size. Files that aren't there are a *MissingError unless
// h.Forget is their count, and then they're forgotten in the Write that
// moves the root. The root's place bumps, so its Node adopts it there.
func (r *Roots) Rehome(ctx context.Context, _ identity.Admin, id ID, h Rehome) (View, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var was Root
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		was, err = Get(ctx, rd, id)
		return err
	})
	if err != nil {
		return View{}, err
	}
	to := was
	if h.Node != nil {
		to.Node = *h.Node
	}
	if h.Path != nil {
		to.Path = *h.Path
	}
	if !r.cfg.Roster.Holds(role.Storage, to.Node) {
		return View{}, ErrNotStorage
	}
	v, err := r.check(ctx, to.Node, to.Path, &id)
	if err != nil {
		return View{}, err
	}
	if v.Refusal != nil {
		return View{}, &UnusableError{Refusal: *v.Refusal}
	}
	to.Path = v.Path
	defer func() {
		r.setProgress(id, nil)
		r.cfg.Changed.Publish(ctx, changedOf([]ID{id}))
	}()
	checked := map[storing.Recorded]storing.MissingReason{}
	for {
		var recorded []storing.Recorded
		err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
			var err error
			recorded, err = r.cfg.Contents.Recorded(ctx, rd, id)
			return err
		})
		if err != nil {
			return View{}, err
		}
		if err := r.verify(ctx, to, recorded, checked); err != nil {
			return View{}, err
		}
		r.setProgress(id, nil)
		view, err := r.move(ctx, to, h.Forget, checked)
		if errors.Is(err, errStale) {
			continue
		}
		if err != nil {
			return View{}, err
		}
		r.cfg.Storage.Touch(was.Node, to.Node)
		return view, nil
	}
}

// verify checks at to's place each of recorded that checked lacks, noting
// why each one isn't there, a zero reason for one that is. The root's row
// shows the progress, and roots.changed carries it at most once a second.
func (r *Roots) verify(ctx context.Context, to Root, recorded []storing.Recorded, checked map[storing.Recorded]storing.MissingReason) error {
	var todo []storing.Recorded
	for _, f := range recorded {
		if _, ok := checked[f]; !ok {
			todo = append(todo, f)
		}
	}
	before, total := int64(len(recorded)-len(todo)), int64(len(recorded))
	r.setProgress(to.ID, &Progress{Checked: before, Total: total})
	r.cfg.Changed.Publish(ctx, changedOf([]ID{to.ID}))
	if len(todo) == 0 {
		return nil
	}
	shown := time.Now()
	missing, err := r.cfg.Remote.Verify(ctx, to.Node, to.Path, todo, func(n int) {
		r.setProgress(to.ID, &Progress{Checked: before + int64(n), Total: total})
		if time.Since(shown) >= time.Second {
			shown = time.Now()
			r.cfg.Changed.Publish(ctx, changedOf([]ID{to.ID}))
		}
	})
	if err != nil {
		return reached(err)
	}
	for _, f := range todo {
		checked[f] = 0
	}
	for _, m := range missing {
		checked[todo[m.Index]] = m.Reason
	}
	return nil
}

// move saves to, forgetting the files checked found missing when forget
// is their count, unless the root recorded a file checked lacks.
func (r *Roots) move(ctx context.Context, to Root, forget *int, checked map[storing.Recorded]storing.MissingReason) (View, error) {
	var view View
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		recorded, err := r.cfg.Contents.Recorded(ctx, tx, to.ID)
		if err != nil {
			return err
		}
		missing, err := absentOf(recorded, checked)
		if err != nil {
			return err
		}
		if len(missing) > 0 && (forget == nil || *forget != len(missing)) {
			return &MissingError{Count: len(missing), First: missing[:min(len(missing), firstMissing)]}
		}
		paths := make([]storing.Rel, len(missing))
		for i, m := range missing {
			paths[i] = m.Path
		}
		if err := r.cfg.Contents.Forget(ctx, tx, to.ID, paths); err != nil {
			return err
		}
		if err := db.New(tx).MoveRoot(ctx, db.MoveRootParams{ID: to.ID, NodeID: to.Node, Path: to.Path}); err != nil {
			return err
		}
		moved, err := Get(ctx, tx, to.ID)
		if err != nil {
			return err
		}
		views, err := r.views(ctx, tx, []Root{moved})
		if err != nil {
			return err
		}
		view = views[0]
		return nil
	})
	return view, err
}

// absentOf is the files of recorded checked found missing, errStale when
// checked lacks one.
func absentOf(recorded []storing.Recorded, checked map[storing.Recorded]storing.MissingReason) ([]Absent, error) {
	var out []Absent
	for _, f := range recorded {
		reason, ok := checked[f]
		if !ok {
			return nil, errStale
		}
		if reason != 0 {
			out = append(out, Absent{Path: f.Path, Reason: reason})
		}
	}
	return out, nil
}

func (r *Roots) setProgress(id ID, p *Progress) {
	r.rehoming.Lock()
	defer r.rehoming.Unlock()
	if p == nil {
		delete(r.rehoming.of, id)
		return
	}
	r.rehoming.of[id] = *p
}

func (r *Roots) progressOf(id ID) *Progress {
	r.rehoming.Lock()
	defer r.rehoming.Unlock()
	p, ok := r.rehoming.of[id]
	if !ok {
		return nil
	}
	return &p
}
