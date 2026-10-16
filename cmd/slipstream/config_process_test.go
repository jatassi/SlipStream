package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/node"
)

func configYAML(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestConfigSetWritesAStoppedNodesFile(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	for _, c := range []struct {
		env    []string
		args   []string
		code   int
		stdout string
		stderr string
		file   string
	}{
		{args: []string{"set", "log.level", "debug"}, stdout: "log.level: debug\n", file: "log:\n  level: debug\n"},
		{args: []string{"set", "log.max_files", "101"}, code: node.ExitUsage, stderr: "slipstream config set: log.max_files: Between 0 and 100 files.\n", file: "log:\n  level: debug\n"},
		{env: []string{"SLIPSTREAM_LOG_LEVEL=warn"}, args: []string{"unset", "log.level"}, code: node.ExitUsage, stderr: "slipstream config unset: log.level: Set by SLIPSTREAM_LOG_LEVEL.\n", file: "log:\n  level: debug\n"},
		{args: []string{"unset", "log.level"}, stdout: "log.level: info\n", file: ""},
	} {
		args := append([]string{"config", c.args[0], "--data-dir", dir}, c.args[1:]...)
		res := slipstream(t, childEnv(c.env...), args...)
		if res.code != c.code || res.stdout != c.stdout || res.stderr != c.stderr {
			t.Fatalf("%v: exit %d, stdout %q, stderr %q; want %d, %q, %q", c.args, res.code, res.stdout, res.stderr, c.code, c.stdout, c.stderr)
		}
		if got := configYAML(t, dir); got != c.file {
			t.Fatalf("%v: config.yaml = %q, want %q", c.args, got, c.file)
		}
	}

	res := slipstream(t, childEnv(), "config", "set", "--data-dir", dir, "log.level")
	if res.code != node.ExitUsage || res.stdout != "" {
		t.Fatalf("set without a value: exit %d, stdout %q", res.code, res.stdout)
	}
}

func TestConfigSetGoesThroughTheRunningNode(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startNode(t, dir)
	p.ready()

	res := slipstream(t, childEnv(), "config", "set", "--data-dir", dir, "log.max_size_mb", "20")
	if want := "log.max_size_mb: 20, applies after restart\n"; res.code != node.ExitOK || res.stdout != want {
		t.Fatalf("exit %d, stdout %q, stderr %q; want %q", res.code, res.stdout, res.stderr, want)
	}
	if got, want := configYAML(t, dir), "log:\n  max_size_mb: 20\n"; got != want {
		t.Fatalf("config.yaml = %q, want %q", got, want)
	}

	res = slipstream(t, childEnv(), "config", "set", "--data-dir", dir, "http.listen", "[::]:9000")
	if want := "slipstream config set: http.listen: Set by SLIPSTREAM_HTTP_LISTEN.\n"; res.code != node.ExitUsage || res.stderr != want {
		t.Fatalf("exit %d, stderr %q; want %d, %q", res.code, res.stderr, node.ExitUsage, want)
	}
}
