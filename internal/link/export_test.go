package link

import (
	"context"
	"net"

	"github.com/jatassi/slipstream/internal/link/internal/wire"
)

// OpenStream opens a stream of kind on key's live session, as S09's
// Transfers will.
func (l *Link) OpenStream(ctx context.Context, key Key, kind byte) (net.Conn, error) {
	l.mu.Lock()
	s := l.live[key]
	l.mu.Unlock()
	if s == nil {
		return nil, ErrDisconnected
	}
	return wire.Open(ctx, s.mux, wire.Kind(kind))
}
