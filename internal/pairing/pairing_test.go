package pairing_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/pairing"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/system"
)

var discard = slog.New(slog.DiscardHandler)

// lazy answers a link's Trust and Pair through whatever is set once the
// link exists.
type lazy struct {
	trust link.Trust
	pair  link.PairFunc
}

func (l *lazy) Admit(key link.Key) link.Admission { return l.trust.Admit(key) }

func (l *lazy) Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	return l.pair(ctx, peer, req)
}

type controller struct {
	node *linktest.Node
	svc  *nodes.Service
}

func startController(t *testing.T, net *fault.Network, name, addr string, advertise []netip.AddrPort) controller {
	t.Helper()
	l := &lazy{}
	node := linktest.Start(t, net, name, []netip.Addr{netip.MustParseAddr(addr)}, linktest.Side{Trust: l, Pair: l.Pair, Advertise: advertise})
	hub := live.NewHub([]live.Event{nodes.ChangedEvent})
	t.Cleanup(hub.Close)
	svc, err := nodes.Open(t.Context(), nodes.Config{
		Library: storagetest.Library(t, nodes.Schema(), role.LibrarySchema()),
		Key:     node.Key,
		Name:    name,
		Link:    node.Link,
		Changed: nodes.ChangedEvent.Bind(hub),
		Log:     discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	l.trust, l.pair = svc, svc.Pair
	return controller{node: node, svc: svc}
}

func (c controller) mint(t *testing.T) joincode.Code {
	t.Helper()
	code, err := c.svc.MintJoinCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return code.Code
}

type claims system.Claim

func (c claims) Claim(context.Context) (system.Claim, error) { return system.Claim(c), nil }

// worker is box on 127.0.0.3 with its pairing over the link.
func worker(t *testing.T, net *fault.Network, dir string, claim system.Claim) (*linktest.Node, *pairing.Pairing) {
	t.Helper()
	l := &lazy{}
	node := linktest.Start(t, net, "box", []netip.Addr{linktest.WorkerAddr}, linktest.Side{Trust: l, Pair: l.Pair})
	p, err := pairing.Open(pairing.Config{DataDir: dir, Link: node.Link, Name: "box", Claims: claims(claim), Log: discard})
	if err != nil {
		t.Fatal(err)
	}
	l.trust, l.pair = p, p.Pair
	return node, p
}

func kind(err error) pairing.Kind {
	if e, ok := errors.AsType[*pairing.Error](err); ok {
		return e.Kind
	}
	return ""
}

func TestARepairSupersedesTheOldController(t *testing.T) {
	dir := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		net := fault.NewNetwork()
		first := startController(t, net, "home", "127.0.0.2", nil)
		second := startController(t, net, "restored", "127.0.0.4", nil)
		box, p := worker(t, net, dir, system.Unclaimed)

		joined, err := p.Claim(t.Context(), first.mint(t))
		if err != nil {
			t.Fatal(err)
		}
		if got := joined.String(); got != "joined home as box (node "+joined.NodeID.String()+"); link up to 127.0.0.2:8485" {
			t.Fatalf("joined line %q", got)
		}
		if _, err := p.Claim(t.Context(), first.mint(t)); kind(err) != pairing.AlreadyJoined || err.Error() != "already joined" {
			t.Fatalf("the pinned Controller's code: %v, want already joined", err)
		}

		if _, err := p.Claim(t.Context(), second.mint(t)); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		rec := checkSuperseded(t, p, box, first, second)
		if _, err := p.Claim(t.Context(), first.mint(t)); kind(err) != pairing.SupersededController || err.Error() != "superseded Controller" {
			t.Fatalf("a superseded Controller's code: %v, want superseded Controller", err)
		}

		reopened, err := pairing.Open(pairing.Config{DataDir: dir})
		if err != nil {
			t.Fatal(err)
		}
		if again, _ := reopened.Record(); again.Controller != rec.Controller || len(again.Superseded) != 1 || again.Superseded[0] != rec.Superseded[0] || again.NodeID != rec.NodeID {
			t.Fatalf("pairing.json read back as %+v, want %+v", again, rec)
		}
	})
}

// checkSuperseded holds the record and Trust after box re-paired from first
// to second, and returns the record.
func checkSuperseded(t *testing.T, p *pairing.Pairing, box *linktest.Node, first, second controller) pairing.Record {
	t.Helper()
	rec, _ := p.Record()
	if len(rec.Superseded) != 1 || rec.Superseded[0].Key != first.node.Key || rec.Superseded[0].By != second.node.Key || rec.Controller.Key != second.node.Key {
		t.Fatalf("record after the re-pair: controller %s, superseded %+v; want %s superseded by %s", rec.Controller.Key.Fingerprint(), rec.Superseded, first.node.Key.Fingerprint(), second.node.Key.Fingerprint())
	}
	if box.State(first.node.Key) != (link.Down{}) {
		t.Fatal("the superseded Controller's session is still up")
	}
	want := link.Superseded{By: second.node.Key, Addresses: rec.Controller.Addresses}
	if got := p.Admit(first.node.Key); got != want {
		t.Fatalf("the superseded key is admitted as %#v, want %#v", got, want)
	}
	if p.Admit(second.node.Key) != (link.Pinned{}) || p.Admit(box.Key) != (link.Refused{}) {
		t.Fatal("the current Controller is not pinned, or an unknown key is not refused")
	}
	return rec
}

// forward is a port forward on its own Host: every connection to at is
// copied both ways to upstream until the test ends.
func forward(t *testing.T, net *fault.Network, at, upstream netip.AddrPort) {
	t.Helper()
	h := net.Host(at.Addr())
	ln, err := h.Listen("tcp", at.String())
	if err != nil {
		t.Fatal(err)
	}
	context.AfterFunc(t.Context(), func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			u, err := h.DialContext(t.Context(), "tcp", upstream.String())
			if err != nil {
				_ = c.Close()
				continue
			}
			context.AfterFunc(t.Context(), func() { _ = c.Close(); _ = u.Close() })
			go func() { _, _ = io.Copy(u, c); _ = u.Close() }()
			go func() { _, _ = io.Copy(c, u); _ = c.Close() }()
		}
	}()
}

func TestARepairReachesAControllerThroughItsAdvertisedAddress(t *testing.T) {
	dir := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		net := fault.NewNetwork()
		forwarded := netip.MustParseAddrPort("127.0.0.9:443")
		first := startController(t, net, "home", "127.0.0.2", nil)
		second := startController(t, net, "restored", "127.0.0.4", []netip.AddrPort{forwarded})
		forward(t, net, forwarded, netip.MustParseAddrPort("127.0.0.4:8485"))
		for _, f := range []fault.Fault{fault.Reset{}, fault.Refuse{}} {
			if err := net.Path(linktest.WorkerAddr, netip.MustParseAddr("127.0.0.4")).Set(fault.Rule{ID: f.String(), Fault: f}); err != nil {
				t.Fatal(err)
			}
		}
		box, p := worker(t, net, dir, system.Unclaimed)
		if _, err := p.Claim(t.Context(), first.mint(t)); err != nil {
			t.Fatal(err)
		}

		code := second.mint(t)
		if got := code.Addresses(); !slices.Equal(got, []netip.AddrPort{forwarded}) {
			t.Fatalf("the Join Code names %v, want [%s]", got, forwarded)
		}
		joined, err := p.Claim(t.Context(), code)
		if err != nil {
			t.Fatal(err)
		}
		if got := joined.String(); got != "joined restored as box (node "+joined.NodeID.String()+"); link up to 127.0.0.9:443" {
			t.Fatalf("joined line %q", got)
		}
		synctest.Wait()
		rec := checkSuperseded(t, p, box, first, second)
		if got := rec.Controller.Addresses.All(); !slices.Equal(got, []netip.AddrPort{forwarded}) {
			t.Fatalf("the record dials the Controller at %v, want [%s]", got, forwarded)
		}
	})
}

func TestAControllerRefusesAJoin(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		net := fault.NewNetwork()
		home := startController(t, net, "home", "127.0.0.2", nil)
		_, p := worker(t, net, t.TempDir(), system.Controller)
		if _, err := p.Claim(t.Context(), home.mint(t)); kind(err) != pairing.IsController || err.Error() != "this Node is a Controller" {
			t.Fatalf("a Controller's Join: %v, want this Node is a Controller", err)
		}
	})
}

func TestAnUnreachableControllerIsNamed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		net := fault.NewNetwork()
		home := startController(t, net, "home", "127.0.0.2", nil)
		code := home.mint(t)
		for _, f := range []fault.Fault{fault.Reset{}, fault.Refuse{}} {
			if err := net.Path(linktest.ControllerAddr, linktest.WorkerAddr).Set(fault.Rule{ID: f.String(), Fault: f}); err != nil {
				t.Fatal(err)
			}
		}
		_, p := worker(t, net, t.TempDir(), system.Unclaimed)
		if _, err := p.Claim(t.Context(), code); kind(err) != pairing.Unreachable {
			t.Fatalf("a Join over a reset path: %v, want controller_unreachable", err)
		}
	})
}

func TestARecordThisBinaryCannotReadIsRefused(t *testing.T) {
	for _, c := range []struct{ name, record, want string }{
		{"another schema", `{"schema":2}`, "schema 2, want 1"},
		{"an unknown key", `{"schema":1,"later":true}`, `decode: json: unknown field "later"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, pairing.RecordFile)
			if err := os.WriteFile(path, []byte(c.record), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := pairing.Open(pairing.Config{DataDir: dir})
			var refused *pairing.RecordError
			if !errors.As(err, &refused) || err.Error() != "pairing record "+path+": "+c.want {
				t.Fatalf("Open = %v, want a RecordError ending %q", err, c.want)
			}
		})
	}
}
