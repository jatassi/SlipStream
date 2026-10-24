package libtorrent_test

import (
	"errors"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// fullVolume names the small tmpfs the test runs its download into once it
// re-ran itself in a mount namespace of its own.
const fullVolume = "SLIPSTREAM_TEST_FULL_VOLUME"

// TestADownloadThatFillsTheDiskPausesUntilResumed downloads 1 MiB into a
// 256 KiB volume: the torrent sends FileError matching ErrDiskFull and
// pauses, and once the volume grows, Resume finishes it.
func TestADownloadThatFillsTheDiskPausesUntilResumed(t *testing.T) {
	volume := os.Getenv(fullVolume)
	if volume == "" {
		inMountNamespace(t)
		return
	}
	payload := make([]byte, 1<<20)
	_, _ = rand.NewChaCha8([32]byte{'f', 'u', 'l', 'l'}).Read(payload)
	meta := singleFileTorrent(t, "payload.bin", payload, 64<<10)
	seedDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(seedDir, "payload.bin"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	seeder, leecher := open(t), open(t)
	started(t, seeder, engine.AddParams{Metainfo: meta, SavePath: seedDir, SeedMode: true})
	seederAddr, err := seeder.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}
	h := started(t, leecher, engine.AddParams{Metainfo: meta, SavePath: volume})
	if err := leecher.ConnectPeer(h, seederAddr); err != nil {
		t.Fatal(err)
	}

	failed := waitFor(t, leecher.Events(), func(f engine.FileError) bool { return f.Torrent == h })
	if !errors.Is(failed.Err, engine.ErrDiskFull) {
		t.Fatalf("writing to a full volume the torrent failed with %v, want %v", failed.Err, engine.ErrDiskFull)
	}
	waitFor(t, leecher.Events(), func(p engine.Paused) bool { return p.Torrent == h })

	if out, err := exec.CommandContext(t.Context(), "mount", "-o", "remount,size=4m", volume).CombinedOutput(); err != nil {
		t.Fatalf("grow the volume: %v: %s", err, out)
	}
	if err := leecher.Resume(h); err != nil {
		t.Fatal(err)
	}
	if err := leecher.ConnectPeer(h, seederAddr); err != nil {
		t.Fatal(err)
	}
	waitFor(t, leecher.Events(), func(f engine.Finished) bool { return f.Torrent == h })
}

// inMountNamespace runs the calling test again as root of a user and mount
// namespace of its own, with a 256 KiB tmpfs mounted for it.
func inMountNamespace(t *testing.T) {
	t.Helper()
	volume := t.TempDir()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "unshare", "--user", "--map-root-user", "--mount", "sh", "-c",
		`mount -t tmpfs -o size=256k tmpfs "$0" || exit 99; exec "$@"`,
		volume, self, "-test.run=^"+t.Name()+"$", "-test.v", "-test.timeout=5m")
	cmd.Env = append(os.Environ(), fullVolume+"="+volume)
	start := time.Now()
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) && (exit.ExitCode() == 99 || strings.Contains(string(out), "unshare: ")) {
		t.Skipf("this host refuses a mount namespace with a tmpfs to an unprivileged user: %s", out)
	}
	if err != nil || !strings.Contains(string(out), "--- PASS: "+t.Name()) {
		t.Fatalf("in its own mount namespace the test ran for %s: %v\n%s", time.Since(start), err, out)
	}
}
