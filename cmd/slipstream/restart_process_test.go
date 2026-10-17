package main

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/node"
)

func nodeStatus(t *testing.T, dir string) node.StatusReply {
	t.Helper()
	res := slipstream(t, childEnv(), "status", "--data-dir", dir)
	var s node.StatusReply
	if err := json.Unmarshal([]byte(res.stdout), &s); err != nil || res.code != node.ExitOK {
		t.Fatalf("status: exit %d, stdout %q, stderr %q", res.code, res.stdout, res.stderr)
	}
	return s
}

// startedSince checks status's started_at falls between launched and now,
// and removes it, so the rest compares as literals.
func startedSince(t *testing.T, status map[string]any, launched time.Time) {
	t.Helper()
	raw, _ := status["started_at"].(string)
	started, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil || started.Before(launched) || started.After(time.Now()) {
		t.Fatalf("started_at = %q, want a time since %s", raw, launched)
	}
	delete(status, "started_at")
}

// freeTorrentListen is a loopback address no listener holds now.
func freeTorrentListen(t *testing.T) string {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return "SLIPSTREAM_TORRENT_LISTEN=" + addr
}

// restarted runs slipstream restart on dir and returns the Node's next
// readiness record and status.
func restarted(t *testing.T, p *nodeProc, dir string) (node.Ready, node.StatusReply) {
	t.Helper()
	res := slipstream(t, childEnv(), "restart", "--data-dir", dir)
	if res.code != node.ExitOK {
		t.Fatalf("restart: exit %d, stderr %q:\n%s", res.code, res.stderr, p.output())
	}
	ready := p.ready()
	status := nodeStatus(t, dir)
	if res.stdout != status.URL+"\n" {
		t.Fatalf("restart printed %q, want the successor's URL %s", res.stdout, status.URL)
	}
	return ready, status
}

func stopped(t *testing.T, p *nodeProc, dir string, wantExit int) {
	t.Helper()
	if res := slipstream(t, childEnv(), "stop", "--data-dir", dir); res.code != node.ExitOK {
		t.Fatalf("stop: exit %d, stderr %q", res.code, res.stderr)
	}
	if code := p.wait(); code != wantExit {
		t.Fatalf("exit %d, want %d:\n%s", code, wantExit, p.output())
	}
}
