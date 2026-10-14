package node

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/link"
	probev1 "github.com/jatassi/slipstream/internal/pb/slipstream/probe/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/system"
)

func dataDirHolds(t *testing.T, d DataDir) []string {
	t.Helper()
	entries, err := os.ReadDir(d.Path())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// caughtUp is whether the Controller half's in-process session has caught
// up, as its control half sees it: the node scope's facts hold the version
// the control half derives now.
func caughtUp(t *testing.T, n *Node) bool {
	t.Helper()
	h := n.half.Load()
	if h == nil || h.controls == nil {
		return false
	}
	nodes, ok := h.peers.(role.Nodes)
	if !ok {
		t.Fatalf("the Controller half's peers %T are no roster", h.peers)
	}
	self := nodes.Self()
	facts, ok := h.controls.NodeFacts(self)
	if !ok {
		return false
	}
	version, err := h.controls.NodeVersion(t.Context(), self)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Equal(facts.GetAppliedVersion(), version)
}

// TestAClaimedSingleMachineNodeReachesLive claims an Unclaimed Node and
// finds its in-process holder half caught up with its control half, then
// again after a restart.
func TestAClaimedSingleMachineNodeReachesLive(t *testing.T) {
	env, _ := testEnv(t)
	synctest.Test(t, func(t *testing.T) {
		n, c := startInMemory(t, env)
		if caughtUp(t, n) {
			t.Fatal("an Unclaimed Node has a holder half")
		}
		claimed := c.claim(n.Ready().SetupCode, "1234")
		if claimed.status != http.StatusCreated {
			t.Fatalf("claimController = %d %s", claimed.status, claimed.body)
		}
		synctest.Wait()
		if !caughtUp(t, n) {
			t.Fatal("the claimed Node's holder half has not caught up with its control half")
		}
		expect(t, c.do(call{method: http.MethodGet, path: "/api/v1/status"}), 200, `{"claim":"controller"}`+"\n")
		for _, want := range []string{"library.db", "library.key", "local.db"} {
			if !slices.Contains(dataDirHolds(t, env.DataDir), want) {
				t.Errorf("the claimed data dir holds %v, want %s among them", dataDirHolds(t, env.DataDir), want)
			}
		}
		if !loggedClaimed(t, env.DataDir, "controller") {
			t.Fatal("the log holds no claimed record with claim controller")
		}

		n.Stop()
		<-n.Done()
		again, _ := startInMemory(t, env)
		synctest.Wait()
		if again.Ready().Claim != system.Controller || !caughtUp(t, again) {
			t.Fatalf("restarted as %s, live %v; want a Controller whose holder half is live", again.Ready().Claim, caughtUp(t, again))
		}
	})
}

// TestAClaimAnswersOnceTheControllerHoldsItsRoles holds the node scope's
// first delivery to the claimed Controller's own holder half, and finds
// claimController answering only after it lands, so an Admin's first call
// to a Role, such as creating an Indexer, finds this Node holding it.
func TestAClaimAnswersOnceTheControllerHoldsItsRoles(t *testing.T) {
	env, _ := testEnv(t)
	gate := role.Def[*probev1.Intent, *probev1.Facts, *probev1.Result]{Name: "gate"}
	env.Test.Always = []role.Name{gate.Name}
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		env.Test.Served = []role.Served{gate.Serve(
			func(ctx context.Context, _ storage.Reader[storage.Library], _ role.NodeID) (*probev1.Intent, error) {
				select {
				case <-release:
					return &probev1.Intent{}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
			func(context.Context, *storage.Tx[storage.Library], role.NodeID, *probev1.Result) error { return nil },
		)}
		n, c := startInMemory(t, env)
		answered := make(chan reply, 1)
		go func() { answered <- c.claim(n.Ready().SetupCode, "1234") }()
		synctest.Wait()
		select {
		case r := <-answered:
			t.Fatalf("claimController answered %d %s before the Controller's holder half held its Roles", r.status, r.body)
		default:
		}
		close(release)
		if r := <-answered; r.status != http.StatusCreated {
			t.Fatalf("claimController = %d %s", r.status, r.body)
		}
		if !caughtUp(t, n) {
			t.Fatal("claimController answered before the Controller's holder half caught up")
		}
	})
}

func loggedClaimed(t *testing.T, d DataDir, claim string) bool {
	t.Helper()
	logged, err := os.ReadFile(filepath.Join(d.LogsDir(), "slipstream.log"))
	if err != nil {
		t.Fatal(err)
	}
	return slices.ContainsFunc(records(t, string(logged)), func(r map[string]any) bool { return r["msg"] == "claimed" && r["claim"] == claim })
}

// TestAnUnclaimedNodeAnswersFromItsFilesAlone holds the rule for callers of
// a Node with no library.db: no credential is read, Account and Admin checks
// answer signed_out, a sign-in matches no Account, and a stray API key never
// stands between a Setup operation and the gate.
func TestAnUnclaimedNodeAnswersFromItsFilesAlone(t *testing.T) {
	env, _ := testEnv(t)
	synctest.Test(t, func(t *testing.T) {
		n, c := startInMemory(t, env)
		key, err := link.OpenKey(env.DataDir.Path())
		if err != nil {
			t.Fatal(err)
		}
		r := n.Ready()
		if r.Claim != system.Unclaimed || r.NodeKey != key.Public().Fingerprint() || !r.Listeners.Link.IsValid() {
			t.Fatalf("ready = claim %s, node_key %s, link %s; want unclaimed, %s and a link listener", r.Claim, r.NodeKey, r.Listeners.Link, key.Public().Fingerprint())
		}
		madeUp := map[string]string{"X-Api-Key": "ssk_made-up"}
		expect(t, c.do(call{method: http.MethodGet, path: "/api/v1/session", session: "made-up"}), 401, signedOut)
		expect(t, c.do(call{method: http.MethodGet, path: "/api/v1/system", headers: madeUp}), 401, signedOut)
		expect(t, c.signIn("admin@example.com", "1234"), 401, pinWrong)
		expect(t, c.do(call{method: http.MethodPost, path: "/api/v1/setup/controller", body: strings.NewReader(claimBody("1234")), headers: map[string]string{"X-Api-Key": "ssk_made-up", "X-Setup-Code": "0000-0000-0000"}}),
			403, problem(403, "setup_code_wrong", "The Setup code is wrong"))
		if got, want := dataDirHolds(t, env.DataDir), []string{"logs", "node.key", "run"}; !slices.Equal(got, want) {
			t.Fatalf("after those calls the data dir holds %v, want %v", got, want)
		}
	})
}
