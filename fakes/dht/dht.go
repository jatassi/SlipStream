// Package dht is fake-dht: the Fake world's DHT, a KRPC node with one UDP
// socket per origin, which each Node's endpoints.torrent.dht_routers
// names. A get_peers through a Node's socket gets that Node's seeder
// fronts for the public seeded releases of the hash, so DHT peers cross
// fronts as fake-tracker's peers do, and every query is a record of its
// lookups capture.
package dht

import (
	"context"
	"net"
	"net/netip"

	"github.com/jatassi/slipstream/fakes/dht/krpc"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

// Lookups is the capture of every query, in dht.query records whose node
// is the origin of the socket it arrived on.
var Lookups = fake.CaptureSpec{Name: "lookups", Kind: "dht.query"}

var Fake = fake.Spec{
	Name:   "dht",
	Scheme: "udp",
	NewUDP: newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.Torrent.DHTRouters = []endpoint.HostPort{endpoint.MustParseHostPort(at.AddrPort.String())}
	},
	Online:   fake.NoOnlineRun("the public DHT holds no torrent of the Fake world's, and libtorrent's contract cases run against fake-dht"),
	Captures: []fake.CaptureSpec{Lookups},
}

// Query is one query as its record holds it: InfoHash for get_peers and
// announce_peer, and Port, the port announced, for announce_peer.
type Query struct {
	Method   string `json:"method"`
	InfoHash string `json:"infoHash,omitempty"`
	From     string `json:"from"`
	Port     uint16 `json:"port,omitempty"`
}

type server struct {
	fronts  fake.Fronts
	seeders map[bencode.InfoHash]fault.Target
	lookups *fake.Capture
}

func newServer(_ scenario.Scenario, env fake.Env) (fake.UDPServer, error) {
	lookups, err := env.Capture(Lookups.Name)
	if err != nil {
		return nil, err
	}
	s := &server{fronts: env.Fronts, seeders: map[bencode.InfoHash]fault.Target{}, lookups: lookups}
	for _, r := range env.Releases.Releases {
		if r.Public && r.Seeded {
			s.seeders[r.InfoHash] = tracker.SeedersTarget(r.Name)
		}
	}
	return s, nil
}

func (s *server) ServeUDP(ctx context.Context, sockets map[string]*net.UDPConn) error {
	n := krpc.New(krpc.Config{
		Peers: s.peers,
		Record: func(q krpc.Query) {
			rec := Query{Method: q.Method, From: q.From.String(), Port: q.Port}
			if q.Method == "get_peers" || q.Method == "announce_peer" {
				rec.InfoHash = q.InfoHash.String()
			}
			s.lookups.Record(fake.WithOrigin(ctx, q.Origin), rec)
		},
	})
	for origin, conn := range sockets {
		n.Serve(origin, conn)
	}
	<-ctx.Done()
	return n.Close() //nolint:wrapcheck // closing the sockets has nothing to add
}

// peers is origin's seeder front of the public seeded release of h.
func (s *server) peers(origin string, h bencode.InfoHash) []netip.AddrPort {
	target, ok := s.seeders[h]
	if !ok {
		return nil
	}
	at, ok := s.fronts.Find(target, origin)
	if !ok {
		return nil
	}
	return []netip.AddrPort{at.AddrPort}
}
