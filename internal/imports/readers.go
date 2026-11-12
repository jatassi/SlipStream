package imports

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Cells is files.Inflight: grabs' cells. Stateless, so the composition
// root builds it before grabs.New and imports.Open.
type Cells struct{ Grabs grabs.GrabCells }

func (c Cells) GrabCells(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[files.CellKey]files.GrabCell, error) {
	return c.Grabs.GrabCells(ctx, r, ws)
}

func (c Cells) Unfinished(ctx context.Context, r storage.Reader[file]) ([]storage.UUID, error) {
	return c.Grabs.Unfinished(ctx, r)
}

func (c Cells) ReleaseTitles(ctx context.Context, r storage.Reader[file]) (map[storage.UUID]string, error) {
	return c.Grabs.ReleaseTitles(ctx, r)
}

// Uses is downloadclients.Uses: grabs' Uses of each Torrent. Stateless.
type Uses struct{ grabs.TorrentUses }

// RootImports is roots.Inflight: the unfinished Grabs into a root, or of
// ws's Works.
type RootImports struct{}

func (RootImports) Into(ctx context.Context, r storage.Reader[file], root roots.ID, ws []works.WorkID) (int64, error) {
	return grabs.RootGrabs{}.Into(ctx, r, root, ws)
}

func (RootImports) Release(ctx context.Context, tx *storage.Tx[file], root roots.ID) error {
	return grabs.RootGrabs{}.Release(ctx, tx, root)
}
