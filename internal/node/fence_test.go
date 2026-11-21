package node

import (
	"context"
	"crypto/rand"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/storage"
)

type supersededPage struct {
	State       string `json:"state"`
	Worker      string `json:"worker"`
	Replacement *struct {
		Key string `json:"key"`
	} `json:"replacement"`
}

func supersededOn(t *testing.T, c client, apiKey string) supersededPage {
	t.Helper()
	return decode[supersededPage](t, c.admin(apiKey, http.MethodGet, "/api/v1/superseded", ""), http.StatusOK)
}

// historyItems is the key of each Health item n's History records.
func historyItems(t *testing.T, n *Node) []string {
	t.Helper()
	var keys []string
	err := n.lib.Load().db.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := r.QueryContext(ctx, `SELECT key FROM history_item ORDER BY event_id`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			if err := rows.Scan(&k); err != nil {
				return err
			}
			keys = append(keys, k)
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	return keys
}

// ownIntent is the Downloading intent n's Controller derives for its own
// Node.
func ownIntent(t *testing.T, n *Node) (keepAll, pauseAll bool) {
	t.Helper()
	h := n.half.Load()
	derive := downloadclients.Derive{Keeps: h.reconciling, HoldingsPage: n.cfg.Timings.Downloading.HoldingsPage}
	err := n.lib.Load().db.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		in, err := derive.Intent(ctx, r, h.self)
		keepAll, pauseAll = in.GetKeepAll(), in.GetPauseAll()
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return keepAll, pauseAll
}

// TestARestoreOnANewMachineFencesTheOldControllerThroughAWorker restores
// home's Backup on spare while home still runs, then pairs box, home's
// Worker, with spare. box carries spare's proof to home: home pauses every
// torrent on its own Node, answers superseded on every Admin page but its
// own, and records no History row.
func TestARestoreOnANewMachineFencesTheOldControllerThroughAWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		net := fault.NewNetwork()
		homeEnv, homeHost := machine(t, net, "127.0.0.2", "home")
		boxEnv, boxHost := machine(t, net, "127.0.0.3", "box")
		spareEnv, spareHost := machine(t, net, "127.0.0.4", "spare")
		boxEnv.Restart = func() error { return nil }
		home, hc := startOn(t, homeEnv, homeHost)
		apiKey, code := claimedHome(t, home, hc)
		box, bc := startOn(t, boxEnv, boxHost)
		joinBox(t, box, bc, boxEnv.DataDir, code, apiKey)
		synctest.Wait()
		b := takeBackup(t, home)
		if _, err := RestoreOffline(t.Context(), spareEnv, restoreFlags{file: b.file, oldOff: true}, b.words); err != nil {
			t.Fatal(err)
		}
		_, sc := startOn(t, spareEnv, spareHost)
		synctest.Wait()
		before := len(historyItems(t, home))

		reply, err := Call(t.Context(), boxHost, boxEnv.DataDir, pairing.JoinOp, pairing.JoinRequest{Code: mintCode(t, sc, apiKey)})
		if err != nil || reply.Refused != nil {
			t.Fatalf("box's join op with spare's code = %+v, %v; want joined", reply, err)
		}
		<-box.Done()
		startOn(t, boxEnv, boxHost)
		synctest.Wait()

		if page := supersededOn(t, hc, apiKey); page.State != "proven" || page.Worker != "box" || page.Replacement == nil {
			t.Fatalf("home's Superseded page reads %+v, want proven by box with spare as the replacement", page)
		}
		if r := hc.do(call{method: http.MethodGet, path: "/api/v1/status"}); r.status != http.StatusOK || !decode[struct {
			Superseded bool `json:"superseded"`
		}](t, r, http.StatusOK).Superseded {
			t.Fatalf("home's status = %d %s, want superseded", r.status, r.body)
		}
		expect(t, hc.admin(apiKey, http.MethodGet, "/api/v1/nodes", ""), 409, problem(409, "superseded", "This Controller was replaced"))
		if keepAll, pauseAll := ownIntent(t, home); !keepAll || !pauseAll {
			t.Fatalf("home's own Downloading intent keeps all %v, pauses all %v; want both", keepAll, pauseAll)
		}
		if added := historyItems(t, home)[before:]; len(added) != 0 {
			t.Fatalf("home's History recorded %v, want no row for the fence", added)
		}
		if !listedOn(t, sc, apiKey, "box") {
			t.Fatal("spare doesn't list box")
		}
	})
}

// TestAHostileWorkerCantFenceAHealthyController has box, paired with
// home, tell home a restore replaced it with a proof box made up. home
// records the claim as unproven and keeps running.
func TestAHostileWorkerCantFenceAHealthyController(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		net := fault.NewNetwork()
		homeEnv, homeHost := machine(t, net, "127.0.0.2", "home")
		boxEnv, boxHost := machine(t, net, "127.0.0.3", "box")
		home, hc := startOn(t, homeEnv, homeHost)
		apiKey, code := claimedHome(t, home, hc)
		box, bc := startOn(t, boxEnv, boxHost)
		joinBox(t, box, bc, boxEnv.DataDir, code, apiKey)
		synctest.Wait()
		before := len(historyItems(t, home))

		var by link.Key
		proof := make([]byte, 36)
		_, _ = rand.Read(by[:])
		_, _ = rand.Read(proof)
		at, err := link.ParseAddresses([]netip.AddrPort{home.Ready().Listeners.Link})
		if err != nil {
			t.Fatal(err)
		}
		forged := link.Superseded{By: by, Addresses: at, Proof: proof}
		if err := box.link.Tell(t.Context(), link.Target{Key: home.key.Public(), Addresses: at}, forged); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()

		if page := supersededOn(t, hc, apiKey); page.State != "unproven" || page.Worker != "box" {
			t.Fatalf("home's Superseded page reads %+v, want box's claim unproven", page)
		}
		if !listedOn(t, hc, apiKey, "box") {
			t.Fatal("home refused its Nodes page after a forged proof")
		}
		if keepAll, pauseAll := ownIntent(t, home); pauseAll {
			t.Fatalf("home's own Downloading intent keeps all %v, pauses all %v; want no pause", keepAll, pauseAll)
		}
		if added := historyItems(t, home)[before:]; len(added) != 1 || !strings.HasSuffix(added[0], ".unproven") {
			t.Fatalf("home's History recorded %v, want only the Health Error of an unproven claim", added)
		}
	})
}
