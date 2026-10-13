package wire_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/jatassi/slipstream/internal/link/internal/mux"
	"github.com/jatassi/slipstream/internal/link/internal/wire"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/node/v1/nodev1connect"
)

const window = 16 << 20

var muxConfig = mux.Config{Keepalive: 25 * time.Second, WriteTimeout: 10 * time.Second, StreamWindow: window}

// acker acks each Up frame with its count.
type acker struct{}

func (acker) Sync(_ context.Context, s nodev1connect.SyncServiceSyncServerStream) error {
	for seq := uint64(1); ; seq++ {
		if _, err := s.Receive(); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
		if err := s.Send(&nodev1.Down{Frame: &nodev1.Down_Ack{Ack: &nodev1.Ack{Seq: seq}}}); err != nil {
			return err
		}
	}
}

// serveKinds hands s's RPC-kind streams to rpc.
func serveKinds(s *mux.Session, rpc *wire.Conns) {
	for {
		st, err := s.Accept()
		if err != nil {
			return
		}
		go func() {
			if k, err := wire.ReadKind(st); err == nil && k == wire.RPC {
				rpc.Push(st)
				return
			}
			_ = st.Reset()
		}()
	}
}

func sessions(t *testing.T) (client, server *mux.Session) {
	t.Helper()
	c, s := conns(t)
	client, err := mux.Client(c, muxConfig)
	if err != nil {
		t.Fatal(err)
	}
	server, err = mux.Server(s, muxConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close(); _ = client.Close() })
	return client, server
}

// serveSync serves the acker on s's RPC-kind streams over h2c.
func serveSync(t *testing.T, s *mux.Session) {
	rpc := wire.NewConns()
	srv := connect.NewServer()
	nodev1connect.RegisterSyncServiceHandler(srv, acker{})
	routes := http.NewServeMux()
	connecthttp.Mount(routes, srv)
	h := wire.Server(window, routes, nil, slog.New(slog.DiscardHandler))
	go func() { _ = h.Serve(rpc) }()
	t.Cleanup(func() { _ = h.Close() })
	go serveKinds(s, rpc)
}

// TestBidiStreamOverH2COnMuxStreams runs a connect-go bidi stream through
// http.Transport's h2c with a DialContext that opens RPC-kind streams.
func TestBidiStreamOverH2COnMuxStreams(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, server := sessions(t)
		serveSync(t, server)

		tr := wire.Transport(window, func(ctx context.Context) (net.Conn, error) { return wire.Open(ctx, client, wire.RPC) })
		defer tr.CloseIdleConnections()
		sync := nodev1connect.NewSyncServiceClient(connect.NewClient(connecthttp.NewTransport(&http.Client{Transport: tr}, "http://link")))
		stream, err := sync.Sync(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		var acks []uint64
		for range 3 {
			if err := stream.Send(&nodev1.Up{Frame: &nodev1.Up_Facts{Facts: &nodev1.Facts{Scope: "node"}}}); err != nil {
				t.Fatal(err)
			}
			down, err := stream.Receive()
			if err != nil {
				t.Fatal(err)
			}
			acks = append(acks, down.GetAck().GetSeq())
		}
		if err := stream.CloseSend(); err != nil {
			t.Fatal(err)
		}
		if _, err := stream.Receive(); !errors.Is(err, io.EOF) {
			t.Fatalf("after CloseSend, Receive = %v, want io.EOF", err)
		}
		if !slices.Equal(acks, []uint64{1, 2, 3}) {
			t.Fatalf("acks %v, want [1 2 3]", acks)
		}
	})
}
