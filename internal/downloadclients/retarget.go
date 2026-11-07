package downloadclients

import (
	"context"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// InClientError refuses sending a Torrent its client already holds to
// another client on the same Node, which holds one live copy of a
// torrent. Client names that client, "" for the Node's built-in one.
type InClientError struct{ Client, Node string }

func (e *InClientError) Error() string {
	if e.Client == "" {
		return e.Node + "'s built-in client already has the torrent"
	}
	return e.Client + " already has the torrent"
}

// Retargeted is the Torrent a Retarget leaves its Grabs on and that
// Torrent's Node, with the Nodes whose intents changed and the Torrents
// that did.
type Retargeted struct {
	ID      ID
	Node    role.NodeID
	Touched []role.NodeID
	Changed []ID
}

// Retarget sends live Torrent id to to, in tx. Before its Node reports it,
// no client holds it, so it moves to to's client and Node in place with
// no removal. Once reported, a target on the same Node is *InClientError,
// and one on another Node removes the Torrent as an Admin's remove does,
// files kept, and takes to.Node's live Torrent of the hash or adds one
// there from the same metainfo. A Torrent of the hash Removing on to.Node
// is ErrRemoving. A Torrent already at to stays as it is.
func Retarget(ctx context.Context, tx *storage.Tx[file], id ID, to Target, now storage.Time) (Retargeted, error) {
	q := db.New(tx)
	t, err := q.GetRetarget(ctx, id)
	if err != nil {
		return Retargeted{}, err
	}
	stay := Retargeted{ID: id, Node: t.NodeID}
	if t.NodeID == to.Node && sameClient(t.ClientID, to.Client) {
		return stay, nil
	}
	moved := Retargeted{Node: to.Node, Touched: []role.NodeID{t.NodeID, to.Node}, Changed: []ID{id}}
	there, err := heldOn(ctx, q, t.InfoHash, to.Node, id)
	if err != nil {
		return Retargeted{}, err
	}
	reported := !t.ReportedAt.IsZero()
	switch {
	case there == nil && !reported:
		moved.ID = id
		return moved, q.Rehome(ctx, db.RehomeParams{NodeID: to.Node, ClientID: to.Client, ID: id})
	case t.NodeID == to.Node:
		return Retargeted{}, inClient(ctx, tx, id)
	case there != nil && !there.RemovedAt.IsZero():
		return Retargeted{}, ErrRemoving
	}
	rows, err := readActed(ctx, tx, []ID{id})
	if err != nil {
		return Retargeted{}, err
	}
	if err := removeIn(ctx, q, rows[id], ByAdmin, false, now); err != nil {
		return Retargeted{}, err
	}
	m, err := bencode.DecodeMetainfo(t.Metainfo)
	if err != nil {
		return Retargeted{}, err
	}
	want := Want{Target: to, Metainfo: m, Raw: t.Metainfo, ContentType: t.ContentType}
	if t.TrackerID != nil {
		want.Tracker = *t.TrackerID
	}
	if there != nil {
		moved.ID = there.ID
		moved.Changed = append(moved.Changed, there.ID)
		return moved, reached(ctx, q, there.ID, want)
	}
	moved.ID = ID{UUID: storage.NewUUID()}
	moved.Changed = append(moved.Changed, moved.ID)
	return moved, insert(ctx, q, moved.ID, want, now)
}

func sameClient(a, b *ClientID) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

// heldOn is the Torrent of hash node holds, live or Removing, other than
// not, nil for none.
func heldOn(ctx context.Context, q *db.Queries, hash []byte, node role.NodeID, not ID) (*db.ListHeldOfHashRow, error) {
	held, err := q.ListHeldOfHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	for i, h := range held {
		if h.NodeID == node && h.ID != not {
			return &held[i], nil
		}
	}
	return nil, nil
}

// inClient is the *InClientError for Torrent id, named as its row reads.
func inClient(ctx context.Context, r storage.Reader[file], id ID) error {
	rows, err := readActed(ctx, r, []ID{id})
	if err != nil {
		return err
	}
	names, err := namesIn(ctx, r)
	if err != nil {
		return err
	}
	return &InClientError{Client: rows[id].ClientName, Node: names[rows[id].NodeID]}
}

// Retargeted tells the readers once a Retarget's Write commits: the
// Nodes' intents and the Torrents it changed.
func (c *Clients) Retargeted(r Retargeted) { c.committed(r.Touched, r.Changed) }
