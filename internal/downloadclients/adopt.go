package downloadclients

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Adopted is a torrent a Node holds that the library had no Torrent for.
type Adopted struct {
	Node role.NodeID
	// Client is nil for the built-in client.
	Client *ClientID
	Held   Holding
	// Metainfo is checked to hash to Held's info hash, nil only for an
	// external client that can't export it.
	Metainfo []byte
}

// Adopt adds a Torrent for a, adopted: no Grab made it, so it has no
// Content Type. It keeps the Node's seed time and save folder, and the
// metainfo's name, size, privacy and trackers when a carries it, else
// what the Node reported.
func Adopt(ctx context.Context, tx *storage.Tx[file], a Adopted) (ID, error) {
	q := db.New(tx)
	h := a.Held
	name, hosts, private := "", h.Hosts, h.Private
	var size int64
	if h.Content != nil {
		name, size = h.Content.Name, h.Content.Size
	}
	if a.Metainfo != nil {
		m, err := bencode.DecodeMetainfo(a.Metainfo)
		if err != nil {
			return ID{}, fmt.Errorf("adopt torrent %s: %w", h.InfoHash, err)
		}
		name, hosts, private, size = m.Info.Name, HostsOf(m.Trackers()), m.Info.Private, 0
		for _, f := range m.Info.Files {
			size += f.Length
		}
	}
	trackers, err := json.Marshal(append([]string{}, hosts...))
	if err != nil {
		return ID{}, fmt.Errorf("encode the trackers: %w", err)
	}
	order, err := q.NextOrder(ctx)
	if err != nil {
		return ID{}, err
	}
	var folder *string
	if h.Folder != "" {
		folder = &h.Folder
	}
	var priv int64
	if private {
		priv = 1
	}
	id := ID{UUID: storage.NewUUID()}
	return id, q.InsertAdopted(ctx, db.InsertAdoptedParams{
		ID: id, NodeID: a.Node, ClientID: a.Client, InfoHash: h.InfoHash[:], Metainfo: a.Metainfo, Name: name, Size: size,
		Private: priv, Trackers: string(trackers), SaveFolder: folder, AddedOrder: order, AddedAt: storage.Now(),
		SeedingSeconds: int64(h.Seed.Seconds()),
	})
}

// Recorded is the Torrent the library records for an info hash: its live
// one, or the one a write-off of its Node parted.
type Recorded struct {
	ID         ID
	Node       role.NodeID
	WrittenOff bool
}

// RecordedIn is the Torrent the library records for each of hashes that
// it records one for, in r's snapshot. A hash live on several Nodes
// answers the first added.
func RecordedIn(ctx context.Context, r storage.Reader[file], hashes []bencode.InfoHash) (map[bencode.InfoHash]Recorded, error) {
	out := map[bencode.InfoHash]Recorded{}
	if len(hashes) == 0 {
		return out, nil
	}
	raw := make([][]byte, len(hashes))
	for i := range hashes {
		raw[i] = hashes[i][:]
	}
	rows, err := db.New(r).ListRecorded(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("read the recorded Torrents: %w", err)
	}
	for _, row := range rows {
		h := bencode.InfoHash(row.InfoHash)
		if _, ok := out[h]; !ok {
			out[h] = Recorded{ID: row.ID, Node: row.NodeID, WrittenOff: row.WrittenOff}
		}
	}
	return out, nil
}

// OnNode is one of a Node's live Torrents.
type OnNode struct {
	ID       ID
	InfoHash bencode.InfoHash
}

// LiveOnNode is a page of at most page of node's live Torrents after
// after in id order, leaving out the review list's.
func LiveOnNode(ctx context.Context, r storage.Reader[file], node role.NodeID, after ID, page int) ([]OnNode, error) {
	rows, err := db.New(r).ListLiveOnNodeAfter(ctx, db.ListLiveOnNodeAfterParams{NodeID: node, After: after, Page: int64(page)})
	if err != nil {
		return nil, fmt.Errorf("list a Node's Torrents: %w", err)
	}
	out := make([]OnNode, len(rows))
	for i, row := range rows {
		out[i] = OnNode{ID: row.ID, InfoHash: bencode.InfoHash(row.InfoHash)}
	}
	return out, nil
}

// MoveRecorded moves Torrent id to client on node, keeping its totals,
// Grab and reported_at. A Torrent a write-off parted is live again on
// node. A built-in target keeps no external origin or raw counters. The
// caller Touches both Nodes after commit.
func MoveRecorded(ctx context.Context, tx *storage.Tx[file], id ID, node role.NodeID, client *ClientID) error {
	n, err := db.New(tx).MoveRecorded(ctx, db.MoveRecordedParams{NodeID: node, ClientID: client, ID: id})
	if err != nil {
		return fmt.Errorf("move Torrent %s: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("move Torrent %s: SlipStream no longer tracks it", id)
	}
	return nil
}

// Withhold puts Torrents ids on the review list, which keeps them out of
// every intent.
func Withhold(ctx context.Context, tx *storage.Tx[file], ids []ID) error {
	if len(ids) == 0 {
		return nil
	}
	return db.New(tx).Withhold(ctx, db.WithholdParams{At: storage.Now(), Ids: ids})
}

// Release takes Torrents ids off the review list, back into their Nodes'
// intents.
func Release(ctx context.Context, tx *storage.Tx[file], ids []ID) error {
	if len(ids) == 0 {
		return nil
	}
	return db.New(tx).Release(ctx, ids)
}
