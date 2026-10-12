package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/cmd/release/internal/worktree"
	"github.com/jatassi/slipstream/internal/buildinfo"
)

func TestBuildStampsTheWorktreeIntoDist(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the web app and the release binary")
	}
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	want, err := worktree.Stamp(root)
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

	if err := runBuild(nil); err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(root, "dist", "slipstream")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	got, err := buildinfo.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("stamp in %s = %+v, want %+v", bin, got, want)
	}

	printed, err := exec.Command(bin, "version").Output()
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
