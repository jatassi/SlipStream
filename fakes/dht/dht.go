// Package dht is fake-dht: SlipStream's own DHT node speaking KRPC (BEP 5)
// over UDP, with one socket per origin. It answers ping, find_node with no
// nodes, get_peers with the peers the caller lists for that origin and
// hash, and announce_peer, and records every query it receives.
package dht

import (
	"crypto/sha1" //nolint:gosec // a node ID only needs to be 20 bytes that differ per origin
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/jatassi/slipstream/internal/bencode"
)

// Query is one query a socket received. InfoHash is set for get_peers
// and announce_peer, and Port for announce_peer: the port it announced,
// or the one it sent from when it asked for that.
type Query struct {
	Origin   string           `json:"origin"`
	Method   string           `json:"method"`
	From     netip.AddrPort   `json:"from"`
	InfoHash bencode.InfoHash `json:"infoHash"`
	Port     uint16           `json:"port,omitempty"`
}

// Config is what a Server answers and where it records.
type Config struct {
	// Peers is the peers get_peers answers for h when asked through
	// origin's socket.
	Peers func(origin string, h bencode.InfoHash) []netip.AddrPort
	// Record receives every query, from the socket's own goroutine.
	Record func(Query)
}

type Server struct {
	cfg   Config
	mu    sync.Mutex
	conns []*net.UDPConn
	wg    sync.WaitGroup
}

func New(cfg Config) *Server { return &Server{cfg: cfg} }

// Listen opens origin's socket on addr and returns the address it bound.
func (s *Server) Listen(origin string, addr netip.AddrPort) (netip.AddrPort, error) {
	conn, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(addr))
	if err != nil {
		return netip.AddrPort{}, fmt.Errorf("dht: listen for %s: %w", origin, err)
	}
	s.mu.Lock()
	s.conns = append(s.conns, conn)
	s.mu.Unlock()
	s.wg.Go(func() { s.serve(origin, conn) })
	return netip.MustParseAddrPort(conn.LocalAddr().String()), nil
}

// Close closes every socket and waits for each to stop.
func (s *Server) Close() error {
	s.mu.Lock()
	var errs []error
	for _, c := range s.conns {
		errs = append(errs, c.Close())
	}
	s.conns = nil
	s.mu.Unlock()
	s.wg.Wait()
	return errors.Join(errs...)
}

func (s *Server) serve(origin string, conn *net.UDPConn) {
	id := nodeID(origin)
	buf := make([]byte, 64<<10)
	for {
		n, from, err := conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			return
		}
		res, ok := s.answer(origin, id, from, buf[:n])
		if !ok {
			continue
		}
		out, err := bencode.Encode(res)
		if err != nil {
			panic(fmt.Sprintf("dht: encode a response: %v", err))
		}
		_, _ = conn.WriteToUDPAddrPort(out, from)
	}
}

// KRPC error codes.
const (
	protocolError = 203
	methodUnknown = 204
)

// answer is the response to the message msg from from, false for a
// message that is no query, which KRPC never answers.
func (s *Server) answer(origin, id string, from netip.AddrPort, msg []byte) (map[string]any, bool) {
	v, err := bencode.Decode(msg)
	m, isDict := v.(map[string]any)
	if err != nil || !isDict || m["y"] != "q" {
		return nil, false
	}
	t, _ := m["t"].(string)
	method, _ := m["q"].(string)
	args, _ := m["a"].(map[string]any)
	q := Query{Origin: origin, Method: method, From: from}
	r := map[string]any{"id": id}
	switch method {
	case "ping":
	case "find_node":
		r["nodes"] = ""
	case "get_peers", "announce_peer":
		h, ok := infoHash(args)
		if !ok {
			return failure(t, protocolError, "info_hash isn't 20 bytes"), true
		}
		q.InfoHash = h
		if method == "announce_peer" {
			q.Port = announcedPort(args, from)
			break
		}
		r["token"] = id[:4]
		r["nodes"] = ""
		if peers := s.cfg.Peers(origin, h); len(peers) > 0 {
			r["values"] = compact(peers)
		}
	default:
		s.cfg.Record(q)
		return failure(t, methodUnknown, "Method Unknown"), true
	}
	s.cfg.Record(q)
	return map[string]any{"t": t, "y": "r", "r": r}, true
}

func failure(t string, code int, text string) map[string]any {
	return map[string]any{"t": t, "y": "e", "e": []any{code, text}}
}

func infoHash(args map[string]any) (bencode.InfoHash, bool) {
	var h bencode.InfoHash
	s, ok := args["info_hash"].(string)
	if !ok || len(s) != len(h) {
		return h, false
	}
	copy(h[:], s)
	return h, true
}

func announcedPort(args map[string]any, from netip.AddrPort) uint16 {
	if implied, _ := args["implied_port"].(int64); implied != 0 {
		return from.Port()
	}
	port, _ := args["port"].(int64)
	return uint16(port) //nolint:gosec // a port a peer sent, kept as it reads
}

// compact is BEP 5's compact peer info: 6 bytes for an IPv4 address and
// port, 18 for IPv6.
func compact(peers []netip.AddrPort) []any {
	out := make([]any, len(peers))
	for i, p := range peers {
		ip := p.Addr().Unmap().AsSlice()
		out[i] = string(binary.BigEndian.AppendUint16(ip, p.Port()))
	}
	return out
}

func nodeID(origin string) string {
	sum := sha1.Sum([]byte("fake-dht " + origin)) //nolint:gosec // see the import
	return string(sum[:])
}
