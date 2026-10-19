package libtorrent_test

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// A Node's stop closes its session among its other stops under one
// deadline, which the e2e worlds set in their scenario, so one session
// closing mid-download takes a small share of it.
func TestClosingMidDownloadTakesATenthOfAStopDeadline(t *testing.T) {
	home, err := scenario.Load("home")
	if err != nil {
		t.Fatal(err)
	}
	bound := home.Timings.Node.StopDeadline / 10

	payload := make([]byte, payloadSize)
	_, _ = rand.NewChaCha8([32]byte{'c', 'l', 'o', 's', 'e'}).Read(payload)
	meta := singleFileTorrent(t, "payload.bin", payload, 256<<10)

	leecher, st := partlyDownloaded(t, meta, payload)
	start := time.Now()
	err = leecher.Close()
	took := time.Since(start)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if took > bound {
		t.Fatalf("Close with %d of %d bytes done took %s, want under %s, a tenth of the home world's stop deadline", st.Done, st.Wanted, took, bound)
	}
	t.Logf("Close with %d of %d bytes done took %s", st.Done, st.Wanted, took)
}

// partlyDownloaded has a session download payload from a seeder until some
// of it is done.
func partlyDownloaded(t *testing.T, meta, payload []byte) (engine.Engine, engine.Status) {
	t.Helper()
	seedDir, leechDir := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(seedDir, "payload.bin"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	seeder, leecher := open(t), open(t)
	if _, err := seeder.Add(engine.AddParams{Metainfo: meta, SavePath: seedDir, SeedMode: true}); err != nil {
		t.Fatalf("seeder add: %v", err)
	}
	h, err := leecher.Add(engine.AddParams{Metainfo: meta, SavePath: leechDir})
	if err != nil {
		t.Fatalf("leecher add: %v", err)
	}
	seederAddr, err := seeder.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}
	if err := leecher.ConnectPeer(h, seederAddr); err != nil {
		t.Fatalf("connect peer: %v", err)
	}
	var st engine.Status
	for deadline := time.Now().Add(60 * time.Second); st.Done == 0; time.Sleep(time.Millisecond) {
		if st, err = leecher.Status(h); err != nil {
			t.Fatalf("status: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("status = %+v, want a piece done", st)
		}
	}
	return leecher, st
}
