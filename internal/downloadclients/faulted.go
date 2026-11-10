package downloadclients

import (
	"context"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// FaultedTorrent is a Torrent with a release fault. Client is nil for its
// Node's built-in client.
type FaultedTorrent struct {
	ID       ID
	InfoHash bencode.InfoHash
	Fault    Fault
	Client   *ClientRef
	Node     role.NodeID
	Removed  bool
}

// Faulted is every Torrent with a release fault, removed ones too, oldest
// fault first. A setup fault, such as a client's error, a full disk or an
// away Node, is never a release fault.
func Faulted(ctx context.Context, r storage.Reader[file]) ([]FaultedTorrent, error) {
	rows, err := db.New(r).ListFaulted(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]FaultedTorrent, len(rows))
	for i, row := range rows {
		out[i] = FaultedTorrent{
			ID: row.ID, InfoHash: bencode.InfoHash(row.InfoHash), Node: row.NodeID, Removed: !row.RemovedAt.IsZero(),
			Fault:  Fault{Kind: FaultKind(*row.Fault), At: row.FaultAt.Time()},
			Client: refOf(row.ClientID, row.ClientKind, row.ClientName),
		}
	}
	return out, nil
}
