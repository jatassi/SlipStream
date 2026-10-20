package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Kept is every Grab whose Torrent node's client keeps, with the release's
// title, in its intent's order.
func Kept(ctx context.Context, r storage.Reader[file], node role.NodeID) ([]nodes.Grabbed, error) {
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
	of := map[downloadclients.ID][]db.ListGrabsOfTorrentsRow{}
	for _, row := range rows {
		of[row.TorrentID] = append(of[row.TorrentID], row)
	}
	var out []nodes.Grabbed
	for _, h := range held {
		for _, row := range of[h.ID] {
			out = append(out, nodes.Grabbed{Grab: row.ID.UUID, InfoHash: h.InfoHash, Title: row.ReleaseTitle, Done: h.Done, Wanted: h.Size})
		}
	}
	return out, nil
}
