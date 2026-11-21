package reconciling

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// Waits is what Reconciling waits on, for the Reconciling page, the
// banner and each refusal's detail. No gate reads it.
type Waits struct {
	// Rehome: the old Controller's folders aren't found yet.
	Rehome bool
	// Nodes is each Node whose Holdings aren't adopted, in id order.
	Nodes []NodeWait
	// Roots is each root not scanned since Reconciling began.
	Roots []roots.ID
}

// NodeWait is one Node Reconciling waits on.
type NodeWait struct {
	Node nodes.ID
	// Reported: every page of its Holdings is in.
	Reported bool
	// Refused is why SlipStream refused its Holdings, "" when it didn't.
	Refused string
}

// Waiting is what Reconciling waits on in r, the zero Waits outside it.
// A Node continued without, or a root, waits no more.
func Waiting(ctx context.Context, r storage.Reader[file]) (Waits, error) {
	q := db.New(r)
	state, err := q.GetState(ctx)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return Waits{}, nil
	case err != nil:
		return Waits{}, fmt.Errorf("read what Reconciling waits on: %w", err)
	}
	w := Waits{Rehome: state.RehomeSavedAt.IsZero()}
	rows, err := q.ListNodes(ctx)
	if err != nil {
		return Waits{}, fmt.Errorf("read what Reconciling waits on: %w", err)
	}
	for _, row := range rows {
		if row.AdoptedAt.IsZero() && row.ContinuedWithoutAt.IsZero() {
			nw := NodeWait{Node: row.NodeID, Reported: !row.ReportedAt.IsZero()}
			if row.Refused != nil {
				nw.Refused = *row.Refused
			}
			w.Nodes = append(w.Nodes, nw)
		}
	}
	rootRows, err := q.ListRoots(ctx)
	if err != nil {
		return Waits{}, fmt.Errorf("read what Reconciling waits on: %w", err)
	}
	for _, row := range rootRows {
		if row.ScannedAt.IsZero() && row.ContinuedWithoutAt.IsZero() {
			w.Roots = append(w.Roots, row.RootID)
		}
	}
	return w, nil
}

// tryEnd ends Reconciling in one Write once the old Controller's folders
// are found, every Node's Holdings are adopted or it was continued
// without, and every root has scanned or was continued without.
func (s *Service) tryEnd(ctx context.Context) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		state, err := q.GetState(ctx)
		if errors.Is(err, storage.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if state.RehomeSavedAt.IsZero() {
			return nil
		}
		nodesLeft, err := q.CountFinished(ctx)
		if err != nil || nodesLeft > 0 {
			return err
		}
		rootsLeft, err := q.CountUnscanned(ctx)
		if err != nil || rootsLeft > 0 {
			return err
		}
		if err := q.DeleteRoots(ctx); err != nil {
			return err
		}
		return q.DeleteState(ctx)
	})
	if err != nil {
		return fmt.Errorf("end Reconciling: %w", err)
	}
	return nil
}
