package link_test

import (
	"context"
	"errors"
	"net/netip"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

// unreachableCode names key at an address nothing listens on, as a Join
// Code from a Controller behind CGNAT reads to its Worker.
func unreachableCode(t *testing.T, key link.Key) joincode.Code {
	t.Helper()
	code, err := joincode.New(key[:], secret, []netip.AddrPort{netip.MustParseAddrPort("127.0.0.9:8485")})
	if err != nil {
		t.Fatal(err)
	}
	return code
}

// awaiting is an Unclaimed Worker, whose Trust refuses the Controller,
// awaiting code in the background.
func awaiting(t *testing.T, p *linktest.Pair, code joincode.Code) <-chan awaited {
	t.Helper()
	p.Worker.Trust.Set(p.Controller.Key, link.Refused{})
	done := make(chan awaited, 1)
	go func() {
		paired, err := p.Worker.AwaitPair(t.Context(), code, "box")
		done <- awaited{paired, err}
	}()
	synctest.Wait()
	return done
}

type awaited struct {
	paired link.Paired
	err    error
}

func TestInviteCarriesTheAwaitedPair(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, calls := pairing(t, paired(0x01, 0x92), true)
		code := unreachableCode(t, p.Controller.Key)
		if _, err := p.Worker.Pair(t.Context(), code, "box"); err == nil {
			t.Fatal("Pair reached a Controller at an address nothing listens on")
		}
		done := awaiting(t, p, code)
		if err := p.Controller.Invite(t.Context(), p.Worker.Target()); err != nil {
			t.Fatal(err)
		}
		got := <-done
		if got.err != nil {
			t.Fatal(got.err)
		}
		call := <-calls
		if call.peer != p.Worker.Key || !slices.Equal(call.secret, secret[:]) || call.name != "box" {
			t.Fatalf("the Controller's Pair handler got %+v", call)
		}
		if !slices.Equal(got.paired.NodeID, []byte{0x01, 0x92}) || got.paired.Controller != p.Controller.Key || got.paired.Remote.Addr() != linktest.ControllerAddr {
			t.Fatalf("AwaitPair returned %+v", got.paired)
		}
		if live, ok := p.Controller.State(p.Worker.Key).(link.Live); !ok || !live.Dialed {
			t.Fatalf("the Controller sees %+v, want the session it dialed live", p.Controller.State(p.Worker.Key))
		}
		if live, ok := p.Worker.State(p.Controller.Key).(link.Live); !ok || live.Dialed {
			t.Fatalf("the Worker sees %+v, want the session it accepted live", p.Worker.State(p.Controller.Key))
		}
		if err := p.Controller.Invite(t.Context(), p.Worker.Target()); err != nil {
			t.Fatalf("Invite with the session live = %v, want nil at once", err)
		}
	})
}

// An Invite before the Worker awaits fails its handshake, and its retry
// reaches the await once it starts.
func TestInviteDialsAgainUntilTheWorkerAwaits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, _ := pairing(t, paired(1), true)
		p.Worker.Trust.Set(p.Controller.Key, link.Refused{})
		invited := make(chan error, 1)
		go func() { invited <- p.Controller.Invite(t.Context(), p.Worker.Target()) }()
		time.Sleep(5 * time.Second)
		done := awaiting(t, p, unreachableCode(t, p.Controller.Key))
		if err := <-invited; err != nil {
			t.Fatal(err)
		}
		if got := <-done; got.err != nil {
			t.Fatal(got.err)
		}
	})
}

// A Controller that refuses the code ends the await, and its Invite
// keeps dialing until its context ends.
func TestARefusedPairEndsTheAwait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, _ := pairing(t, refused(linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_USED), false)
		done := awaiting(t, p, unreachableCode(t, p.Controller.Key))
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		invited := make(chan error, 1)
		go func() { invited <- p.Controller.Invite(ctx, p.Worker.Target()) }()
		got := <-done
		if refusal, ok := errors.AsType[*link.RefusedError](got.err); !ok || refusal.Reason != linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_USED {
			t.Fatalf("AwaitPair = %v, want the Controller's refusal", got.err)
		}
		if err := <-invited; !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Invite = %v, want its context's deadline", err)
		}
	})
}

// Only the key the awaited code names passes TLS for the await.
func TestAnAwaitAdmitsOnlyItsController(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, _ := pairing(t, paired(1), true)
		done := awaiting(t, p, unreachableCode(t, p.Controller.Key))
		other := linktest.Start(t, p.Net, "other", []netip.Addr{netip.MustParseAddr("127.0.0.4")}, linktest.Side{})
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()
		if err := other.Invite(ctx, p.Worker.Target()); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("another key's Invite = %v, want it refused until its deadline", err)
		}
		select {
		case got := <-done:
			t.Fatalf("the await ended with %+v on another key's dial", got)
		default:
		}
	})
}
