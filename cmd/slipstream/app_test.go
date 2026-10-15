//go:build linux || windows

package main

import (
	"encoding/json"
	"testing"

	"github.com/jatassi/slipstream/internal/node"
)

// statusOf is slipstream status on dir, decoded.
func statusOf(t *testing.T, dir string) map[string]any {
	t.Helper()
	res := slipstream(t, childEnv(), "status", "--data-dir", dir)
	if res.code != node.ExitOK {
		t.Fatalf("status: exit %d, stderr %q", res.code, res.stderr)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(res.stdout), &got); err != nil {
		t.Fatalf("status output %q: %v", res.stdout, err)
	}
	return got
}
