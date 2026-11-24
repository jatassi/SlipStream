package downloadclients

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"slices"

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
// it records one for, as here reads it, in r's snapshot. A hash live on
// several Nodes answers the first added, unless a Move's arriving Torrent
// of it is on here: that one is recorded on its own Node.
func RecordedIn(ctx context.Context, r storage.Reader[file], here role.NodeID, hashes []bencode.InfoHash) (map[bencode.InfoHash]Recorded, error) {
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
		if _, ok := out[h]; !ok || row.Arriving && row.NodeID == here {
			out[h] = Recorded{ID: row.ID, Node: row.NodeID, WrittenOff: row.WrittenOff}
		}
	}
	return out, nil
}

// OnNode is one of a Node's live Torrents, and its place in added order.
type OnNode struct {
	ID       ID
	InfoHash bencode.InfoHash
	Order    int64
}

// LiveOnNode is a page of at most page of node's live Torrents added
// after order after, leaving out the review list's.
func LiveOnNode(ctx context.Context, r storage.Reader[file], node role.NodeID, after int64, page int) ([]OnNode, error) {
	rows, err := db.New(r).ListLiveOnNodeAfter(ctx, db.ListLiveOnNodeAfterParams{NodeID: node, After: after, Page: int64(page)})
	if err != nil {
		return nil, fmt.Errorf("list a Node's Torrents: %w", err)
	}
	out := make([]OnNode, len(rows))
	for i, row := range rows {
		out[i] = OnNode{ID: row.ID, InfoHash: bencode.InfoHash(row.InfoHash), Order: row.AddedOrder}
	}
	return out, nil
}

// Rerecord records Torrent id on client on node, keeping its totals,
// Grab and reported_at. A Torrent a write-off parted is live again on
// node. A built-in target keeps no external origin or raw counters. The
// caller Touches both Nodes after commit.
func Rerecord(ctx context.Context, tx *storage.Tx[file], id ID, node role.NodeID, client *ClientID) error {
	n, err := db.New(tx).Rerecord(ctx, db.RerecordParams{NodeID: node, ClientID: client, ID: id})
	if err != nil {
		return fmt.Errorf("record Torrent %s on another Node: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("record Torrent %s on another Node: SlipStream no longer tracks it", id)
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

// LiveOfHash is whether a live Torrent of hash is on any Node: one a Grab
// at fetching takes once Reconciling ends.
func LiveOfHash(ctx context.Context, r storage.Reader[file], hash bencode.InfoHash) (bool, error) {
	held, err := db.New(r).ListHeldOfHash(ctx, hash[:])
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(held, func(t db.ListHeldOfHashRow) bool { return t.RemovedAt.IsZero() }), nil
}

// Forget removes each Torrent of ids still on the review list, so its Grab
// fails restored. No Node holds it, so it parts at once.
func Forget(ctx context.Context, tx *storage.Tx[file], ids []ID) error {
	q := db.New(tx)
	rows, err := readActed(ctx, tx, ids)
	if err != nil {
		return err
	}
	now := storage.Now()
	for _, id := range ids {
		row, ok := rows[id]
		if !ok || !row.RemovedAt.IsZero() {
			continue
		}
		if err := q.RemoveParted(ctx, db.RemovePartedParams{At: now, RemovedReason: new(string(ByAdmin)), ID: id}); err != nil {
			return fmt.Errorf("forget Torrent %s: %w", id, err)
		}
	}
	return nil
}

// LookFile is one of a Torrent's files: its path under its save folder,
// with / between parts, and its size.
type LookFile struct {
	Name string
	Size int64
}

// Look is where a Torrent's files should be on its Node: its save folder,
// "" when none is recorded, and each file.
type Look struct {
	Node   role.NodeID
	Folder string
	Files  []LookFile
}

// LookOf is where Torrent id's files should be, for the review list's
// look at what its Node still has.
func LookOf(ctx context.Context, r storage.Reader[file], id ID) (Look, error) {
	row, err := db.New(r).GetLook(ctx, id)
	if err != nil {
		return Look{}, fmt.Errorf("read where a Torrent's files are: %w", err)
	}
	m, err := bencode.DecodeMetainfo(row.Metainfo)
	if err != nil {
		return Look{}, fmt.Errorf("read where a Torrent's files are: %w", err)
	}
	l := Look{Node: row.NodeID}
	if row.SaveFolder != nil {
		l.Folder = *row.SaveFolder
	}
	for _, f := range m.Info.Files {
		l.Files = append(l.Files, LookFile{Name: path.Join(append([]string{m.Info.Name}, f.Path...)...), Size: f.Length})
	}
	return l, nil
}
