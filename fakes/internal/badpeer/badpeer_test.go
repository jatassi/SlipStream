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
func leech(t *testing.T, mode scenario.BadPeer) (engine.Engine, engine.Handle) {
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
	return e, h
}

// watch reads e's events for d, posting status updates as it goes, and
// returns whether a piece failed its hash check and the latest status of h.
func watch(t *testing.T, e engine.Engine, h engine.Handle, d time.Duration, until func(hashFailed bool) bool) (hashFailed bool, last engine.Status) {
	t.Helper()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	deadline := time.After(d)
	for {
		select {
		case ev := <-e.Events():
			switch ev := ev.(type) {
			case engine.HashFailed:
				hashFailed = hashFailed || ev.Torrent == h
			case engine.Updates:
				for _, s := range ev.Statuses {
					if s.Torrent == h {
						last = s
					}
				}
			}
			if until(hashFailed) {
				return hashFailed, last
			}
		case <-tick.C:
			if err := e.PostUpdates(); err != nil {
				t.Fatal(err)
			}
		case <-deadline:
			return hashFailed, last
		}
	}
}

func TestTheBadPeerSendsCorruptPieces(t *testing.T) {
	e, h := leech(t, scenario.Corrupt)
	if failed, last := watch(t, e, h, time.Minute, func(failed bool) bool { return failed }); !failed {
		t.Fatalf("no piece failed its hash check within a minute of downloading from a corrupt peer; last status %+v", last)
	}
}

// TestTheBadPeerStalls has a leecher wait on a bad peer that holds every
// piece and sends none: in each mode it stays connected or keeps dialing,
// and downloads nothing.
func TestTheBadPeerStalls(t *testing.T) {
	for _, mode := range []scenario.BadPeer{scenario.Stall, scenario.Choke, scenario.Drop} {
		t.Run(string(mode), func(t *testing.T) {
			e, h := leech(t, mode)
			failed, last := watch(t, e, h, 5*time.Second, func(bool) bool { return false })
			if failed || last.Done != 0 || last.LastPayload != (time.Time{}) {
				t.Fatalf("a leecher of a peer that plays %s: hash failed %v, status %+v; want nothing downloaded", mode, failed, last)
			}
			if wantPeers := mode != scenario.Drop; (last.Peers == 1) != wantPeers {
				t.Errorf("a leecher of a peer that plays %s has %d peers, want connected %v", mode, last.Peers, wantPeers)
			}
		})
	}
}
