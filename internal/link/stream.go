package link

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/link/internal/mux"
	"github.com/jatassi/slipstream/internal/link/internal/wire"
)

// Kind names what a stream carries. The link serves kind 1, its RPC, itself.
type Kind byte

const KindTransfer Kind = 2

// Stream is one stream of a session. Reset abandons it in both directions.
type Stream = mux.Stream

// OpenStream opens a stream of kind on key's live session. It is lockstep
// work, so it fails before sending with ErrDisconnected, or ErrMismatched
// when the peer runs another version.
func (l *Link) OpenStream(ctx context.Context, key Key, kind Kind) (Stream, error) {
	l.mu.Lock()
	s := l.live[key]
	l.mu.Unlock()
	switch {
	case s == nil:
		return nil, ErrDisconnected
	case !s.matched:
		return nil, ErrMismatched
	}
	st, err := wire.Open(ctx, s.mux, wire.Kind(kind))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDisconnected, err)
	}
	return st, nil
}

// HandleStream serves kind with h, given the peer's key. The link resets a
// stream from a session that isn't live with a peer on this version before
// h sees it. Call it before Serve and before any dial.
func (l *Link) HandleStream(kind Kind, h func(peer Key, s Stream)) {
	k := wire.Kind(kind)
	if k == wire.RPC {
		panic("link: kind 1 is the link's own RPC")
	}
	if l.kinds[k] != nil {
		panic(fmt.Sprintf("link: kind %d already has a handler", kind))
	}
	l.kinds[k] = func(s *session, st Stream) {
		if l.inStep(l.ctx, s) != nil {
			_ = st.Reset()
			return
		}
		h(s.key, st)
	}
}
