package nodes

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

var ErrNodeKeyMismatch = errors.New("the Node address names another Node's key")

// DialBy is which side dialed a Node's live session.
type DialBy uint8

const (
	NoSession DialBy = iota
	ByNode
	ByController
)

func (d DialBy) String() string { return [...]string{"none", "node", "controller"}[d] }

// Dial is how a Node's session comes up: who dialed the live one, and
// the addresses the Controller dials it at. ControllerDials is empty for
// a Node that dials the Controller.
type Dial struct {
	By              DialBy
	ControllerDials []netip.AddrPort
}

// SetDial has the Controller dial node at addr's addresses from now on,
// or stop dialing it when addr is nil. addr must name node's key.
func (s *Service) SetDial(ctx context.Context, id ID, addr *joincode.NodeAddress) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		row, err := q.GetNode(ctx, id)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var addrs []netip.AddrPort
		if addr != nil {
			if !slices.Equal(addr.Key(), row.PublicKey) {
				return ErrNodeKeyMismatch
			}
			if _, err := targetOf(*addr); err != nil {
				return err
			}
			addrs = addr.Addresses()
		}
		return writeDials(ctx, q, id, addrs)
	})
	if err != nil {
		return fmt.Errorf("set how the Controller dials node %s: %w", id, err)
	}
	s.reconcileDials(ctx)
	s.changed(id)
	return nil
}

func writeDials(ctx context.Context, q *db.Queries, id ID, addrs []netip.AddrPort) error {
	if err := q.DeleteDials(ctx, id); err != nil {
		return err
	}
	for i, a := range addrs {
		if err := q.InsertDial(ctx, db.InsertDialParams{NodeID: id, Position: int64(i), Address: a.String()}); err != nil {
			return err
		}
	}
	return nil
}

// reconcileDials keeps every Node nodes_dial names at its addresses and
// drops every Node it no longer names. Run calls it on each link change,
// so a Node paired over an invited session is kept once that session is
// live, and the keeper waits on it instead of dialing a second.
func (s *Service) reconcileDials(ctx context.Context) {
	want, err := s.readDials(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.cfg.Log.Error("read the Nodes the Controller dials", "err", err.Error())
		}
		return
	}
	s.dmu.Lock()
	defer s.dmu.Unlock()
	for key, t := range want {
		if held, ok := s.kept[key]; !ok || held != t {
			s.cfg.Link.Keep(t)
		}
	}
	for key := range s.kept {
		if _, ok := want[key]; !ok {
			s.cfg.Link.Drop(key)
		}
	}
	s.kept = want
}

func (s *Service) readDials(ctx context.Context) (map[link.Key]link.Target, error) {
	rows, err := db.New(s.cfg.Library).ListDials(ctx)
	if err != nil {
		return nil, err
	}
	byNode := map[ID][]string{}
	for _, r := range rows {
		byNode[r.NodeID] = append(byNode[r.NodeID], r.Address)
	}
	want := make(map[link.Key]link.Target, len(byNode))
	for id, held := range byNode {
		key, ok := s.Key(id)
		if !ok {
			continue
		}
		addrs, err := addressRecord(id, held)
		if err != nil {
			return nil, err
		}
		want[key] = link.Target{Key: key, Addresses: addrs}
	}
	return want, nil
}

func (s *Service) dialOf(ctx context.Context, row db.NodesNode, key link.Key) (Dial, error) {
	held, err := db.New(s.cfg.Library).NodeDials(ctx, row.ID)
	if err != nil {
		return Dial{}, err
	}
	addrs, err := addressRecord(row.ID, held)
	if err != nil {
		return Dial{}, err
	}
	d := Dial{ControllerDials: addrs.All()}
	if up, ok := s.cfg.Link.State(key).(link.Live); ok {
		d.By = ByNode
		if up.Dialed {
			d.By = ByController
		}
	}
	return d, nil
}

// pasted is the Node address an Admin pasted for code, when it names peer.
func pasted(code db.NodesJoinCode, peer link.Key) []netip.AddrPort {
	if code.DialAddress == nil {
		return nil
	}
	addr, err := joincode.ParseNodeAddress(*code.DialAddress)
	if err != nil || !slices.Equal(addr.Key(), peer[:]) {
		return nil
	}
	return addr.Addresses()
}
