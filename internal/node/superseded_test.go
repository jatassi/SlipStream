package node

import (
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/lifecycle"
)

// wipedBox is a data dir as Wipe and join leaves it before its restart:
// the old Controller's library, its keys and Backups index, and a wipe
// record holding code.
func wipedBox(t *testing.T, env Env, code string) {
	t.Helper()
	seedMigrated(t, env, env.DataDir, "old")
	for _, name := range []string{"backup.key", "backups.json"} {
		writeFile(t, filepath.Join(env.DataDir.Path(), name), name)
	}
	if err := env.DataDir.lifecycle(0).Begin(&lifecycle.Wipe{Stage: lifecycle.Deleting, JoinCode: code}); err != nil {
		t.Fatal(err)
	}
}

// assertWiped checks that the library, its keys, the Backups index and
// the record are gone, and node.key stayed.
func assertWiped(t *testing.T, d DataDir) {
	t.Helper()
	holds := dataDirHolds(t, d)
	for _, name := range append(slices.Clone(lifecycle.WipeFiles), lifecycle.FileName) {
		if slices.Contains(holds, name) {
			t.Fatalf("the wiped data dir still holds %s: %v", name, holds)
		}
	}
	if !slices.Contains(holds, "node.key") {
		t.Fatalf("the wiped data dir holds %v, want node.key kept", holds)
	}
}

// listedOn is whether home's Nodes page lists the Node named name.
func listedOn(t *testing.T, hc client, apiKey, name string) bool {
	t.Helper()
	list := decode[struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	}](t, hc.admin(apiKey, http.MethodGet, "/api/v1/nodes", ""), http.StatusOK).Nodes
	return slices.ContainsFunc(list, func(n struct {
		Name string `json:"name"`
	}) bool {
		return n.Name == name
	})
}

// TestWipeAndJoinJoinsAtTheNextStartAcrossACrash stops a wiped Node's
// start before each delete, before the record ends and before the join,
// then starts it again. Each time the library, its keys and the Backups
// index are gone and node.key stays. Every stop before the record ends
// joins home with the record's code as a new Node; a stop after it loses
// the code, so the Node waits Unclaimed.
func TestWipeAndJoinJoinsAtTheNextStartAcrossACrash(t *testing.T) {
	net := fault.NewNetwork()
	homeEnv, homeHost := machine(t, net, "127.0.0.2", "home")
	steps := append(slices.Clone(lifecycle.WipeFiles[:1]), "library.key", "backup.key", "backups.json", WipeFinish, WipeJoin)
	synctest.Test(t, func(t *testing.T) {
		home, hc := startOn(t, homeEnv, homeHost)
		apiKey, _ := claimedHome(t, home, hc)
		for i, step := range steps {
			name := fmt.Sprintf("old-%d", i)
			boxEnv, boxHost := machine(t, net, fmt.Sprintf("127.0.0.%d", 10+i), name)
			wipedBox(t, boxEnv, mintCode(t, hc, apiKey))
			boxEnv.Test.BeforeWipeStep = func(s string) error {
				if s == step {
					return errCrash
				}
				return nil
			}
			if _, err := Start(t.Context(), boxEnv); !errors.Is(err, errCrash) {
				t.Fatalf("a start stopped before %s = %v, want the crash", step, err)
			}
			boxEnv.Test.BeforeWipeStep = nil
			box, bc := startOn(t, boxEnv, boxHost)
			synctest.Wait()
			assertWiped(t, boxEnv.DataDir)
			if step == WipeJoin {
				expect(t, bc.do(call{method: http.MethodGet, path: "/api/v1/status"}), 200, statusOf(box, "unclaimed"))
				continue
			}
			expect(t, bc.do(call{method: http.MethodGet, path: "/api/v1/status"}), 200, statusOf(box, "worker"))
			if !listedOn(t, hc, apiKey, name) {
				t.Fatalf("stopped before %s, %s joined but home doesn't list it", step, name)
			}
		}
	})
}

// TestARefusedJoinCodeLeavesTheWipedNodeUnclaimed wipes a Node whose
// record holds a Join Code for a Controller nobody runs: the wipe
// finishes, the join is refused once in the log, and the Node waits
// Unclaimed with a Setup code.
func TestARefusedJoinCodeLeavesTheWipedNodeUnclaimed(t *testing.T) {
	net := fault.NewNetwork()
	boxEnv, boxHost := machine(t, net, "127.0.0.3", "old")
	code, err := joincode.New(make([]byte, 32), [joincode.SecretSize]byte{7}, []netip.AddrPort{netip.MustParseAddrPort("127.0.0.9:8485")})
	if err != nil {
		t.Fatal(err)
	}
	synctest.Test(t, func(t *testing.T) {
		wipedBox(t, boxEnv, code.String())
		box, bc := startOn(t, boxEnv, boxHost)
		time.Sleep(Defaults(boxEnv.Types, boxEnv.Install).Timings.Link.AwaitDial + time.Minute)
		synctest.Wait()
		assertWiped(t, boxEnv.DataDir)
		expect(t, bc.do(call{method: http.MethodGet, path: "/api/v1/status"}), 200, statusOf(box, "unclaimed"))
		if box.Ready().SetupCode == "" {
			t.Fatal("the wiped Node gave no Setup code")
		}
		logged, err := os.ReadFile(filepath.Join(boxEnv.DataDir.LogsDir(), "slipstream.log"))
		if err != nil {
			t.Fatal(err)
		}
		refused := 0
		for _, r := range records(t, string(logged)) {
			if r["msg"] == "join refused" {
				refused++
			}
		}
		if refused != 1 {
			t.Fatalf("the log holds %d join refused records, want 1", refused)
		}
	})
}

func mintCode(t *testing.T, hc client, apiKey string) string {
	t.Helper()
	return decode[struct {
		Code string `json:"code"`
	}](t, hc.admin(apiKey, http.MethodPost, "/api/v1/join-codes", ""), http.StatusCreated).Code
}
