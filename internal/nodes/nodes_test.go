package nodes_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodes"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

var discard = slog.New(slog.DiscardHandler)

// side is the Controller link's Trust and Pair, answered by the Service
// once it is open.
type side struct{ svc *nodes.Service }

func (s *side) Admit(key link.Key) link.Admission { return s.svc.Admit(key) }

func (s *side) Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	return s.svc.Pair(ctx, peer, req)
}

// published records each nodes.changed payload.
type published struct {
	mu  sync.Mutex
	ids []string
}

func (p *published) Publish(_ context.Context, v nodes.Changed) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ids = append(p.ids, v.NodeID)
}

func (p *published) has(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Contains(p.ids, id)
}

type controller struct {
	net  *fault.Network
	node *linktest.Node
	svc  *nodes.Service
	pub  *published
}

func startController(t *testing.T) *controller {
	t.Helper()
	net := fault.NewNetwork()
	s := &side{}
	node := linktest.Start(t, net, "home", []netip.Addr{linktest.ControllerAddr}, linktest.Side{Trust: s, Pair: s.Pair})
	pub := &published{}
	svc, err := nodes.Open(t.Context(), nodes.Config{
		Library: storagetest.Library(t, nodes.Schema(), role.LibrarySchema()),
		Key:     node.Key,
		Name:    "home",
		Link:    node.Link,
		Changed: pub,
		Log:     discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	s.svc = svc
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = svc.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return &controller{net: net, node: node, svc: svc, pub: pub}
}

func (c *controller) mint(t *testing.T) nodes.JoinCode {
	t.Helper()
	code, err := c.svc.MintJoinCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func (c *controller) worker(t *testing.T, name, addr string) *linktest.Node {
	t.Helper()
	return linktest.Start(t, c.net, name, []netip.Addr{netip.MustParseAddr(addr)}, linktest.Side{})
}

func refusal(err error) linkv1.RefusedReason {
	if r, ok := errors.AsType[*link.RefusedError](err); ok {
		return r.Reason
	}
	return linkv1.RefusedReason_REFUSED_REASON_UNSPECIFIED
}

func (c *controller) pair(t *testing.T, w *linktest.Node, code nodes.JoinCode, name string) (link.Paired, role.NodeID) {
	t.Helper()
	paired, err := w.Pair(t.Context(), code.Code, name)
	if err != nil {
		t.Fatal(err)
	}
	id, err := storage.UUIDFromBytes(paired.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	return paired, role.NodeID{UUID: id}
}

func TestRepeatingAPairReturnsTheSameNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		code := c.mint(t)
		if got, want := code.Code.Addresses(), []netip.AddrPort{netip.MustParseAddrPort("127.0.0.2:8485")}; !slices.Equal(got, want) {
			t.Fatalf("the code names %v, want %v", got, want)
		}
		box := c.worker(t, "box", "127.0.0.3")
		first, id := c.pair(t, box, code, "box")
		if first.Name != "box" || first.ControllerName != "home" || first.Controller != c.node.Key {
			t.Fatalf("paired as %q with %q (%s), want box with home (%s)", first.Name, first.ControllerName, first.Controller.Fingerprint(), c.node.Key.Fingerprint())
		}
		synctest.Wait()
		if !c.pub.has(id.String()) {
			t.Errorf("nodes.changed was not published for %s", id)
		}
		if again, _ := c.pair(t, box, code, "box"); !bytes.Equal(again.NodeID, first.NodeID) {
			t.Fatalf("pairing again with the same code gave node %x, want %x", again.NodeID, first.NodeID)
		}
		_, err := c.worker(t, "other", "127.0.0.4").Pair(t.Context(), code.Code, "other")
		if got := refusal(err); got != linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_USED {
			t.Fatalf("another key with a used code: %v, want REFUSED_REASON_JOIN_CODE_USED", err)
		}
	})
}

func TestAKnownKeyKeepsItsRoles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		box := c.worker(t, "box", "127.0.0.3")
		_, id := c.pair(t, box, c.mint(t), "box")
		box.Drop(c.node.Key)
		synctest.Wait()
		if _, err := c.svc.SetRoles(t.Context(), id, []role.Name{role.Downloading, role.Searching, role.Searching}); err != nil {
			t.Fatal(err)
		}
		if _, again := c.pair(t, box, c.mint(t), "renamed"); again != id {
			t.Fatalf("a new code paired the known key as %s, want %s", again, id)
		}
		got, err := c.svc.Get(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if want := []role.Name{role.Searching, role.Downloading}; got.Name != "box" || !slices.Equal(got.Roles, want) {
			t.Fatalf("after a new code the known key is %q holding %v, want box holding %v", got.Name, got.Roles, want)
		}
	})
}

func TestACodePastItsLifetimeIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		if err := c.svc.SetJoinCodeLifetime(t.Context(), 10*time.Minute); err != nil {
			t.Fatal(err)
		}
		code := c.mint(t)
		time.Sleep(10 * time.Minute)
		_, err := c.worker(t, "box", "127.0.0.3").Pair(t.Context(), code.Code, "box")
		if got := refusal(err); got != linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_EXPIRED {
			t.Fatalf("a code past its lifetime: %v, want REFUSED_REASON_JOIN_CODE_EXPIRED", err)
		}
	})
}

func TestAnUnknownCodeIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		minted := c.mint(t)
		var secret [16]byte
		forged, err := joincode.New(ed25519.PublicKey(c.node.Key[:]), secret, minted.Code.Addresses())
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.worker(t, "box", "127.0.0.3").Pair(t.Context(), forged, "box")
		if got := refusal(err); got != linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_UNKNOWN {
			t.Fatalf("a code the Controller never minted: %v, want REFUSED_REASON_JOIN_CODE_UNKNOWN", err)
		}
	})
}

func TestTheControllersOwnRow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		list, err := c.svc.List(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 {
			t.Fatalf("listed %d Nodes, want the Controller's own row", len(list))
		}
		own := list[0]
		want := slices.DeleteFunc(role.Names(), func(r role.Name) bool { return r == role.Relaying })
		if !own.Self || own.Name != "home" || own.Key != c.node.Key || own.Link != nodes.Local || !slices.Equal(own.Roles, want) {
			t.Fatalf("own row = self %v, %q, %s, %s, %v; want self, home, its key, local, %v", own.Self, own.Name, own.Key.Fingerprint(), own.Link, own.Roles, want)
		}
		if err := c.svc.SetJoinCodeLifetime(t.Context(), 9*time.Minute); !errors.Is(err, nodes.ErrLifetime) {
			t.Fatalf("a 9 minute lifetime: %v, want ErrLifetime", err)
		}
		if _, err := c.svc.SetRoles(t.Context(), role.NodeID{UUID: storage.NewUUID()}, nil); !errors.Is(err, nodes.ErrNotFound) {
			t.Fatalf("roles for an unknown Node: %v, want ErrNotFound", err)
		}
	})
}

func TestTheNodesSetPassesTheAudit(t *testing.T) {
	storagetest.AuditLibrary(t, []storage.Schema[storage.Library]{nodes.Schema()})
}
