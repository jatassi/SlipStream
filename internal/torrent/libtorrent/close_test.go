package libtorrent_test

import (
	"encoding/binary"
	"math/rand/v2"
	"net"
	"net/netip"
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
	// The home world's Nodes peer over TCP alone (its tcp-peers
	// divergence); a uTP peer adds about 0.8 s to a close.
	if err := leecher.Configure(engine.Limits{}); err != nil {
		t.Fatal(err)
	}
	started(t, seeder, engine.AddParams{Metainfo: meta, SavePath: seedDir, SeedMode: true})
	h := started(t, leecher, engine.AddParams{Metainfo: meta, SavePath: leechDir})
	seederAddr, err := seeder.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}
	if err := leecher.ConnectPeer(h, seederAddr); err != nil {
		t.Fatalf("connect peer: %v", err)
	}
	var st engine.Status
	for deadline := time.Now().Add(60 * time.Second); st.Done == 0; time.Sleep(time.Millisecond) {
		if s, ok := updated(t, leecher, h); ok {
			st = s
		}
		if time.Now().After(deadline) {
			t.Fatalf("status = %+v, want a piece done", st)
		}
	}
	return leecher, st
}

// slowUDPTracker is a BEP 15 tracker on loopback that responds to a
// stopped announce after delay, as a distant one does, and to every other
// request at once. stopped receives each stopped it responded to.
func slowUDPTracker(t *testing.T, delay time.Duration) (url string, stopped <-chan struct{}) {
	t.Helper()
	conn, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:0")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	answered := make(chan struct{}, 8)
	go func() {
		buf := make([]byte, 2048)
		for {
			n, from, err := conn.ReadFromUDPAddrPort(buf)
			if err != nil {
				return
			}
			p := buf[:n]
			if n < 16 {
				continue
			}
			reply := binary.BigEndian.AppendUint32(nil, binary.BigEndian.Uint32(p[8:]))
			reply = append(reply, p[12:16]...)
			if binary.BigEndian.Uint32(p[8:]) == 0 {
				_, _ = conn.WriteToUDPAddrPort(binary.BigEndian.AppendUint64(reply, 7), from)
				continue
			}
			for _, v := range []uint32{1800, 0, 1} {
				reply = binary.BigEndian.AppendUint32(reply, v)
			}
			if n < 98 || binary.BigEndian.Uint32(p[80:]) != 3 {
				_, _ = conn.WriteToUDPAddrPort(reply, from)
				continue
			}
			time.AfterFunc(delay, func() {
				answered <- struct{}{}
				_, _ = conn.WriteToUDPAddrPort(reply, from)
			})
		}
	}()
	return "udp://" + conn.LocalAddr().String() + "/announce", answered
}

// TestCloseWaitsForAUDPTrackersStoppedResponse closes a session seeding
// from a UDP tracker that takes 300 ms to respond to stopped. The close
// waits for that response: libtorrent's abort closes the socket it would
// arrive on and then waits out its 5 s stop_tracker_timeout.
func TestCloseWaitsForAUDPTrackersStoppedResponse(t *testing.T) {
	url, stopped := slowUDPTracker(t, 300*time.Millisecond)
	dir := t.TempDir()
	payload := make([]byte, 1<<16)
	_, _ = rand.NewChaCha8([32]byte{'u', 'd', 'p'}).Read(payload)
	if err := os.WriteFile(filepath.Join(dir, "udp.bin"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	e := open(t)
	h := started(t, e, engine.AddParams{Metainfo: trackedTorrent(t, "udp.bin", payload, 16<<10, url), SavePath: dir, SeedMode: true})
	waitFor(t, e.Events(), func(r engine.TrackerReplied) bool { return r.Torrent == h })
	start := time.Now()
	if err := e.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("Close took %s, want it done soon after the tracker's 300 ms response to stopped", took)
	}
	select {
	case <-stopped:
	default:
		t.Error("the tracker responded to no stopped before Close returned")
	}
}
