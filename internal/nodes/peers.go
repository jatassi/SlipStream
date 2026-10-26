package nodes

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Peer is a Node as another Node reaches it directly: its name, its key,
// and the addresses to dial it on, the Admin's reverse-dial addresses
// first, then the link addresses its Hello last advertised. A removed
// Node keeps its name and has no key.
type Peer struct {
	Name      string
	Key       link.Key
	Addresses []string
	Removed   bool
}

// PeersIn is every Node in rd, removed ones included, as a peer reaches
// it.
func PeersIn(ctx context.Context, rd storage.Reader[file]) (map[ID]Peer, error) {
	q := db.New(rd)
	names, err := q.ListNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the Nodes' names: %w", err)
	}
	out := make(map[ID]Peer, len(names))
	for _, n := range names {
		out[n.ID] = Peer{Name: n.Name, Removed: !n.RemovedAt.IsZero()}
	}
	live, err := q.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the Nodes: %w", err)
	}
	for _, row := range live {
		key, err := link.KeyFromBytes(row.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", row.ID, err)
		}
		p := out[row.ID]
		p.Key = key
		out[row.ID] = p
	}
	dials, err := q.ListDials(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the Nodes' dial addresses: %w", err)
	}
	addresses, err := q.ListAddresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the Nodes' link addresses: %w", err)
	}
	for _, a := range append(dialAddresses(dials), addresses...) {
		if p, ok := out[a.NodeID]; ok && !p.Removed {
			p.Addresses = append(p.Addresses, a.Address)
			out[a.NodeID] = p
		}
	}
	return out, nil
}

func dialAddresses(dials []db.NodesDial) []db.NodesAddress {
	out := make([]db.NodesAddress, len(dials))
	for i, d := range dials {
		out[i] = db.NodesAddress(d)
	}
	return out
}
