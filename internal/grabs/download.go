package grabs

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// TorrentUses answers downloadclients which Grabs used each Torrent.
type TorrentUses struct{}

// Of is each Torrent's Grabs, newest first.
func (TorrentUses) Of(ctx context.Context, r storage.Reader[file], ids []downloadclients.ID) (map[downloadclients.ID][]downloadclients.Use, error) {
	rows, err := db.New(r).ListGrabsOfTorrents(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := map[downloadclients.ID][]downloadclients.Use{}
	for _, row := range slices.Backward(rows) {
		u := downloadclients.Use{
			Grab: row.ID.UUID, Work: row.WorkID, Item: row.ItemID, Edition: string(row.Edition), Release: row.ReleaseTitle,
			Stage: row.Stage.Use(), Created: row.CreatedAt.Time(),
		}
		if row.FileIndex != nil {
			u.File = new(int(*row.FileIndex))
		}
		u.Floor = seeding.Stated(row.MinRatio, seconds(row.MinSeedSeconds))
		out[row.TorrentID] = append(out[row.TorrentID], u)
	}
	return out, nil
}

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
