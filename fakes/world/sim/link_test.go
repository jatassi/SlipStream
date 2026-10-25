package sim_test

import (
	"context"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/pairing"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/link/v1/linkv1connect"
	rpctestv1 "github.com/jatassi/slipstream/internal/pb/slipstream/rpctest/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/rpctest/v1/rpctestv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/role/roletest"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// key is the Node named name's public key, from its node.key.
func key(t *testing.T, w *sim.World, name string) link.Key {
	t.Helper()
	k, err := link.OpenKey(w.Node(name).DataDir().Path())
	if err != nil {
		t.Fatal(err)
	}
	return k.Public()
}

// pings answers Ping and counts the calls that reached it.
type pings struct{ calls chan struct{} }

func (p pings) Ping(context.Context, *rpctestv1.PingRequest) (*rpctestv1.PingResponse, error) {
	p.calls <- struct{}{}
	return &rpctestv1.PingResponse{}, nil
}

// TestMismatch restarts seedbox on another build. Hello still succeeds,
// so the link is live but unmatched: a lockstep call is not sent, no Sync
// stream reaches home, and seedbox's Probe keeps its last intent as
// Mismatched, while the frozen services still answer. The link tier sends
// the raw Hello POST (TestMismatchedPeersServeOnlyTheFrozenPackage).
func TestMismatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithProbe(), sim.Build("0.0.0-a"))
		synctest.Wait()
		box := nodeID(t, w, "seedbox")
		err := w.Node("home").Half().Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return roletest.SetIntent(ctx, tx, box, "before")
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Node("home").Control().Touch(box)
		synctest.Wait()

		worker := w.Node("seedbox")
		syncs := w.Node("home").SyncStreams()
		if syncs == 0 {
			t.Fatal("no Sync stream reached home from the matched seedbox")
		}
		worker.Stop()
		worker.Start(func(e *node.Env) { e.Version = "0.0.0-b" })
		synctest.Wait()
		if opened := w.Node("home").SyncStreams() - syncs; opened != 0 {
			t.Fatalf("%d Sync streams reached home from the mismatched seedbox, want none", opened)
		}
		homeKey := key(t, w, "home")
		if live, ok := worker.Half().Link.State(homeKey).(link.Live); !ok || live.Matched || live.Peer.Version != "0.0.0-a" {
			t.Fatalf("seedbox's link to home is %+v, want live and unmatched with 0.0.0-a", worker.Half().Link.State(homeKey))
		}
		checkNotSent(t, w, box)
		r := seedbox(t, w.API())
		if r.Link.State != "mismatched" || r.Facts != nil {
			t.Fatalf("listNodes shows seedbox %s with facts %+v, want mismatched with none, since no Sync stream opened", r.Link.State, r.Facts)
		}
		snap, ok := worker.Probe().Snapshot()
		if !ok || snap.Standing != role.Mismatched || snap.Intent.GetValue() != "before" {
			t.Fatalf("seedbox's Probe snapshot is %v, want the cached intent before, mismatched", snap)
		}

		res, err := linkv1connect.NewLinkServiceClient(worker.Half().Link.Client(homeKey)).Hello(t.Context(), &linkv1.HelloRequest{Node: &linkv1.NodeInfo{Version: "0.0.0-b"}})
		if err != nil || res.GetWelcome().GetNode().GetVersion() != "0.0.0-a" {
			t.Fatalf("Hello from seedbox to home = %v, %v; want home's Welcome", res, err)
		}
	})
}

// checkNotSent calls a lockstep service on seedbox from home: the call is
// NotSent as Mismatched and never reaches seedbox's handler.
func checkNotSent(t *testing.T, w *sim.World, box role.NodeID) {
	t.Helper()
	p := pings{calls: make(chan struct{}, 1)}
	rpctestv1connect.PingService.Mount(w.Node("seedbox").Half().Dispatch, p)
	_, err := rpctestv1connect.PingService.On(w.Node("home").Half().Cluster, box).Ping(t.Context(), &rpctestv1.PingRequest{})
	if why, ok := rpc.NotSent(err); !ok || why != rpc.Mismatched {
		t.Fatalf("a call from home to seedbox = %v, want NotSent mismatched", err)
	}
	select {
	case <-p.calls:
		t.Fatal("the call reached seedbox's handler")
	default:
	}
}

// office is a second Controller: the Unclaimed Node office of the world,
// claimed with an Admin of its own. It returns its API key.
func office(t *testing.T, w *sim.World) string {
	t.Helper()
	ready := w.Node("office").Ready()
	apiKey, err := nodeapi.ClaimController(t.Context(), w.Client(), "http://"+ready.Listeners.HTTP.String(), ready.SetupCode, nodeapi.Admin{Email: "olu@example.test", DisplayName: "Olu Bello", PIN: "1357"})
	if err != nil {
		t.Fatal(err)
	}
	return apiKey
}

// mint is a Join Code the Controller named name mints with apiKey.
func mint(t *testing.T, w *sim.World, name, apiKey string) joincode.Code {
	t.Helper()
	text, err := nodeapi.CreateJoinCode(t.Context(), w.Client(), "http://"+w.Node(name).Ready().Listeners.HTTP.String(), apiKey)
	if err != nil {
		t.Fatal(err)
	}
	code, err := joincode.Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	return code
}

// record is seedbox's pairing.json, as much of it as a re-pair changes.
type record struct {
	Controller struct {
		Key string `json:"key"`
	} `json:"controller"`
	Superseded []struct {
		Key   string `json:"key"`
		ByKey string `json:"byKey"`
	} `json:"superseded"`
}

func readRecord(t *testing.T, n *sim.Node) record {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(n.DataDir().Path(), pairing.RecordFile))
	if err != nil {
		t.Fatal(err)
	}
	var r record
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

// logged is the records of n's log with message msg, as key-to-value maps.
func logged(t *testing.T, n *sim.Node, msg string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range n.Log() {
		var r map[string]any
		if json.Unmarshal([]byte(line), &r) == nil && r["msg"] == msg {
			out = append(out, r)
		}
	}
	return out
}

// TestRepairSupersedes joins seedbox to a second Controller, office, the
// way an operator does on a stopped Node. Its record lists home's key as
// superseded by office's, home's Hello is answered that office superseded
// it, and a Join Code home mints later is ignored.
func TestRepairSupersedes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := home(t)
		s.Name = "repair"
		s.Nodes = append(s.Nodes, scenario.Node{Name: "office"})
		w := sim.Launch(t, s)
		officeKey := office(t, w)
		worker := w.Node("seedbox")
		worker.Stop()
		joined, err := worker.Join(t.Context(), mint(t, w, "office", officeKey), 0, func(pairing.Pending) {})
		if err != nil {
			t.Fatalf("join seedbox to office: %v", err)
		}
		if joined.ControllerName != "office" {
			t.Fatalf("seedbox joined %s, want office", joined.ControllerName)
		}
		worker.Start()
		synctest.Wait()

		homeKey, officeLinkKey, boxKey := key(t, w, "home"), key(t, w, "office"), key(t, w, "seedbox")
		r := readRecord(t, worker)
		if r.Controller.Key != officeLinkKey.String() || len(r.Superseded) != 1 || r.Superseded[0].Key != homeKey.String() || r.Superseded[0].ByKey != officeLinkKey.String() {
			t.Fatalf("seedbox's record is %+v, want controller office %s and home %s superseded by it", r, officeLinkKey, homeKey)
		}
		rows, err := nodeapi.ListNodes(t.Context(), w.Client(), "http://"+w.Node("office").Ready().Listeners.HTTP.String(), officeKey)
		if err != nil {
			t.Fatal(err)
		}
		if i := slices.IndexFunc(rows, func(n nodeapi.Node) bool { return n.Name == "seedbox" }); i < 0 || rows[i].Link.State != "up" {
			t.Fatalf("office lists %+v, want seedbox up", rows)
		}

		home := w.Node("home")
		home.Half().Link.Keep(link.Target{Key: boxKey, Addresses: addresses(t, worker.Ready().Listeners.Link)})
		synctest.Wait()
		refusals := logged(t, home, "link refused")
		if len(refusals) != 1 || refusals[0]["err"] != "the peer reported that key "+officeLinkKey.Fingerprint()+" superseded this Node" {
			t.Fatalf("home logged link refused %v, want one naming office's key %s", refusals, officeLinkKey.Fingerprint())
		}
		home.Half().Link.Drop(boxKey)

		code := mint(t, w, "home", w.Manifest().APIKey)
		worker.Stop()
		worker.Start(func(e *node.Env) { e.Environ = append(e.Environ, "SLIPSTREAM_JOIN_CODE="+code.String()) })
		synctest.Wait()
		if ignored := logged(t, worker, "join code ignored: superseded Controller"); len(ignored) != 1 {
			t.Fatalf("seedbox logged %d join code ignored records for home's code, want 1", len(ignored))
		}
		if r := readRecord(t, worker); r.Controller.Key != officeLinkKey.String() {
			t.Fatalf("home's code moved seedbox to %s, want it still office's %s", r.Controller.Key, officeLinkKey)
		}
	})
}

func addresses(t *testing.T, aps ...netip.AddrPort) link.Addresses {
	t.Helper()
	a, err := link.ParseAddresses(aps)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
