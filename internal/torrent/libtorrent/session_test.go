package libtorrent_test

import (
	"crypto/sha256"
	"errors"
	"math/rand/v2"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

const payloadSize = 8 << 20

func open(t *testing.T) engine.Engine {
	t.Helper()
	e, err := libtorrent.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.1:0"), Rand: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))}) //nolint:gosec // picks listen ports
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if err := e.Close(); err != nil && !errors.Is(err, engine.ErrClosed) {
			t.Errorf("close: %v", err)
		}
	})
	return e
}

// started adds a torrent and resumes it.
func started(t *testing.T, e engine.Engine, p engine.AddParams) engine.Handle {
	t.Helper()
	h, err := e.Add(p)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := e.Resume(h); err != nil {
		t.Fatalf("resume: %v", err)
	}
	return h
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
	_, _ = rand.NewChaCha8([32]byte{'s', 'l', 'i', 'p'}).Read(payload)
	meta := singleFileTorrent(t, "payload.bin", payload, 256<<10)

	seedDir, leechDir := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(seedDir, "payload.bin"), payload, 0o644); err != nil {
		t.Fatal(err)
	}

	seeder := open(t)
	leecher := open(t)

	seedHandle := started(t, seeder, engine.AddParams{Metainfo: meta, SavePath: seedDir, SeedMode: true})
	leechHandle := started(t, leecher, engine.AddParams{Metainfo: meta, SavePath: leechDir})
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
	awaitUploaded(t, seeder, seedHandle, payloadSize)

	for _, side := range []struct {
		name string
		e    engine.Engine
		h    engine.Handle
	}{{"leecher", leecher, leechHandle}, {"seeder", seeder, seedHandle}} {
		if err := side.e.Remove(side.h, false); err != nil {
			t.Fatalf("%s remove: %v", side.name, err)
		}
		waitFor(t, side.e.Events(), func(e engine.Removed) bool { return e.Torrent == side.h })
		if err := side.e.Close(); err != nil {
			t.Fatalf("%s close: %v", side.name, err)
		}
	}
}

// awaitUploaded waits until torrent h on e has uploaded at least n bytes.
func awaitUploaded(t *testing.T, e engine.Engine, h engine.Handle, n int64) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(100 * time.Millisecond) {
		st, err := e.Status(h)
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		if st.Totals.Uploaded >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("status = %+v, want at least %d bytes uploaded", st, n)
		}
	}
}

func TestEngineSaysWhyAFixedPortDoesNotListen(t *testing.T) {
	_, err := libtorrent.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("192.0.2.1:6881")})
	const want = "shim error 3: not listening on 192.0.2.1:6881: system error "
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("open on an address the host does not have: err = %v, want it to start %q", err, want)
	}
}

func TestEngineRejectsBadInput(t *testing.T) {
	e := open(t)

	_, err := e.Add(engine.AddParams{Metainfo: []byte("d4:infoi1e"), SavePath: t.TempDir()})
	if !errors.Is(err, engine.CategoryBdecode) {
		t.Fatalf("add garbage metainfo: err = %v, want a bdecode error", err)
	}

	unknown := engine.Handle(4242)
	if _, err := e.Status(unknown); !errors.Is(err, engine.ErrUnknownTorrent) {
		t.Fatalf("status of unknown handle: err = %v, want %v", err, engine.ErrUnknownTorrent)
	}
	if err := e.Remove(unknown, false); !errors.Is(err, engine.ErrUnknownTorrent) {
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

func TestFilesListThePathsLibtorrentSanitized(t *testing.T) {
	data := make([]byte, 3000)
	meta := torrent(t, map[string]any{
		"name": "Orchard",
		"files": []any{
			map[string]any{"length": 1000, "path": []any{"Extras", "..", "trailer.mkv"}},
			map[string]any{"length": 2000, "path": []any{"feature.mkv"}},
		},
	}, data, 16<<10)
	e := open(t)
	h, err := e.Add(engine.AddParams{Metainfo: meta, SavePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.Files(h)
	if err != nil {
		t.Fatal(err)
	}
	want := []engine.File{
		{Index: 0, Path: "Orchard/Extras/_/trailer.mkv", Size: 1000},
		{Index: 1, Path: "Orchard/feature.mkv", Size: 2000},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Files = %+v, want %+v", got, want)
	}
	if _, err := e.Files(h + 1); !errors.Is(err, engine.ErrUnknownTorrent) {
		t.Fatalf("Files of an unknown torrent err = %v", err)
	}
}
