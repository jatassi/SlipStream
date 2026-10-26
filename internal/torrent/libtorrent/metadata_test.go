package libtorrent_test

import (
	"errors"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// TestAFetchRemovedBeforeItsMetadataIsReadLosesNoEvent holds the alert
// pump back with a full event queue while a metadata fetch gets its info
// dictionary and is removed, so the pump reads the metadata's alert only
// once the torrent is gone. The batch it came in still reaches the reader.
func TestAFetchRemovedBeforeItsMetadataIsReadLosesNoEvent(t *testing.T) {
	dir := t.TempDir()
	payload := make([]byte, 1<<20)
	_, _ = rand.NewChaCha8([32]byte{'G'}).Read(payload)
	if err := os.WriteFile(filepath.Join(dir, "Gone.mkv"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	meta := singleFileTorrent(t, "Gone.mkv", payload, 64<<10)
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	seeder := open(t)
	started(t, seeder, engine.AddParams{Metainfo: meta, SavePath: dir, SeedMode: true})
	addr, err := seeder.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}

	e := open(t)
	for range 1000 {
		if err := e.PostUpdates(); err != nil {
			t.Fatal(err)
		}
	}
	until(t, "the event queue is full", func() bool { return len(e.Events()) == cap(e.Events()) })
	h, err := e.FetchMetadata(engine.MetadataParams{Link: magnet.Link{InfoHash: m.InfoHash}, SavePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ConnectPeer(h, addr); err != nil {
		t.Fatal(err)
	}
	until(t, "the metadata arrives", func() bool { _, err := e.Files(h); return !errors.Is(err, engine.ErrNoMetadata) })
	if err := e.Remove(h, false); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	timeout := time.After(60 * time.Second)
	for {
		select {
		case ev := <-e.Events():
			switch ev := ev.(type) {
			case engine.Dropped:
				t.Fatal("the engine dropped the batch that carried the fetch's removal")
			case engine.Removed:
				if ev.Torrent == h {
					return
				}
			}
		case <-timeout:
			t.Fatal("timed out waiting for the fetch's removal")
		}
	}
}
