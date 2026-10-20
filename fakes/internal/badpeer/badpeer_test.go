package badpeer_test

import (
	"crypto/sha1" //nolint:gosec // v1 piece hashes are SHA-1 by the protocol
	"math/rand/v2"
	"net/netip"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/badpeer"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

const pieceLength = 16 << 10

// leech has a libtorrent session download a torrent of five pieces from
// a bad peer that misbehaves as mode, and nothing else.
func leech(t *testing.T, mode scenario.BadPeer) *leecher {
	t.Helper()
	payload := make([]byte, 5*pieceLength-100)
	_, _ = rand.NewChaCha8([32]byte{'b', 'a', 'd'}).Read(payload)
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
	peer, err := badpeer.Listen(t.Context(), m, mode)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = peer.Close() })

	e, err := libtorrent.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.1:0"), Rand: rand.New(rand.NewPCG(1, 2))}) //nolint:gosec // picks listen ports
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	h, err := e.Add(engine.AddParams{Metainfo: meta, SavePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Resume(h); err != nil {
		t.Fatal(err)
	}
	if err := e.ConnectPeer(h, peer.Addr()); err != nil {
		t.Fatal(err)
	}
	return &leecher{e: e, h: h}
}

// leecher is what a test has seen of the leecher's torrent.
type leecher struct {
	e          engine.Engine
	h          engine.Handle
	hashFailed bool
	status     engine.Status
}

// watch reads the leecher's events for up to d, posting status updates as
// it goes, until done holds.
func (l *leecher) watch(t *testing.T, d time.Duration, done func(*leecher) bool) {
	t.Helper()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	deadline := time.After(d)
	for !done(l) {
		select {
		case ev := <-l.e.Events():
			switch ev := ev.(type) {
			case engine.HashFailed:
				l.hashFailed = l.hashFailed || ev.Torrent == l.h
			case engine.Updates:
				for _, s := range ev.Statuses {
					if s.Torrent == l.h {
						l.status = s
					}
				}
			}
		case <-tick.C:
			if err := l.e.PostUpdates(); err != nil {
				t.Fatal(err)
			}
		case <-deadline:
			return
		}
	}
}

func never(*leecher) bool { return false }

func TestTheBadPeerSendsCorruptPieces(t *testing.T) {
	l := leech(t, scenario.Corrupt)
	if l.watch(t, time.Minute, func(l *leecher) bool { return l.hashFailed }); !l.hashFailed {
		t.Fatalf("no piece failed its hash check within a minute of downloading from a corrupt peer; last status %+v", l.status)
	}
}

// TestTheBadPeerStalls has a leecher wait on a bad peer that holds every
// piece and sends none: it stays connected, or keeps dialing a peer that
// hangs up, and downloads nothing.
func TestTheBadPeerStalls(t *testing.T) {
	for _, mode := range []scenario.BadPeer{scenario.Stall, scenario.Choke, scenario.Drop} {
		t.Run(string(mode), func(t *testing.T) {
			l := leech(t, mode)
			if mode != scenario.Drop {
				if l.watch(t, time.Minute, func(l *leecher) bool { return l.status.Peers == 1 }); l.status.Peers != 1 {
					t.Fatalf("a leecher of a peer that plays %s did not connect within a minute: %+v", mode, l.status)
				}
			}
			l.watch(t, 5*time.Second, never)
			if l.hashFailed || l.status.Done != 0 || !l.status.LastPayload.IsZero() {
				t.Fatalf("a leecher of a peer that plays %s: hash failed %v, status %+v; want nothing downloaded", mode, l.hashFailed, l.status)
			}
			if mode != scenario.Drop && l.status.Peers != 1 {
				t.Errorf("a leecher of a peer that plays %s has %d peers after 5 s, want it still connected", mode, l.status.Peers)
			}
		})
	}
}
