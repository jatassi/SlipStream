package link

import (
	"context"
	"net/http"

	"connectrpc.com/connect/v2"

	"github.com/jatassi/slipstream/internal/link/internal/wire"
)

// RawStream opens a stream of kind on key's live session whatever the
// peer's version, as a peer that skipped OpenStream's check could.
func (l *Link) RawStream(ctx context.Context, key Key, kind Kind) (Stream, error) {
	l.mu.Lock()
	s := l.live[key]
	l.mu.Unlock()
	return wire.Open(ctx, s.mux, wire.Kind(kind))
}

// UnwelcomedStream dials t and opens a stream of kind over that session
// with no Pair or Hello.
func (l *Link) UnwelcomedStream(ctx context.Context, t Target, kind Kind) (Stream, error) {
	s, err := l.race(ctx, t.Addresses, func(Key) error { return nil })
	if err != nil {
		return nil, err
	}
	return wire.Open(ctx, s.mux, wire.Kind(kind))
}

// UnwelcomedClient dials t and calls over that session with no Pair or
// Hello: what a key the peer admits only to Pair could send.
func (l *Link) UnwelcomedClient(ctx context.Context, t Target) (*connect.Client, error) {
	s, err := l.race(ctx, t.Addresses, func(Key) error { return nil })
	if err != nil {
		return nil, err
	}
	return connect.NewClient(s.rpc), nil
}

// HTTPClient sends raw HTTP over key's live session, as any h2c client on
// the link could.
func (l *Link) HTTPClient(key Key) *http.Client {
	l.mu.Lock()
	defer l.mu.Unlock()
	return &http.Client{Transport: l.live[key].http}
}
