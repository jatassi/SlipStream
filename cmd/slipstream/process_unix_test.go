//go:build unix

package main

import (
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
