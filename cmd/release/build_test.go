package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/worktree"
	"github.com/jatassi/slipstream/internal/buildinfo"
)

func TestBuildStampsTheWorktreeIntoDist(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the web app and the release binary")
	}
	t.Setenv("CGO_ENABLED", "0")
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	want, err := worktree.Stamp(root)
	if err != nil {
		t.Fatal(err)
	}
	want.Target = runtime.GOOS + "-" + runtime.GOARCH
	want.NativeBundle = "test-bundle"
	want.Libtorrent = "0.0.0-test"

	if err := runBuild([]string{"-native-bundle", "test-bundle", "-libtorrent", "0.0.0-test"}); err != nil {
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
		"libtorrent     0.0.0-test\n"+
		"fingerprint    %s\n"+
		"native bundle  test-bundle\n",
		want.Version, want.Commit, want.Target, want.Fingerprint)
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
