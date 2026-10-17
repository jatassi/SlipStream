package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// DownloadingOwner counts the torrents in a Node's Downloading intent. They
// seed until seeding goals exist, so a Node never finishes them on its own.
var DownloadingOwner = nodes.Owner{Role: role.Downloading, Held: heldTorrents}

func heldTorrents(ctx context.Context, r storage.Reader[file], node role.NodeID) (nodes.Held, error) {
	row, err := db.New(r).CountTorrentsOfNode(ctx, node)
	return nodes.Held{Units: row.Torrents, Noun: nodes.Noun{One: "torrent", Many: "torrents"}, Bytes: row.Bytes, Endless: true}, err
}
