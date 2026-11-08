package tracker

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
)

// UDPAnnounces is the capture of every BEP 15 announce, in
// tracker.udp-announce records whose node is the origin of the socket it
// came to. They carry no passkey.
var UDPAnnounces = fake.CaptureSpec{Name: "udp-announces", Kind: "tracker.udp-announce"}

// UDP is fake-tracker's BEP 15 announce, which public releases name. It
// is a Spec of its own, since a Spec serves TCP or UDP, and it answers
// from fake-tracker's swarm, so the world serves it after Fake.
var UDP = fake.Spec{
	Name:      "tracker-udp",
	Scheme:    "udp",
	NewUDP:    newUDPServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("public trackers get fixtures only, as private ones do, #26 Q12"),
	Captures:  []fake.CaptureSpec{UDPAnnounces},
}

// BEP 15's actions, and the protocol ID a connect request carries.
const (
	udpProtocolID  = 0x41727101980
	actionConnect  = 0
	actionAnnounce = 1
	actionError    = 3
)

// udpEvents are BEP 15's announce events, by their number on the wire.
var udpEvents = []string{"", "completed", "started", "stopped"}

// announceSize is a BEP 15 announce request before its BEP 41 options.
const announceSize = 98

// udpServer takes an announce on any connection ID: libtorrent keeps the
// IDs it was given in one cache per process keyed by the tracker's IP, so
// a session announces to one loopback tracker with an ID another gave.
type udpServer struct {
	srv       *server
	announces *fake.Capture
	ids       *rand.Rand
}

func newUDPServer(_ scenario.Scenario, env fake.Env) (fake.UDPServer, error) {
	shared, ok := env.Shared.Get(sharedSwarm)
	if !ok {
		return nil, errors.New("tracker-udp needs fake-tracker's swarm, and the world built no fake-tracker before it")
	}
	announces, err := env.Capture(UDPAnnounces.Name)
	if err != nil {
		return nil, err
	}
	srv, _ := shared.(*server)
	return &udpServer{srv: srv, announces: announces, ids: env.Streams.Stream("connection-ids")}, nil
}

func (u *udpServer) ServeUDP(ctx context.Context, sockets map[string]*net.UDPConn) error {
	var wg sync.WaitGroup
	for origin, conn := range sockets {
		wg.Go(func() { u.serve(fake.WithOrigin(ctx, origin), conn) })
	}
	<-ctx.Done()
	var errs []error
	for _, conn := range sockets {
		errs = append(errs, conn.Close())
	}
	wg.Wait()
	return errors.Join(errs...)
}

func (u *udpServer) serve(ctx context.Context, conn *net.UDPConn) {
	buf := make([]byte, 2048)
	for {
		n, from, err := conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			return
		}
		if reply := u.answer(ctx, buf[:n], time.Now()); reply != nil {
			_, _ = conn.WriteToUDPAddrPort(reply, from)
		}
	}
}

// answer is the reply to the datagram p, nil for one that is neither a
// connect nor an announce.
func (u *udpServer) answer(ctx context.Context, p []byte, now time.Time) []byte {
	if len(p) < 16 {
		return nil
	}
	conn, action, tx := binary.BigEndian.Uint64(p), binary.BigEndian.Uint32(p[8:]), p[12:16]
	switch {
	case action == actionConnect && conn == udpProtocolID:
		return binary.BigEndian.AppendUint64(udpHeader(actionConnect, tx), u.ids.Uint64())
	case action == actionAnnounce && len(p) >= announceSize:
		return u.announce(ctx, p, tx, now)
	}
	return nil
}

// announce answers the announce request p with the interval, the swarm
// counts and the peers fake-tracker's HTTPS announce gives, and records
// it.
func (u *udpServer) announce(ctx context.Context, p, tx []byte, now time.Time) []byte {
	var hash bencode.InfoHash
	copy(hash[:], p[16:36])
	event := binary.BigEndian.Uint32(p[80:])
	if int(event) >= len(udpEvents) {
		return udpError(tx, "unknown event")
	}
	a := Announce{
		InfoHash:   hash.String(),
		PeerID:     hex.EncodeToString(p[36:56]),
		Downloaded: int64(binary.BigEndian.Uint64(p[56:])), //nolint:gosec // a peer's byte count
		Left:       int64(binary.BigEndian.Uint64(p[64:])), //nolint:gosec // a peer's byte count
		Uploaded:   int64(binary.BigEndian.Uint64(p[72:])), //nolint:gosec // a peer's byte count
		Event:      udpEvents[event],
		Port:       binary.BigEndian.Uint16(p[96:]),
	}
	s := u.srv
	s.mu.Lock()
	defer s.mu.Unlock()
	a.Rule, a.Failure = s.judgeTorrent(hash)
	a.Accepted = a.Rule == ""
	u.announces.Record(ctx, a)
	if !a.Accepted {
		return udpError(tx, a.Failure)
	}
	origin := fake.OriginOf(ctx)
	s.accept(a, hash, origin, now)
	complete, incomplete := s.swarm(hash, now)
	out := udpHeader(actionAnnounce, tx)
	for _, v := range []int64{int64(s.interval / time.Second), incomplete, complete} {
		out = binary.BigEndian.AppendUint32(out, uint32(v)) //nolint:gosec // an interval and peer counts fit
	}
	return append(out, s.peerFronts(origin, s.torrents[hash].release, a)...)
}

func udpHeader(action uint32, tx []byte) []byte {
	return append(binary.BigEndian.AppendUint32(nil, action), tx...)
}

func udpError(tx []byte, message string) []byte {
	return append(udpHeader(actionError, tx), message...)
}
