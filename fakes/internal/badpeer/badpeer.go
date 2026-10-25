// Package badpeer is the bad peer of a release: a BitTorrent peer on
// 127.0.0.1 that claims to hold every piece, then misbehaves as its
// Scenario says once a leecher asks for them.
package badpeer

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/netip"
	"sync"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
)

const protocol = "\x13BitTorrent protocol"

// peerID is the peer ID every bad peer answers with.
var peerID = []byte("-BP0001-badpeerbadpe")

// The BEP 3 messages a bad peer reads or sends.
const (
	msgUnchoke  = 1
	msgBitfield = 5
	msgRequest  = 6
	msgPiece    = 7
)

// maxMessage bounds a message a leecher sends: a bitfield of a large
// torrent, far more than any request.
const maxMessage = 1 << 20

// junk fills every block a corrupt peer sends. A piece of it never matches
// a piece's hash.
const junk = 0xba

// Peer serves one torrent until Close.
type Peer struct {
	hash     bencode.InfoHash
	bitfield []byte
	mode     scenario.BadPeer
	l        net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// Listen starts a bad peer of the torrent m on 127.0.0.1.
func Listen(ctx context.Context, m bencode.Metainfo, mode scenario.BadPeer) (*Peer, error) {
	l, err := new(net.ListenConfig).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("bad peer: %w", err)
	}
	pieces := len(m.Info.Pieces) / 20
	bitfield := bytes.Repeat([]byte{0xff}, (pieces+7)/8)
	if spare := pieces % 8; spare != 0 {
		bitfield[len(bitfield)-1] = byte(0xff << (8 - spare))
	}
	p := &Peer{hash: m.InfoHash, bitfield: bitfield, mode: mode, l: l}
	p.ctx, p.cancel = context.WithCancel(context.WithoutCancel(ctx))
	p.wg.Go(p.accept)
	return p, nil
}

// Addr is where the peer listens.
func (p *Peer) Addr() netip.AddrPort { return netip.MustParseAddrPort(p.l.Addr().String()) }

// Close stops listening and hangs up on every leecher.
func (p *Peer) Close() error {
	p.cancel()
	err := p.l.Close()
	p.wg.Wait()
	if err != nil {
		return fmt.Errorf("bad peer: %w", err)
	}
	return nil
}

func (p *Peer) accept() {
	for {
		c, err := p.l.Accept()
		if err != nil {
			return
		}
		p.wg.Go(func() {
			stop := context.AfterFunc(p.ctx, func() { _ = c.Close() })
			defer stop()
			defer func() { _ = c.Close() }()
			p.serve(c)
		})
	}
}

// serve answers a leecher's handshake for the peer's torrent and its
// bitfield, then misbehaves until the leecher hangs up. An encrypted
// handshake gets a hang up, so the leecher dials again in the clear.
func (p *Peer) serve(c net.Conn) {
	var theirs [68]byte
	if _, err := io.ReadFull(c, theirs[:]); err != nil {
		return
	}
	if string(theirs[:len(protocol)]) != protocol || !bytes.Equal(theirs[28:48], p.hash[:]) {
		return
	}
	hello := append(append(append([]byte(protocol), make([]byte, 8)...), p.hash[:]...), peerID...)
	hello = append(hello, message(msgBitfield, p.bitfield)...)
	if p.mode == scenario.Stall || p.mode == scenario.Corrupt {
		hello = append(hello, message(msgUnchoke, nil)...)
	}
	if _, err := c.Write(hello); err != nil || p.mode == scenario.Drop {
		return
	}
	for {
		id, payload, ok := read(c)
		if !ok {
			return
		}
		if id != msgRequest || p.mode != scenario.Corrupt || len(payload) != 12 {
			continue
		}
		length := binary.BigEndian.Uint32(payload[8:])
		if length > maxMessage {
			return
		}
		block := make([]byte, 8, 8+length)
		copy(block, payload[:8])
		block = append(block, bytes.Repeat([]byte{junk}, int(length))...)
		if _, err := c.Write(message(msgPiece, block)); err != nil {
			return
		}
	}
}

func message(id byte, payload []byte) []byte {
	m := binary.BigEndian.AppendUint32(nil, uint32(1+len(payload))) //nolint:gosec // a bitfield or a block fits
	return append(append(m, id), payload...)
}

// read is the next message but a keep-alive, not ok once the leecher hangs
// up or sends a message too long for any it may send.
func read(r io.Reader) (id byte, payload []byte, ok bool) {
	for {
		var size [4]byte
		if _, err := io.ReadFull(r, size[:]); err != nil {
			return 0, nil, false
		}
		n := binary.BigEndian.Uint32(size[:])
		if n == 0 {
			continue
		}
		if n > maxMessage {
			return 0, nil, false
		}
		m := make([]byte, n)
		if _, err := io.ReadFull(r, m); err != nil {
			return 0, nil, false
		}
		return m[0], m[1:], true
	}
}
