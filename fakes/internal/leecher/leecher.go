// Package leecher is a peer from outside that connects in to a Node's
// listen port and says hello for a torrent, as a leecher a tracker sent
// there does. libtorrent reports it as an incoming peer, which is what the
// listen-port check counts.
package leecher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"

	"github.com/jatassi/slipstream/internal/bencode"
)

const protocol = "\x13BitTorrent protocol"

// peerID is the peer ID every leecher says hello with.
var peerID = []byte("-LE0001-leecherleech")

// Hello connects to addr, sends a BEP 3 handshake for hash and reads the
// peer's own. It succeeds once the peer at addr answered for hash, which a
// libtorrent session does only while it runs that torrent.
func Hello(ctx context.Context, addr netip.AddrPort, hash bencode.InfoHash) error {
	conn, err := new(net.Dialer).DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return fmt.Errorf("leecher: %w", err)
	}
	defer func() { _ = conn.Close() }()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	hello := make([]byte, 0, 68)
	hello = append(hello, protocol...)
	hello = append(hello, make([]byte, 8)...)
	hello = append(hello, hash[:]...)
	hello = append(hello, peerID...)
	if _, err := conn.Write(hello); err != nil {
		return fmt.Errorf("leecher: say hello: %w", err)
	}
	answer := make([]byte, len(hello))
	if _, err := io.ReadFull(conn, answer); err != nil {
		return fmt.Errorf("leecher: %s did not answer for %s: %w", addr, hash, errors.Join(err, ctx.Err()))
	}
	if !bytes.HasPrefix(answer, []byte(protocol)) || !bytes.Equal(answer[28:48], hash[:]) {
		return fmt.Errorf("leecher: %s answered with no handshake for %s", addr, hash)
	}
	return nil
}
