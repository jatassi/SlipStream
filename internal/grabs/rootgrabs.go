package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// RootGrabs is roots.Inflight over the Grabs.
type RootGrabs struct{}

func (RootGrabs) Into(ctx context.Context, r storage.Reader[storage.Library], root roots.ID, ws []works.WorkID) (int64, error) {
	return db.New(r).CountGrabsInto(ctx, db.CountGrabsIntoParams{RootID: root, WorkIds: ws})
}

func (RootGrabs) Release(ctx context.Context, tx *storage.Tx[storage.Library], root roots.ID) error {
	return db.New(tx).ReleaseRoot(ctx, root)
}
