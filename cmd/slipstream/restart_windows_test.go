package main

import (
	"testing"

	"github.com/jatassi/slipstream/internal/node"
)

// TestRestartStartsASuccessorProcess: Windows has no exec, so the Node
// starts a successor that inherits its stderr and exits 0. The successor
// binds the torrent port its predecessor held.
func TestRestartStartsASuccessorProcess(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startNode(t, dir, freeTorrentListen(t))
	first := p.ready()
	before := nodeStatus(t, dir)

	second, after := restarted(t, p, dir)
	if second.PID == first.PID || second.Listeners.Torrent != first.Listeners.Torrent || !after.StartedAt.After(before.StartedAt) {
		t.Fatalf("successor PID %d on torrent %s started %s; want a new PID on torrent %s, started after %s",
			second.PID, second.Listeners.Torrent, after.StartedAt, first.Listeners.Torrent, before.StartedAt)
	}
	stopped(t, p, dir, node.ExitOK)
}
