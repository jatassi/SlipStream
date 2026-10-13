package link_test

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"testing"
	"testing/synctest"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/link/v1/linkv1connect"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/node/v1/nodev1connect"
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

		res, err := linkv1connect.NewLinkServiceClient(p.Worker.Client(p.Controller.Key)).Hello(t.Context(), &linkv1.HelloRequest{Node: &linkv1.NodeInfo{Version: "0.0.0-b"}})
		if err != nil || res.GetWelcome().GetNode().GetVersion() != "0.0.0-a" {
			t.Fatalf("Hello between versions = %v, %v", res, err)
		}
		_, err = p.Worker.UpdateClient(p.Controller.Key).GetManifest(t.Context(), &linkv1.GetManifestRequest{})
		if connect.CodeOf(err) != connect.CodeUnimplemented {
			t.Fatalf("GetManifest between versions = %v, want it sent and unimplemented", err)
		}
	})
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
		st, err := p.Worker.OpenStream(t.Context(), p.Controller.Key, 2)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.Read(make([]byte, 1)); err == nil || err.Error() != "stream reset by remote, error code: 0" {
			t.Fatalf("reading a kind-2 stream = %v", err)
		}
		if seq, err := syncOnce(t.Context(), p.Worker.Link, p.Controller.Key); err != nil || seq != 1 {
			t.Fatalf("a call after the reset = %d, %v", seq, err)
		}
	})
}
