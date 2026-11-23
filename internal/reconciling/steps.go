package reconciling

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// Changed tells Admins that what Reconciling waits on, or the review
// list, changed.
type Changed struct{}

var ChangedEvent = live.AdminEvent[Changed]("reconciling.changed")

type ChangedPublisher interface {
	Publish(ctx context.Context, v Changed)
}

var (
	ErrNotReconciling = errors.New("SlipStream isn't Reconciling")
	ErrNoRoot         = errors.New("this root isn't one Reconciling waits on")
	ErrNoNode         = errors.New("this Node isn't one Reconciling waits on")
)

// Save is one root the re-home step saves: at a new place, or where it
// is when Node and Path are nil.
type Save struct {
	Root roots.ID
	roots.Rehome
}

// Saved is how one Save went: Err is the root edit's refusal.
type Saved struct {
	Root roots.ID
	Err  error
}

// Rehome is the re-home step: each of save goes through the root edit's
// re-home, which checks the folder and clears the root's Media Server
// Path mapping confirmation, and a root it moved is saved, so its scan
// starts. A root it leaves out is Later: no scan reads it and Reconciling
// waits on it. done ends the step.
func (s *Service) Rehome(ctx context.Context, by identity.Admin, save []Save, done bool) ([]Saved, error) {
	if err := s.reconciling(ctx); err != nil {
		return nil, err
	}
	out := make([]Saved, 0, len(save))
	for _, sv := range save {
		if err := s.waitsOnRoot(ctx, sv.Root); err != nil {
			return nil, err
		}
		_, err := s.cfg.Rehome(ctx, by, sv.Root, sv.Rehome)
		if err == nil {
			err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
				return db.New(tx).SaveRoot(ctx, db.SaveRootParams{At: storage.Now(), RootID: sv.Root})
			})
		}
		out = append(out, Saved{Root: sv.Root, Err: err})
	}
	if done {
		err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			return db.New(tx).SetRehomeSaved(ctx, storage.Now())
		})
		if err != nil {
			return out, fmt.Errorf("finish finding the old Controller's folders: %w", err)
		}
	}
	s.changed(ctx)
	return out, nil
}

// Scanned is imports.Config.Scanned: the Write that finishes a saved
// root's scan ends Reconciling's wait on it.
func (s *Service) Scanned(ctx context.Context, tx *storage.Tx[file], root roots.ID) error {
	n, err := db.New(tx).ScanRoot(ctx, db.ScanRootParams{At: storage.Now(), RootID: root})
	if err != nil {
		return fmt.Errorf("record a root's scan for Reconciling: %w", err)
	}
	if n > 0 {
		s.scanned.Store(true)
	}
	return nil
}

// Later is whether root waits in the re-home step, Later, so no scan may
// read it.
func Later(ctx context.Context, r storage.Reader[file], root roots.ID) (bool, error) {
	row, err := db.New(r).GetRoot(ctx, root)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("read whether a root waits for its folder: %w", err)
	}
	return row.SavedAt.IsZero() && row.ContinuedWithoutAt.IsZero(), nil
}

// ContinueNode stops Reconciling waiting for node. It still keeps every
// torrent until its Holdings are adopted.
func (s *Service) ContinueNode(ctx context.Context, node nodes.ID) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		row, err := db.New(tx).GetNode(ctx, node)
		switch {
		case errors.Is(err, storage.ErrNoRows):
			return ErrNoNode
		case err != nil:
			return err
		case !row.AdoptedAt.IsZero():
			return ErrNoNode
		}
		return ContinueWithout(ctx, tx, node)
	})
	if err != nil {
		return err
	}
	s.touch([]role.NodeID{node})
	s.changed(ctx)
	return nil
}

// ContinueRoot stops Reconciling waiting for root's scan.
func (s *Service) ContinueRoot(ctx context.Context, root roots.ID) error {
	if err := s.waitsOnRoot(ctx, root); err != nil {
		return err
	}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).ContinueRoot(ctx, db.ContinueRootParams{At: storage.Now(), RootID: root})
	})
	if err != nil {
		return err
	}
	s.changed(ctx)
	return nil
}

// Finish is Finish now: Reconciling stops waiting for everything left,
// and every Node still keeps its torrents until it reports them.
func (s *Service) Finish(ctx context.Context) error {
	if err := s.reconciling(ctx); err != nil {
		return err
	}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		now := storage.Now()
		if err := q.SetRehomeSaved(ctx, now); err != nil {
			return err
		}
		if err := q.ContinueAllNodes(ctx, now); err != nil {
			return err
		}
		return q.ContinueAllRoots(ctx, now)
	})
	if err != nil {
		return fmt.Errorf("finish Reconciling now: %w", err)
	}
	s.changed(ctx)
	return nil
}

func (s *Service) reconciling(ctx context.Context) error {
	return s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		_, err := db.New(r).GetState(ctx)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNotReconciling
		}
		return err
	})
}

func (s *Service) waitsOnRoot(ctx context.Context, root roots.ID) error {
	return s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		_, err := db.New(r).GetRoot(ctx, root)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNoRoot
		}
		return err
	})
}

// changed publishes reconciling.changed and has Run look again.
func (s *Service) changed(ctx context.Context) {
	s.cfg.Changed.Publish(ctx, Changed{})
	s.Wake()
}

// scan starts the scan of each saved root Reconciling waits on once its
// Node reports it Ready, and again each time it turns Ready: a scan of a
// root its Node hasn't checked since it started ends Skipped, and records
// nothing Reconciling waits on.
func (s *Service) scan(ctx context.Context) error {
	var waits []roots.ID
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		waits, err = db.New(r).ListToScan(ctx)
		return err
	})
	if err != nil || len(waits) == 0 {
		return err
	}
	ready, err := s.cfg.Ready(ctx)
	if err != nil {
		return err
	}
	var due []roots.ID
	for _, id := range waits {
		switch {
		case !slices.ContainsFunc(ready, func(r roots.Root) bool { return r.ID == id }):
			delete(s.asked, id)
		case !s.asked[id]:
			due = append(due, id)
		}
	}
	if len(due) == 0 {
		return nil
	}
	if err := s.cfg.Scan(ctx, due); err != nil {
		return fmt.Errorf("start the scans Reconciling waits on: %w", err)
	}
	for _, id := range due {
		s.asked[id] = true
	}
	return nil
}

// Later is imports.Reconciling's Later.
func (s *Service) Later(ctx context.Context, r storage.Reader[file], root roots.ID) (bool, error) {
	return Later(ctx, r, root)
}
