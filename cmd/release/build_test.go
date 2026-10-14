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
	"github.com/jatassi/slipstream/cmd/release/internal/worktree"
	"github.com/jatassi/slipstream/internal/buildinfo"
)

func TestBuildStampsTheWorktreeIntoDist(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the web app and the release binary")
	}
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

	dist := t.TempDir()
	if err := runBuild(t.Context(), []string{"-o", dist}); err != nil {
		t.Fatal(err)
	}

	bin, e2e := filepath.Join(dist, "slipstream"), filepath.Join(dist, "slipstream-e2e")
	if runtime.GOOS == "windows" {
		bin, e2e = bin+".exe", e2e+".exe"
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

	checkTools(t, dist)

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
