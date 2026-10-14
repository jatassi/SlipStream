package link_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"testing"
	"testing/synctest"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/node/v1/nodev1connect"
	"github.com/jatassi/slipstream/internal/rpc"
)

// syncer is a lockstep SyncService that acks one frame with the caller's
// key in its scope, so a test sees whom the link said called.
type syncer struct {
	calls chan link.Key
}

func (s syncer) Sync(ctx context.Context, stream nodev1connect.SyncServiceSyncServerStream) error {
	caller, _ := link.PeerOf(ctx)
	s.calls <- caller
	if _, err := stream.Receive(); err != nil {
		return err
	}
	return stream.Send(&nodev1.Down{Frame: &nodev1.Down_Ack{Ack: &nodev1.Ack{Seq: 1}}})
}

func lockstep(calls chan link.Key) http.Handler {
	srv := connect.NewServer()
	nodev1connect.RegisterSyncServiceHandler(srv, syncer{calls})
	m := http.NewServeMux()
	connecthttp.Mount(m, srv)
	return m
}

// syncOnce opens Sync to key over l, sends one frame and reads the ack.
func syncOnce(ctx context.Context, l *link.Link, key link.Key) (uint64, error) {
	stream, err := nodev1connect.NewSyncServiceClient(l.Client(key)).Sync(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = stream.Close() }()
	if err := stream.Send(&nodev1.Up{Frame: &nodev1.Up_Open{Open: &nodev1.Open{}}}); err != nil {
		return 0, err
	}
	down, err := stream.Receive()
	if err != nil {
		return 0, err
	}
	return down.GetAck().GetSeq(), nil
}

func TestLockstepCallsRunBothWays(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		atController, atWorker := make(chan link.Key, 1), make(chan link.Key, 1)
		p := linktest.New(t, linktest.Side{Lockstep: lockstep(atController)}, linktest.Side{Lockstep: lockstep(atWorker)})
		p.Connect(t)

		want := link.Live{
			Peer: link.NodeInfo{
				Version: "0.0.0-test", OS: "linux", Arch: "amd64",
				InstallKind: linkv1.InstallKind_INSTALL_KIND_DEV,
				Addresses:   addresses(t, "127.0.0.2:8485"),
			},
			Matched: true,
			Local:   netip.MustParseAddrPort("127.0.0.3:49152"),
			Remote:  netip.MustParseAddrPort("127.0.0.2:8485"),
		}
		if got := p.Worker.State(p.Controller.Key); got != want {
			t.Fatalf("the Worker's state of the Controller is %+v, want %+v", got, want)
		}

		if seq, err := syncOnce(t.Context(), p.Worker.Link, p.Controller.Key); seq != 1 || err != nil {
			t.Fatalf("Worker to Controller: %d, %v", seq, err)
		}
		if caller := <-atController; caller != p.Worker.Key {
			t.Fatalf("the Controller saw the call from %s, want the Worker", caller.Fingerprint())
		}
		if seq, err := syncOnce(t.Context(), p.Controller.Link, p.Worker.Key); seq != 1 || err != nil {
			t.Fatalf("Controller to Worker: %d, %v", seq, err)
		}
		if caller := <-atWorker; caller != p.Controller.Key {
			t.Fatalf("the Worker saw the call from %s, want the Controller", caller.Fingerprint())
		}
	})
}

func TestMismatchedPeersServeOnlyTheFrozenPackage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := make(chan link.Key, 1)
		p := linktest.New(t, linktest.Side{Version: "0.0.0-a", Lockstep: lockstep(calls)}, linktest.Side{Version: "0.0.0-b"})
		p.Connect(t)

		live, ok := p.Worker.State(p.Controller.Key).(link.Live)
		if !ok || live.Matched || live.Peer.Version != "0.0.0-a" {
			t.Fatalf("the Worker's state of the Controller is %+v", p.Worker.State(p.Controller.Key))
		}
		_, err := syncOnce(t.Context(), p.Worker.Link, p.Controller.Key)
		if !errors.Is(err, link.ErrMismatched) || err.Error() != "failed_precondition: the peer runs another version, so lockstep services are closed" {
			t.Fatalf("a lockstep call between versions = %v", err)
		}
		select {
		case <-calls:
			t.Fatal("the lockstep call reached the Controller")
		default:
		}

		if res := rawHello(t, p.Worker.Link, p.Controller.Key, "0.0.0-b"); res.GetWelcome().GetNode().GetVersion() != "0.0.0-a" {
			t.Fatalf("Hello between versions = %v, want the Controller's Welcome", res)
		}
		_, err = p.Worker.UpdateClient(p.Controller.Key).GetManifest(t.Context(), &linkv1.GetManifestRequest{})
		if connect.CodeOf(err) != connect.CodeUnimplemented {
			t.Fatalf("GetManifest between versions = %v, want it sent and unimplemented", err)
		}
	})
}

// rawHello POSTs a Hello from version to key as plain application/proto
// over the session, with no connect client.
func rawHello(t *testing.T, l *link.Link, key link.Key, version string) *linkv1.HelloResponse {
	t.Helper()
	body, err := proto.Marshal(&linkv1.HelloRequest{Node: &linkv1.NodeInfo{Version: version}})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+key.String()+"/slipstream.link.v1.LinkService/Hello", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/proto")
	resp, err := l.HTTPClient(key).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("POST Hello = %d %q, %v; want 200", resp.StatusCode, out, err)
	}
	var res linkv1.HelloResponse
	if err := proto.Unmarshal(out, &res); err != nil {
		t.Fatal(err)
	}
	return &res
}

func TestACallWithNoSessionIsNotSent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := linktest.New(t, linktest.Side{}, linktest.Side{})
		_, err := syncOnce(t.Context(), p.Worker.Link, p.Controller.Key)
		if !errors.Is(err, link.ErrDisconnected) || err.Error() != "unavailable: no live link session to the peer" {
			t.Fatalf("a call with no session = %v", err)
		}
	})
}

func addresses(t *testing.T, list ...string) link.Addresses {
	t.Helper()
	var aps []netip.AddrPort
	for _, s := range list {
		aps = append(aps, netip.MustParseAddrPort(s))
	}
	a, err := link.ParseAddresses(aps)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// The reset ends only that stream: the session still carries calls.
func TestAStreamOfAnUnknownKindIsReset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := linktest.New(t, linktest.Side{Lockstep: lockstep(make(chan link.Key, 1))}, linktest.Side{})
		p.Connect(t)
		st, err := p.Worker.OpenStream(t.Context(), p.Controller.Key, 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.Read(make([]byte, 1)); err == nil || err.Error() != "stream reset by remote, error code: 0" {
			t.Fatalf("reading a kind-3 stream = %v", err)
		}
		if seq, err := syncOnce(t.Context(), p.Worker.Link, p.Controller.Key); err != nil || seq != 1 {
			t.Fatalf("a call after the reset = %d, %v", seq, err)
		}
	})
}

// record serves a stream by reporting its peer and closing it.
func record(peers chan link.Key) map[link.Kind]func(link.Key, link.Stream) {
	return map[link.Kind]func(link.Key, link.Stream){
		link.KindTransfer: func(peer link.Key, s link.Stream) {
			peers <- peer
			_ = s.Close()
		},
	}
}

func TestATransferStreamReachesItsHandler(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		peers := make(chan link.Key, 1)
		p := linktest.New(t, linktest.Side{}, linktest.Side{Streams: record(peers)})
		p.Connect(t)
		st, err := p.Controller.OpenStream(t.Context(), p.Worker.Key, link.KindTransfer)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
			t.Fatalf("reading a stream its handler closed = %v", err)
		}
		if peer := <-peers; peer != p.Controller.Key {
			t.Fatalf("the handler saw %s, want the Controller", peer.Fingerprint())
		}
	})
}

func TestATransferStreamWithNoSessionIsNotSent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := linktest.New(t, linktest.Side{}, linktest.Side{Streams: record(make(chan link.Key, 1))})
		_, err := p.Controller.OpenStream(t.Context(), p.Worker.Key, link.KindTransfer)
		if why, ok := rpc.NotSent(err); !ok || why != rpc.Disconnected {
			t.Fatalf("opening with no session = %v, want it not sent as disconnected", err)
		}
	})
}

// Each side refuses: the opener before sending, and the acceptor for a
// peer that opened anyway.
func TestATransferStreamBetweenVersionsIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		peers := make(chan link.Key, 1)
		p := linktest.New(t, linktest.Side{Version: "0.0.0-a"}, linktest.Side{Version: "0.0.0-b", Streams: record(peers)})
		p.Connect(t)
		_, err := p.Controller.OpenStream(t.Context(), p.Worker.Key, link.KindTransfer)
		if why, ok := rpc.NotSent(err); !ok || why != rpc.Mismatched {
			t.Fatalf("opening between versions = %v, want it not sent as mismatched", err)
		}
		st, err := p.Controller.RawStream(t.Context(), p.Worker.Key, link.KindTransfer)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.Read(make([]byte, 1)); err == nil || err.Error() != "stream reset by remote, error code: 0" {
			t.Fatalf("reading a stream opened between versions = %v", err)
		}
		expectNoPeer(t, peers)
	})
}

func TestATransferStreamOnAnUnwelcomedSessionIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		peers := make(chan link.Key, 1)
		p := linktest.New(t, linktest.Side{}, linktest.Side{Streams: record(peers)})
		st, err := p.Controller.UnwelcomedStream(t.Context(), p.Worker.Target(), link.KindTransfer)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.Read(make([]byte, 1)); err == nil {
			t.Fatal("a stream on a session with no Hello was served")
		}
		expectNoPeer(t, peers)
	})
}

func expectNoPeer(t *testing.T, peers chan link.Key) {
	t.Helper()
	synctest.Wait()
	select {
	case peer := <-peers:
		t.Fatalf("the handler served %s", peer.Fingerprint())
	default:
	}
}
