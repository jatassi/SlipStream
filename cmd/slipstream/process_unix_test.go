//go:build unix

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/jatassi/slipstream/internal/node"
)

func TestSIGTERMStopsTheNodeCleanly(t *testing.T) {
	t.Parallel()
	p := startNode(t, dataDir(t), listenLocal)
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
	p := startNode(t, dir, listenLocal)
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
