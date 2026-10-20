package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Kept is every Torrent node's client keeps, in its intent's order,
// titled by its newest Grab's release, or by its name once no Grab is
// left.
func Kept(ctx context.Context, r storage.Reader[file], node role.NodeID) ([]nodes.Kept, error) {
	held, err := downloadclients.Kept(ctx, r, node)
	if err != nil {
		return nil, err
	}
	ids := make([]downloadclients.ID, len(held))
	for i, h := range held {
		ids[i] = h.ID
	}
	rows, err := db.New(r).ListGrabsOfTorrents(ctx, ids)
	if err != nil {
		return nil, err
	}
	newest := map[downloadclients.ID]string{}
	for _, row := range rows {
		newest[row.TorrentID] = row.ReleaseTitle
	}
	out := make([]nodes.Kept, len(held))
	for i, h := range held {
		title, ok := newest[h.ID]
		if !ok {
			title = h.Name
		}
		out[i] = nodes.Kept{Torrent: h.ID.UUID, Title: title, Done: h.Done, Wanted: h.Size}
	}
	return out, nil
}
