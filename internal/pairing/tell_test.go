package pairing_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/pairing"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/system"
)

// counted counts the Tells a Worker's pairing sends.
type counted struct {
	pairing.Link
	tells atomic.Int32
}

func (c *counted) Tell(ctx context.Context, t link.Target, s link.Superseded) error {
	c.tells.Add(1)
	return c.Link.Tell(ctx, t, s)
}

// supersede records on c that a restore replaced the Controller holding
// old, with proof.
func (c controller) supersede(t *testing.T, old link.Key, proof []byte) {
	t.Helper()
	if err := c.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Supersede(ctx, tx, c.svc.Self(), old[:], proof)
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTheWorkerStopsTellingOnceAnswered re-pairs box from home to the
// restored Controller, whose Paired carries home's key with a proof: box
// tells home once, at the address it last reached home on, records that
// home answered, and never dials it again.
func TestTheWorkerStopsTellingOnceAnswered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		net := fault.NewNetwork()
		home := startController(t, net, "home", "127.0.0.2", nil)
		restored := startController(t, net, "restored", "127.0.0.4", nil)
		proof := bytes.Repeat([]byte{7}, 36)
		restored.supersede(t, home.node.Key, proof)
		l := &lazy{}
		box := linktest.Start(t, net, "box", []netip.Addr{linktest.WorkerAddr}, linktest.Side{Trust: l, Pair: l.Pair})
		over := &counted{Link: box.Link}
		p, err := pairing.Open(pairing.Config{DataDir: t.TempDir(), Key: box.Key, Link: over, Name: "box", Claims: claims(system.Unclaimed), AwaitDial: 10 * time.Minute, Log: discard})
		if err != nil {
			t.Fatal(err)
		}
		l.trust, l.pair = p, p.Pair
		if _, err := p.Claim(t.Context(), home.mint(t), ignore); err != nil {
			t.Fatal(err)
		}
		if _, err := p.Claim(t.Context(), restored.mint(t), ignore); err != nil {
			t.Fatal(err)
		}
		go func() { _ = p.Tell(t.Context(), time.Hour) }()
		synctest.Wait()

		rec, _ := p.Record()
		if len(rec.Superseded) != 1 || !bytes.Equal(rec.Superseded[0].Proof, proof) || rec.Superseded[0].ToldAt.IsZero() || rec.Superseded[0].Addresses != home.node.Target().Addresses {
			t.Fatalf("after the Tell the record holds %+v, want home's key told with its proof at home's address", rec.Superseded)
		}
		heard := home.side.heard()
		if len(heard) != 1 || heard[0].By != restored.node.Key || !bytes.Equal(heard[0].Proof, proof) || heard[0].Addresses != rec.Controller.Addresses {
			t.Fatalf("home heard %+v, want one claim by the restored Controller with the proof", heard)
		}
		if box.State(home.node.Key) != (link.Down{}) {
			t.Fatalf("box's link to home is %#v, want down: a Tell never goes live", box.State(home.node.Key))
		}

		time.Sleep(3 * time.Hour)
		synctest.Wait()
		if n := over.tells.Load(); n != 1 {
			t.Fatalf("box told home %d times in three superseded_dial periods, want once", n)
		}
	})
}

// TestAnUnknownKeyRefusalLetsTheControllersCodeRepair has home lose its
// library: home refuses box as a key it has no record of, box records the
// refusal, and a Join Code from home, ignored as already joined before,
// pairs box again as a new row and clears the refusal.
func TestAnUnknownKeyRefusalLetsTheControllersCodeRepair(t *testing.T) {
	dir := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		net := fault.NewNetwork()
		home := startController(t, net, "home", "127.0.0.2", nil)
		box, p := worker(t, net, dir, system.Unclaimed)
		go func() { _ = p.Run(t.Context()) }()
		first, err := p.Claim(t.Context(), home.mint(t), ignore)
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if _, err := p.Claim(t.Context(), home.mint(t), ignore); kind(err) != pairing.AlreadyJoined {
			t.Fatalf("home's code while home pins box: %v, want already joined", err)
		}

		home = home.forget(t)
		home.node.Reset(box.Key)
		synctest.Wait()
		time.Sleep(time.Minute)
		synctest.Wait()
		if got := box.State(home.node.Key); got != (link.Down{Refused: linkv1.RefusedReason_REFUSED_REASON_UNKNOWN_KEY}) {
			t.Fatalf("box's link to home is %#v, want refused as an unknown key", got)
		}
		if rec, _ := p.Record(); rec.RefusedUnknownAt.IsZero() || record(t, dir)["refusedUnknownAt"] == nil {
			t.Fatalf("after the refusal pairing.json is %v, want refusedUnknownAt set", record(t, dir))
		}

		again, err := p.Claim(t.Context(), home.mint(t), ignore)
		if err != nil {
			t.Fatal(err)
		}
		if again.NodeID == first.NodeID || again.Controller != home.node.Key {
			t.Fatalf("box joined again as %s of %s, want a new row on home", again.NodeID, again.Controller.Fingerprint())
		}
		rec, _ := p.Record()
		if !rec.RefusedUnknownAt.IsZero() || len(rec.Superseded) != 0 || record(t, dir)["refusedUnknownAt"] != nil {
			t.Fatalf("after the re-pair pairing.json is %v, want no refusal and nothing superseded", record(t, dir))
		}
		if home.svc.Admit(box.Key) != (link.Pinned{}) {
			t.Fatalf("home admits box as %#v, want Pinned", home.svc.Admit(box.Key))
		}
	})
}

// schemaTwo is a schema 2 pairing.json naming home and the key it
// replaced, with each superseded entry's fields extended by extra.
func schemaTwo(t *testing.T, home, old link.Key, extra string) string {
	t.Helper()
	k := func(key link.Key) string {
		b, err := json.Marshal(key)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	return `{"schema":2,"nodeId":"0193a6e2-7b1c-7d2e-8f3a-4b5c6d7e8f90","controller":{"key":` + k(home) + `,"name":"home","addresses":["127.0.0.2:8485"]},` +
		`"superseded":[{"key":` + k(old) + `,"byKey":` + k(home) + `,"at":"2026-01-02T03:04:05Z"` + extra + `}],"pairedAt":"2026-01-02T03:04:05Z"}`
}

var (
	homeKey = link.NewPrivateKey([32]byte{1}).Public()
	oldKey  = link.NewPrivateKey([32]byte{2}).Public()
)

func writeRecord(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, pairing.RecordFile), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestASchemaTwoRecordReadsAsThree reads a schema 2 record as it was, and
// writes it as schema 3 at the next change.
func TestASchemaTwoRecordReadsAsThree(t *testing.T) {
	dir := writeRecord(t, schemaTwo(t, homeKey, oldKey, ""))
	p, err := pairing.Open(pairing.Config{DataDir: dir, Log: discard})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	rec, ok := p.Record()
	want := pairing.Superseded{Key: oldKey, By: homeKey, At: at}
	if !ok || rec.Controller.Key != homeKey || rec.ControllerName != "home" || !rec.PairedAt.Equal(at) || len(rec.Superseded) != 1 || !sameSuperseded(rec.Superseded[0], want) || !rec.RefusedUnknownAt.IsZero() {
		t.Fatalf("the schema 2 record reads as %+v", rec)
	}

	proof := bytes.Repeat([]byte{7}, 36)
	p.Remember([]link.SupersededProof{{Key: oldKey, Proof: proof}})
	if got := record(t, dir)["schema"]; got != float64(3) {
		t.Fatalf("pairing.json after a change is schema %v, want 3", got)
	}
	reopened, err := pairing.Open(pairing.Config{DataDir: dir, Log: discard})
	if err != nil {
		t.Fatal(err)
	}
	want.Proof = proof
	if again, _ := reopened.Record(); again.NodeID != rec.NodeID || again.Controller != rec.Controller || len(again.Superseded) != 1 || !sameSuperseded(again.Superseded[0], want) {
		t.Fatalf("the rewritten record reads as %+v, want %+v with the proof", again, rec)
	}
}

// TestASchemaTwoRecordWithAProofIsRefused refuses a schema 2 record that
// carries a field only schema 3 writes.
func TestASchemaTwoRecordWithAProofIsRefused(t *testing.T) {
	for _, c := range []struct{ name, extra string }{
		{"a proof", `,"proof":"BwcH"`},
		{"addresses", `,"addresses":["127.0.0.2:8485"]`},
		{"toldAt", `,"toldAt":"2026-01-02T03:04:05Z"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := writeRecord(t, schemaTwo(t, homeKey, oldKey, c.extra))
			_, err := pairing.Open(pairing.Config{DataDir: dir})
			want := "pairing record " + filepath.Join(dir, pairing.RecordFile) + ": a schema 3 field in a schema 2 record"
			if _, ok := errors.AsType[*pairing.RecordError](err); !ok || err.Error() != want {
				t.Fatalf("Open = %v, want a RecordError %q", err, want)
			}
		})
	}
}
