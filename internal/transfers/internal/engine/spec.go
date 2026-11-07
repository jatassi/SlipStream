package engine

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// spec is one active Transfer as this Node runs it: an intent's Active,
// parsed.
type spec struct {
	id    ID
	side  tv1.Side
	peer  link.Key
	addrs link.Addresses
	route tv1.Route
	// root and place are the receiver's destination.
	root  storage.UUID
	place int64
	files []file
}

type file struct {
	position int
	// wire is the source as Open names it.
	wire *tv1.Source
	ref  source.Ref
}

// sameRun reports whether o runs as s does: the same Transfer, side, peer
// and destination. A route change is handed to the run; a shorter file
// list is the Controller catching up on files already reported.
func (s spec) sameRun(o spec) bool {
	return s.id == o.id && s.side == o.side && s.peer == o.peer && s.addrs == o.addrs && s.root == o.root && s.place == o.place
}

func (s spec) file(position int, src *tv1.Source) (file, bool) {
	i := slices.IndexFunc(s.files, func(f file) bool { return f.position == position })
	if i < 0 || !proto.Equal(s.files[i].wire, src) {
		return file{}, false
	}
	return s.files[i], true
}

func parseActive(a *tv1.Active) (spec, error) {
	u, err := storage.UUIDFromBytes(a.GetTransferId())
	if err != nil {
		return spec{}, fmt.Errorf("transfer id: %w", err)
	}
	s := spec{id: ID{UUID: u}, side: a.GetSide(), route: a.GetRoute()}
	if s.side != tv1.Side_SIDE_SEND && s.side != tv1.Side_SIDE_RECEIVE {
		return spec{}, fmt.Errorf("transfer %s: side %s", s.id, s.side)
	}
	if s.route < tv1.Route_ROUTE_RECEIVER_DIALS || s.route > tv1.Route_ROUTE_BOTH_DIAL {
		return spec{}, fmt.Errorf("transfer %s: route %s", s.id, s.route)
	}
	if len(a.GetPeer().GetKey()) != len(link.Key{}) {
		return spec{}, fmt.Errorf("transfer %s: a peer key of %d bytes", s.id, len(a.GetPeer().GetKey()))
	}
	s.peer = link.Key(a.GetPeer().GetKey())
	if s.addrs, err = parseAddresses(a.GetPeer().GetAddresses()); err != nil {
		return spec{}, fmt.Errorf("transfer %s: %w", s.id, err)
	}
	if s.side == tv1.Side_SIDE_RECEIVE {
		root := a.GetDestination().GetRoot()
		if root == nil {
			return spec{}, fmt.Errorf("transfer %s: no root destination", s.id)
		}
		if s.root, err = storage.UUIDFromBytes(root.GetRootId()); err != nil {
			return spec{}, fmt.Errorf("transfer %s: root id: %w", s.id, err)
		}
		s.place = root.GetPlace()
	}
	for _, f := range a.GetFiles() {
		ref, err := parseSource(f.GetSource())
		if err != nil {
			return spec{}, fmt.Errorf("transfer %s file %d: %w", s.id, f.GetPosition(), err)
		}
		s.files = append(s.files, file{position: int(f.GetPosition()), wire: f.GetSource(), ref: ref})
	}
	return s, nil
}

func parseAddresses(in []string) (link.Addresses, error) {
	if len(in) == 0 {
		return link.Addresses{}, nil
	}
	aps := make([]netip.AddrPort, len(in))
	for i, a := range in {
		ap, err := netip.ParseAddrPort(a)
		if err != nil {
			return link.Addresses{}, fmt.Errorf("peer address: %w", err)
		}
		aps[i] = ap
	}
	return link.ParseAddresses(aps)
}

func parseSource(s *tv1.Source) (source.Ref, error) {
	t := s.GetTorrent()
	if t == nil {
		return nil, errors.New("a source of no kind this build reads")
	}
	if len(t.GetInfoHash()) != len(bencode.InfoHash{}) {
		return nil, fmt.Errorf("an info hash of %d bytes", len(t.GetInfoHash()))
	}
	return source.Torrent{Hash: bencode.InfoHash(t.GetInfoHash()), File: int(t.GetFileIndex())}, nil
}
