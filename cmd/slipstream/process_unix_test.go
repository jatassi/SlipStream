//go:build unix

package main

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/jatassi/slipstream/internal/node"
)

func TestSIGTERMStopsTheNodeCleanly(t *testing.T) {
	t.Parallel()
	p := startNode(t, dataDir(t))
	p.ready()
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if code := p.wait(); code != node.ExitOK {
		t.Fatalf("exit %d after SIGTERM, want %d:\n%s", code, node.ExitOK, p.output())
	}
	if last := p.lastRecord(); last != "stopped" {
		t.Fatalf("last record = %q, want stopped:\n%s", last, p.output())
	}
}

func TestDataDirIsPrivateToItsUser(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(dataDir(t), "node")
	p := startNode(t, dir)
	p.ready()
	for _, path := range []string{dir, filepath.Join(dir, "run"), filepath.Join(dir, "logs")} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if mode := info.Mode().Perm(); mode != 0o700 {
			t.Errorf("%s has mode %o, want 700", path, mode)
		}
	}
}

func TestADataDirAnotherUserOwnsExitsWithTheUsageCode(t *testing.T) {
	t.Parallel()
	if os.Getuid() != 0 {
		t.Skip("only root can give a data dir to another user")
	}
	dir := dataDir(t)
	if err := os.Chown(dir, 65534, 65534); err != nil {
		t.Fatal(err)
	}
	owner, err := user.LookupId("65534")
	if err != nil {
		t.Fatal(err)
	}
	res := slipstream(t, childEnv(), "config", "--data-dir", dir)
	want := "slipstream: config: data dir " + dir + " belongs to " + owner.Username + "; run: sudo -u " + owner.Username + " slipstream config ...\n"
	if res.code != node.ExitUsage || !strings.HasPrefix(res.stderr, want) {
		t.Fatalf("foreign data dir: exit %d, stderr %q; want exit %d starting %q", res.code, res.stderr, node.ExitUsage, want)
	}
}
