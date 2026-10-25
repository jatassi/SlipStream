package link_test

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/node/v1/nodev1connect"
)

var secret = [joincode.SecretSize]byte{0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e, 0x6f, 0x70, 0x71, 0x72, 0x73}

// codeFor mints a Join Code naming key, reachable at the Controller's
// address.
func codeFor(t *testing.T, key link.Key) joincode.Code {
	t.Helper()
	code, err := joincode.New(key[:], secret, []netip.AddrPort{netip.MustParseAddrPort("127.0.0.2:8485")})
	if err != nil {
		t.Fatal(err)
	}
	return code
}

type pairCall struct {
	peer   link.Key
	secret []byte
	name   string
}

// pairing is a Controller whose Pair handler records each call and
// answers with outcome, pinning the peer when pin is set, as nodes does
// after its commit.
func pairing(t *testing.T, outcome *linkv1.PairResponse, pin bool) (p *linktest.Pair, calls <-chan pairCall) {
	called := make(chan pairCall, 1)
	p = linktest.New(t, linktest.Side{Pair: func(_ context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error) {
		called <- pairCall{peer, req.GetSecret(), req.GetName()}
		if pin {
			p.Controller.Trust.Set(peer, link.Pinned{})
		}
		return outcome, nil
	}}, linktest.Side{})
	p.Controller.Trust.Set(p.Worker.Key, link.Pairing{})
	return p, called
}

func paired(id ...byte) *linkv1.PairResponse {
	return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Paired{Paired: &linkv1.Paired{NodeId: id}}}
}

func refused(r linkv1.RefusedReason) *linkv1.PairResponse {
	return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Refused{Refused: &linkv1.Refused{Reason: r}}}
}

func TestPairSendsTheSecretThenSaysHello(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, calls := pairing(t, paired(0x01, 0x92), true)
		got, err := p.Worker.Pair(t.Context(), codeFor(t, p.Controller.Key), "box")
		if err != nil {
			t.Fatal(err)
		}
		call := <-calls
		if call.peer != p.Worker.Key || !slices.Equal(call.secret, secret[:]) || call.name != "box" {
			t.Fatalf("the Controller's Pair handler got %+v", call)
		}
		if !slices.Equal(got.NodeID, []byte{0x01, 0x92}) || got.Controller != p.Controller.Key || got.Node.Version != "0.0.0-test" || got.Remote != netip.MustParseAddrPort("127.0.0.2:8485") {
			t.Fatalf("Pair returned %+v", got)
		}
		if live, ok := p.Controller.State(p.Worker.Key).(link.Live); !ok || !live.Matched {
			t.Fatalf("after Pair the Controller sees %+v", p.Controller.State(p.Worker.Key))
		}
	})
}

func TestPairRefusals(t *testing.T) {
	for _, tc := range []struct {
		name    string
		outcome *linkv1.PairResponse
		pin     bool
		want    string
	}{
		{"the Controller refuses the code", refused(linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_USED), false, "the peer refused: REFUSED_REASON_JOIN_CODE_USED"},
		{"the Controller pairs but never pins the key", paired(1), false, "the peer refused: REFUSED_REASON_UNKNOWN_KEY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				p, _ := pairing(t, tc.outcome, tc.pin)
				_, err := p.Worker.Pair(t.Context(), codeFor(t, p.Controller.Key), "box")
				var refusal *link.RefusedError
				if !errors.As(err, &refusal) || err.Error() != tc.want {
					t.Fatalf("Pair = %v, want %s", err, tc.want)
				}
				if got := p.Worker.State(p.Controller.Key); got != (link.Down{}) {
					t.Fatalf("after a refused Pair the Worker sees %+v", got)
				}
			})
		})
	}
}

func TestPairWithACodeNamingAnotherKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, calls := pairing(t, paired(1), true)
		_, err := p.Worker.Pair(t.Context(), codeFor(t, p.Worker.Key), "box")
		if !errors.Is(err, link.ErrKeyMismatch) {
			t.Fatalf("Pair with another Controller's code = %v", err)
		}
		if want := "handshake with 127.0.0.2:8485: the Controller's key doesn't match the Join Code: it holds key " + p.Controller.Key.Fingerprint(); err.Error() != want {
			t.Fatalf("Pair with another Controller's code = %q, want %q", err, want)
		}
		if len(calls) != 0 {
			t.Fatal("the Controller saw a Pair")
		}
	})
}

// An Unclaimed Node's Trust refuses every key, so the handshake fails
// before any call. TLS 1.3 finishes the client's side first, so the
// dialer sees only a closed connection and may return before the
// server logs its refusal.
func TestAnUnclaimedNodeRefusesEveryHandshake(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log := &recordLog{}
		p := linktest.New(t, linktest.Side{Log: slog.New(log), Pair: func(context.Context, link.Key, *linkv1.PairRequest) (*linkv1.PairResponse, error) {
			t.Error("an Unclaimed Node ran Pair")
			return nil, nil
		}}, linktest.Side{})
		p.Controller.Trust.Set(p.Worker.Key, link.Refused{})
		if _, err := p.Worker.Pair(t.Context(), codeFor(t, p.Controller.Key), "box"); err == nil {
			t.Fatal("Pair with an Unclaimed Node succeeded")
		}
		synctest.Wait()
		want := []string{"link handshake failed: peer key " + p.Worker.Key.Fingerprint() + " is refused"}
		if got := log.lines(); !slices.Equal(got, want) {
			t.Fatalf("the Unclaimed Node logged %q, want %q", got, want)
		}
	})
}

// A key the Controller admits only to Pair reaches no lockstep handler:
// its request waits for a Welcome that never comes, and the Controller
// closes the session at its Hello deadline.
func TestAKeyAdmittedToPairCallsNothingElse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := make(chan link.Key, 1)
		p := linktest.New(t, linktest.Side{Lockstep: lockstep(calls)}, linktest.Side{})
		p.Controller.Trust.Set(p.Worker.Key, link.Pairing{})
		c, err := p.Worker.UnwelcomedClient(t.Context(), p.Controller.Target())
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		stream, err := nodev1connect.NewSyncServiceClient(c).Sync(t.Context())
		if err == nil {
			defer func() { _ = stream.Close() }()
			err = stream.Send(&nodev1.Up{Frame: &nodev1.Up_Open{Open: &nodev1.Open{}}})
		}
		if err == nil {
			_, err = stream.Receive()
		}
		if err == nil {
			t.Fatal("Sync on a session admitted only to Pair answered")
		}
		if waited := time.Since(start); waited != 10*time.Second {
			t.Fatalf("Sync failed after %v with %v, want at the Controller's 10s Hello deadline", waited, err)
		}
		select {
		case <-calls:
			t.Fatal("the Controller's Sync handler ran")
		default:
		}
		if got := p.Controller.State(p.Worker.Key); got != (link.Down{}) {
			t.Fatalf("the Controller sees %+v, want Down", got)
		}
	})
}
