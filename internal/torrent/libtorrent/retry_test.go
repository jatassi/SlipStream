package libtorrent_test

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/torrent/engine"
)

const retryPiece = 16 << 10

// seedingThenFailed has a session seed a release, moves its file away and
// has a second session download from it, so the seeder's read for that
// peer fails. It returns the seeder, its torrent, the file, where the
// file went and the payload.
func seedingThenFailed(t *testing.T, name string) (seeder engine.Engine, h engine.Handle, file, away string, payload []byte) {
	t.Helper()
	payload = make([]byte, 8*retryPiece)
	_, _ = rand.NewChaCha8([32]byte{'r', 'e', 't', 'r', 'y'}).Read(payload)
	meta := singleFileTorrent(t, name, payload, retryPiece)
	seedDir := t.TempDir()
	file, away = filepath.Join(seedDir, name), filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	seeder, leecher := open(t), open(t)
	h = started(t, seeder, engine.AddParams{Metainfo: meta, SavePath: seedDir, SeedMode: true})
	until(t, "the seeder seeds", func() bool {
		s, ok := updated(t, seeder, h)
		return ok && s.State == engine.StateSeeding
	})
	if err := os.Rename(file, away); err != nil {
		t.Fatal(err)
	}
	lh := started(t, leecher, engine.AddParams{Metainfo: meta, SavePath: t.TempDir()})
	addr, err := seeder.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}
	if err := leecher.ConnectPeer(lh, addr); err != nil {
		t.Fatal(err)
	}
	failed := waitFor(t, seeder.Events(), func(f engine.TorrentFailed) bool { return f.Torrent == h })
	if failed.Err == nil || failed.Err.Category != engine.CategorySystem {
		t.Fatalf("the seeder failed its torrent with %v, want a system error reading its file", failed.Err)
	}
	until(t, "the failed torrent's status carries its error", func() bool {
		s, ok := updated(t, seeder, h)
		return ok && s.Err != nil
	})
	return seeder, h, file, away, payload
}

// rechecked rechecks h and returns its status once the check ended.
func rechecked(t *testing.T, e engine.Engine, h engine.Handle) engine.Status {
	t.Helper()
	if err := e.Recheck(h); err != nil {
		t.Fatal(err)
	}
	waitFor(t, e.Events(), func(c engine.Checked) bool { return c.Torrent == h })
	var s engine.Status
	until(t, "the checked torrent's status is posted", func() bool {
		got, ok := updated(t, e, h)
		if ok {
			s = got
		}
		return ok && !checkingState(got.State)
	})
	return s
}

func checkingState(s engine.State) bool {
	return s == engine.StateCheckingFiles || s == engine.StateCheckingResumeData
}

func TestARecheckClearsAReadErrorOnceTheFileIsBack(t *testing.T) {
	seeder, h, file, away, payload := seedingThenFailed(t, "Hold.Fast.mkv")
	if err := os.Rename(away, file); err != nil {
		t.Fatal(err)
	}

	s := rechecked(t, seeder, h)
	if s.Err != nil || s.Done != int64(len(payload)) || s.State != engine.StateSeeding || !s.Paused {
		t.Fatalf("rechecked with its file back, the torrent reads %+v, want no error, every byte done, seeding and paused", s)
	}
	if err := seeder.Resume(h); err != nil {
		t.Fatal(err)
	}
	until(t, "the torrent runs again", func() bool {
		s, ok := updated(t, seeder, h)
		return ok && !s.Paused && s.Err == nil && s.State == engine.StateSeeding
	})
}

func TestARecheckAfterALossFindsWhatIsLeft(t *testing.T) {
	seeder, h, file, away, payload := seedingThenFailed(t, "Lost.Half.mkv")
	kept := 3 * retryPiece
	if err := os.WriteFile(file, payload[:kept], 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(away); err != nil {
		t.Fatal(err)
	}

	s := rechecked(t, seeder, h)
	if s.Err != nil || s.Done != int64(kept) || s.State == engine.StateSeeding || !s.Paused {
		t.Fatalf("rechecked with %d of %d bytes on disk, the torrent reads %+v, want no error, %d bytes done, not seeding and paused", kept, len(payload), s, kept)
	}
}
