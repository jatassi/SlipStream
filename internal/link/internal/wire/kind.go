package wire

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/jatassi/slipstream/internal/link/internal/mux"
)

// Kind is the byte that opens every yamux stream and names what it
// carries.
type Kind byte

// RPC is one h2c connection. Other kinds are reserved for Transfers and
// relay forwards.
const RPC Kind = 1

func Open(ctx context.Context, s *mux.Session, k Kind) (mux.Stream, error) {
	st, err := s.Open(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := st.Write([]byte{byte(k)}); err != nil {
		_ = st.Reset()
		return nil, fmt.Errorf("write stream kind: %w", err)
	}
	return st, nil
}

func ReadKind(c net.Conn) (Kind, error) {
	var b [1]byte
	if _, err := io.ReadFull(c, b[:]); err != nil {
		return 0, fmt.Errorf("read stream kind: %w", err)
	}
	return Kind(b[0]), nil
}
