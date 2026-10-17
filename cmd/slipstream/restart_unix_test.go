//go:build unix

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/internal/node"
)

// TestRestartExecsInPlace holds the torrent client to one port, so a
// socket the exec leaked would keep the successor from binding it.
func TestRestartExecsInPlace(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startNode(t, dir, freeTorrentListen(t))
	first := p.ready()
	before := nodeStatus(t, dir)

	second, after := restarted(t, p, dir)
	if second.PID != first.PID || second.Listeners.Torrent != first.Listeners.Torrent || !after.StartedAt.After(before.StartedAt) {
		t.Fatalf("successor PID %d on torrent %s started %s; want PID %d on torrent %s, started after %s",
			second.PID, second.Listeners.Torrent, after.StartedAt, first.PID, first.Listeners.Torrent, before.StartedAt)
	}
	if runtime.GOOS == "linux" {
		held := openFiles(t, first.PID)
		for _, file := range []string{filepath.Join(dir, "logs", "slipstream.log"), filepath.Join(dir, "run", "node.lock")} {
			if n := strings.Count(held, file+"\n"); n != 1 {
				t.Errorf("the successor holds %s open %d times, want once:\n%s", file, n, held)
			}
		}
	}
	stopped(t, p, dir, node.ExitOK)
}

// openFiles is every file pid holds open, one per line.
func openFiles(t *testing.T, pid int) string {
	t.Helper()
	fds := "/proc/" + strconv.Itoa(pid) + "/fd"
	entries, err := os.ReadDir(fds)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, e := range entries {
		if target, err := os.Readlink(filepath.Join(fds, e.Name())); err == nil {
			b.WriteString(target + "\n")
		}
	}
	return b.String()
}

func TestARestartPastItsStopDeadlineExitsWithNoSuccessor(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startNode(t, dir, "SLIPSTREAM_TIMINGS_NODE_STOP_DEADLINE=1ns")
	p.ready()
	ctx, cancel := context.WithCancel(t.Context())
	cli := exec.CommandContext(ctx, testbin.Slipstream(t), "restart", "--data-dir", dir)
	cli.Env = childEnv()
	if err := cli.Start(); err != nil {
		t.Fatal(err)
	}
	code := p.wait()
	cancel()
	_ = cli.Wait()
	if code != node.ExitFailed {
		t.Fatalf("exit %d, want %d:\n%s", code, node.ExitFailed, p.output())
	}
	p.record("stop deadline passed")
	if n := strings.Count(p.output(), `"msg":"ready"`); n != 1 {
		t.Fatalf("%d readiness records, want only the first:\n%s", n, p.output())
	}
}
