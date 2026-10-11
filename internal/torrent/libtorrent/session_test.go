package libtorrent_test

import (
	"crypto/sha256"
	"errors"
	"math/rand/v2"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

const payloadSize = 8 << 20

func open(t *testing.T, addr string) engine.Engine {
	t.Helper()
	e, err := libtorrent.Open(engine.Config{ListenInterface: netip.MustParseAddrPort(addr)})
	if err != nil {
		t.Fatalf("open %s: %v", addr, err)
	}
	t.Cleanup(func() { e.Close() })
	return e
}

func waitFor[E engine.Event](t *testing.T, events <-chan engine.Event, match func(E) bool) E {
	t.Helper()
	timeout := time.After(60 * time.Second)
	for {
		select {
		case ev := <-events:
			if e, ok := ev.(E); ok && match(e) {
				return e
			}
		case <-timeout:
			var zero E
			t.Fatalf("timed out waiting for %T", zero)
		}
	}
}

func TestEngineRoundTrip(t *testing.T) {
	payload := make([]byte, payloadSize)
	rand.NewChaCha8([32]byte{'s', 'l', 'i', 'p'}).Read(payload)
	meta := singleFileTorrent("payload.bin", payload, 256<<10)

	seedDir, leechDir := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(seedDir, "payload.bin"), payload, 0o644); err != nil {
		t.Fatal(err)
	}

	seeder := open(t, "127.0.0.1:0")
	leecher := open(t, "127.0.0.1:0")

	seedHandle, err := seeder.Add(engine.AddParams{Metainfo: meta, SavePath: seedDir, SeedMode: true})
	if err != nil {
		t.Fatalf("seeder add: %v", err)
	}
	leechHandle, err := leecher.Add(engine.AddParams{Metainfo: meta, SavePath: leechDir})
	if err != nil {
		t.Fatalf("leecher add: %v", err)
	}
	seederAddr, err := seeder.ListenAddr()
	if err != nil {
		t.Fatalf("seeder listen addr: %v", err)
	}
	if err := leecher.ConnectPeer(leechHandle, seederAddr); err != nil {
		t.Fatalf("connect peer: %v", err)
	}

	waitFor(t, leecher.Events(), func(e engine.Finished) bool { return e.Torrent == leechHandle })

	got, err := os.ReadFile(filepath.Join(leechDir, "payload.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(got) != sha256.Sum256(payload) {
		t.Fatal("downloaded payload differs from the seeded payload")
	}
	st, err := leecher.Status(leechHandle)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if st.State != engine.StateSeeding || st.Done != payloadSize || st.Wanted != payloadSize {
		t.Fatalf("status = %+v, want seeding with %d of %d bytes done", st, payloadSize, payloadSize)
	}

	for _, side := range []struct {
		name string
		e    engine.Engine
		h    engine.Handle
	}{{"leecher", leecher, leechHandle}, {"seeder", seeder, seedHandle}} {
		if err := side.e.Remove(side.h); err != nil {
			t.Fatalf("%s remove: %v", side.name, err)
		}
		waitFor(t, side.e.Events(), func(e engine.Removed) bool { return e.Torrent == side.h })
		if err := side.e.Close(); err != nil {
			t.Fatalf("%s close: %v", side.name, err)
		}
	}
}

func TestEngineRejectsBadInput(t *testing.T) {
	e := open(t, "127.0.0.1:0")

	_, err := e.Add(engine.AddParams{Metainfo: []byte("d4:infoi1e"), SavePath: t.TempDir()})
	if !errors.Is(err, engine.CategoryBdecode) {
		t.Fatalf("add garbage metainfo: err = %v, want a bdecode error", err)
	}

	unknown := engine.Handle(4242)
	if _, err := e.Status(unknown); !errors.Is(err, engine.ErrUnknownTorrent) {
		t.Fatalf("status of unknown handle: err = %v, want %v", err, engine.ErrUnknownTorrent)
	}
	if err := e.Remove(unknown); !errors.Is(err, engine.ErrUnknownTorrent) {
		t.Fatalf("remove of unknown handle: err = %v, want %v", err, engine.ErrUnknownTorrent)
	}
	if err := e.ConnectPeer(unknown, netip.MustParseAddrPort("127.0.0.1:1")); !errors.Is(err, engine.ErrUnknownTorrent) {
		t.Fatalf("connect peer on unknown handle: err = %v, want %v", err, engine.ErrUnknownTorrent)
	}

	if _, err := e.ListenAddr(); err != nil {
		t.Fatalf("listen addr after rejected calls: %v", err)
	}
	if err := e.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := e.Close(); !errors.Is(err, engine.ErrClosed) {
		t.Fatalf("second close: err = %v, want %v", err, engine.ErrClosed)
	}
	if _, err := e.Status(unknown); !errors.Is(err, engine.ErrClosed) {
		t.Fatalf("status after close: err = %v, want %v", err, engine.ErrClosed)
	}
	drained := make(chan struct{})
	go func() {
		for range e.Events() {
		}
		close(drained)
	}()
	select {
	case <-drained:
	case <-time.After(5 * time.Second):
		t.Fatal("events channel still open after close")
	}
}
