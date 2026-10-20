package leecher_test

import (
	"crypto/sha1" //nolint:gosec // v1 piece hashes are SHA-1 by the protocol
	"math/rand/v2"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/leecher"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

const pieceLength = 16 << 10

// seeding is a libtorrent session on loopback that seeds one torrent and
// watches incoming peers, and that torrent's info hash.
func seeding(t *testing.T) (engine.Engine, bencode.InfoHash) {
	t.Helper()
	payload := make([]byte, 3*pieceLength-100)
	_, _ = rand.NewChaCha8([32]byte{'l', 'e', 'e', 'c', 'h'}).Read(payload)
	var pieces []byte
	for off := 0; off < len(payload); off += pieceLength {
		sum := sha1.Sum(payload[off:min(off+pieceLength, len(payload))]) //nolint:gosec // v1 piece hashes
		pieces = append(pieces, sum[:]...)
	}
	meta, err := bencode.Encode(map[string]any{"info": map[string]any{"length": len(payload), "name": "Orchard.mkv", "piece length": pieceLength, "pieces": pieces}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Orchard.mkv"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	e, err := libtorrent.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.1:0"), Rand: rand.New(rand.NewPCG(1, 2))}) //nolint:gosec // picks listen ports
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	if err := e.Configure(engine.Limits{WatchIncoming: true}); err != nil {
		t.Fatal(err)
	}
	h, err := e.Add(engine.AddParams{Metainfo: meta, SavePath: dir, SeedMode: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Resume(h); err != nil {
		t.Fatal(err)
	}
	await(t, e, "the torrent starting", func(ev engine.Event) bool { r, ok := ev.(engine.Resumed); return ok && r.Torrent == h })
	return e, m.InfoHash
}

// await reads e's events for up to a minute until match takes one.
func await(t *testing.T, e engine.Engine, what string, match func(engine.Event) bool) {
	t.Helper()
	deadline := time.After(time.Minute)
	for {
		select {
		case ev := <-e.Events():
			if match(ev) {
				return
			}
		case <-deadline:
			t.Fatalf("the session sent no event for %s within a minute", what)
		}
	}
}

func TestALoopbackLeecherProvesThePortUnderTheDivergence(t *testing.T) {
	e, hash := seeding(t)
	addr, err := e.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}
	if err := leecher.Hello(t.Context(), addr, hash); err != nil {
		t.Fatalf("a session running the torrent turned the leecher away: %v", err)
	}
	var from netip.Addr
	await(t, e, "an incoming peer", func(ev engine.Event) bool {
		p, ok := ev.(engine.IncomingPeer)
		from = p.Addr
		return ok
	})
	if !downloading.PublicPeer(true)(from) {
		t.Errorf("under loopback-peers, the leecher at %s does not prove the port", from)
	}
	if downloading.PublicPeer(false)(from) {
		t.Errorf("without loopback-peers, the leecher at %s proves the port", from)
	}
}

func TestALeecherOfATorrentTheNodeDoesNotRunIsTurnedAway(t *testing.T) {
	e, _ := seeding(t)
	addr, err := e.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}
	if err := leecher.Hello(t.Context(), addr, bencode.InfoHash{1}); err == nil {
		t.Fatal("a session answered a leecher for a torrent it does not hold")
	}
}
