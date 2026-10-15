package main

import (
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/internal/node"
)

// installedCopy places a copy of the release binary and its tools where
// an install of the host OS's kind puts them, with the marker holding
// marker, and returns the copy's path.
func installedCopy(t *testing.T, marker string) string {
	t.Helper()
	root := dataDir(t)
	binDir, markerDir := root, root
	if runtime.GOOS == "darwin" {
		binDir = filepath.Join(root, "SlipStream.app", "Contents", "MacOS")
		markerDir = filepath.Join(root, "SlipStream.app", "Contents", "Resources")
	}
	for _, d := range []string{binDir, markerDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(binDir, filepath.Base(testbin.Slipstream(t)))
	program, err := os.ReadFile(testbin.Slipstream(t))
	if err != nil {
		t.Fatal(err)
	}
	// A child forked while bin is open for writing would make exec of bin
	// fail with "text file busy".
	syscall.ForkLock.RLock()
	err = os.WriteFile(bin, program, 0o700)
	syscall.ForkLock.RUnlock()
	if err != nil {
		t.Fatal(err)
	}
	testbin.PlaceTools(t, binDir)
	if err := os.WriteFile(filepath.Join(markerDir, "install-kind"), []byte(marker), 0o600); err != nil {
		t.Fatal(err)
	}
	return bin
}

// hostKind is the host OS's per-user install kind, the environment that
// points its default data dir into home, and that data dir.
func hostKind(home string) (kind string, env []string, dir string) {
	switch runtime.GOOS {
	case "windows":
		return "msi", []string{"LOCALAPPDATA=" + home}, filepath.Join(home, "SlipStream2")
	case "darwin":
		return "app", []string{"HOME=" + home}, filepath.Join(home, "Library", "Application Support", "SlipStream2")
	default:
		return "tarball", []string{"XDG_DATA_HOME=" + home}, filepath.Join(home, "slipstream2")
	}
}

func TestTheMarkerPicksTheDefaultDataDir(t *testing.T) {
	t.Parallel()
	home := dataDir(t)
	if runtime.GOOS == "darwin" {
		home = shortDir(t, "/tmp")
	}
	kind, env, want := hostKind(home)
	p := startBinary(t, installedCopy(t, kind+"\n"), "", env...)
	if got := p.ready().DataDir; got != want {
		t.Fatalf("data dir %s, want %s", got, want)
	}
	if _, err := os.Stat(filepath.Join(want, "logs", "slipstream.log")); err != nil {
		t.Fatalf("log in the default data dir: %v", err)
	}
	res := slipstream(t, childEnv(env...), "stop", "--data-dir", want)
	if res.code != node.ExitOK {
		t.Fatalf("stop: exit %d, stderr %q", res.code, res.stderr)
	}
}

func TestABadMarkerExitsWithTheUsageCodeNamingIt(t *testing.T) {
	t.Parallel()
	bin := installedCopy(t, "snap\n")
	marker := filepath.Join(filepath.Dir(bin), "install-kind")
	if runtime.GOOS == "darwin" {
		marker = filepath.Join(filepath.Dir(filepath.Dir(bin)), "Resources", "install-kind")
	}
	cmd := exec.CommandContext(t.Context(), bin, "run", "--data-dir", dataDir(t))
	cmd.Env = childEnv()
	out, _ := cmd.CombinedOutput()
	want := "slipstream: run: install marker " + marker + ` holds "snap\n", not an install kind` + "\n"
	if code := cmd.ProcessState.ExitCode(); code != node.ExitUsage || !strings.HasPrefix(string(out), want) {
		t.Fatalf("bad marker: exit %d, output %q; want exit %d starting %q", code, out, node.ExitUsage, want)
	}
}

func TestStatusAnswersOnceReady(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startNode(t, dir)
	ready := p.ready()
	res := slipstream(t, childEnv(), "status", "--data-dir", dir)
	if res.code != node.ExitOK {
		t.Fatalf("status: exit %d, stderr %q", res.code, res.stderr)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(res.stdout), &got); err != nil {
		t.Fatalf("status output %q: %v", res.stdout, err)
	}
	url := "http://localhost:" + strconv.Itoa(int(ready.Listeners.HTTP.Port())) + "/setup#code=" + ready.SetupCode
	want := map[string]any{"pid": float64(ready.PID), "version": "dev", "install": "dev", "mode": "run", "claim": "unclaimed", "url": url}
	if !maps.Equal(got, want) {
		t.Fatalf("status = %v, want %v", got, want)
	}
	if rec := p.record("setup page"); rec["url"] != url {
		t.Fatalf("setup page record = %v, want url %s", rec, url)
	}
}

// shortDir is a temp dir under parent, for a default data dir whose
// control socket must stay under the 103-byte limit: macOS's own temp dir
// is too long for Library/Application Support/SlipStream2 below it.
func shortDir(t *testing.T, parent string) string {
	t.Helper()
	dir, err := os.MkdirTemp(parent, "ss") //nolint:usetesting // t.TempDir's path is too long for the socket limit on macOS
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}
