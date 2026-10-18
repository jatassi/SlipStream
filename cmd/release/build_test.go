package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/worktree"
)

// TestBuildStampsTheWorktreeIntoDist checks the dist/ a release build of
// this worktree wrote, which CI's build job hands the e2e tier.
func TestBuildStampsTheWorktreeIntoDist(t *testing.T) {
	if _, ok := os.LookupEnv("TEST_SLIPSTREAM_BINARY"); !ok {
		t.Skip("checks a release build: run go run ./cmd/release build, then set TEST_SLIPSTREAM_BINARY and TEST_SLIPSTREAM_E2E_BINARY to dist/slipstream and dist/slipstream-e2e")
	}
	bin, e2e := testbin.Slipstream(t), testbin.E2E(t)
	root, err := repoRoot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want, err := worktree.Stamp(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	want.Target = runtime.GOOS + "-" + runtime.GOARCH
	want.Libtorrent = "2.0.15"
	lock, err := native.LoadLock(filepath.Join(root, native.LockPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range lock.Layers {
		if l.Kind == native.KindLibs && l.Target == want.Target {
			want.NativeBundle = l.Digest
		}
	}

	for _, b := range []string{bin, e2e} {
		got, err := buildinfo.ReadFile(b)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("stamp in %s = %+v, want %+v", b, got, want)
		}
	}

	printed, err := exec.CommandContext(t.Context(), bin, "version").Output()
	if err != nil {
		t.Fatal(err)
	}
	wantPrinted := fmt.Sprintf("slipstream %s\n"+
		"commit         %s\n"+
		"target         %s\n"+
		"libtorrent     %s\n"+
		"fingerprint    %s\n"+
		"native bundle  %s\n",
		want.Version, want.Commit, want.Target, want.Libtorrent, want.Fingerprint, want.NativeBundle)
	if string(printed) != wantPrinted {
		t.Errorf("slipstream version printed\n%s\nwant\n%s", printed, wantPrinted)
	}

	checkTools(t, filepath.Dir(bin))

	index, err := os.ReadFile(filepath.Join(root, "web", "dist", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(binary, index) {
		t.Errorf("%s does not embed web/dist/index.html", bin)
	}
}

func checkTools(t *testing.T, dist string) {
	t.Helper()
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if runtime.GOOS == "windows" {
			tool += ".exe"
		}
		out, err := exec.CommandContext(t.Context(), filepath.Join(dist, tool), "-version").Output()
		if prefix := strings.TrimSuffix(tool, ".exe") + " version 8.1.3-Jellyfin "; err != nil || !strings.HasPrefix(string(out), prefix) {
			t.Errorf("%s -version = %q, %v; want it to start %q", tool, out, err, prefix)
		}
	}
}
