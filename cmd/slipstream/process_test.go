package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/node"
)

const listenLocal = "SLIPSTREAM_TORRENT_LISTEN=127.0.0.1:0"

var binary string

func TestMain(m *testing.M) {
	os.Exit(buildThenRun(m))
}

func buildThenRun(m *testing.M) int {
	dir, err := os.MkdirTemp("", "slipstream-bin")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()
	binary = filepath.Join(dir, "slipstream")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.CommandContext(context.Background(), "go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build slipstream: %v\n%s", err, out)
		return 1
	}
	return m.Run()
}

// childEnv is explicit, so a stray SLIPSTREAM_ variable in the developer's
// environment never reaches the strict loader. Windows needs SystemRoot to
// initialize sockets.
func childEnv(extra ...string) []string {
	var env []string
	for _, name := range []string{"PATH", "SystemRoot", "TEMP", "TMP", "TMPDIR", "HOME"} {
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}
	return append(env, extra...)
}

// dataDir is short, so its control socket path stays under the 103-byte
// limit inside macOS's long temp dir.
func dataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ss") //nolint:usetesting // t.TempDir's path is too long for the socket limit on macOS
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	return dir
}

type nodeProc struct {
	t     *testing.T
	cmd   *exec.Cmd
	lines chan string

	mu     sync.Mutex
	stderr []string
	exited chan struct{}
}

func startNode(t *testing.T, dir string, env ...string) *nodeProc {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), binary, "run", "--data-dir", dir)
	cmd.Env = childEnv(env...)
	pipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	p := &nodeProc{t: t, cmd: cmd, lines: make(chan string, 1024), exited: make(chan struct{})}
	go func() {
		sc := bufio.NewScanner(pipe)
		for sc.Scan() {
			p.mu.Lock()
			p.stderr = append(p.stderr, sc.Text())
			p.mu.Unlock()
			p.lines <- sc.Text()
		}
		close(p.lines)
		_ = cmd.Wait()
		close(p.exited)
	}()
	t.Cleanup(func() {
		select {
		case <-p.exited:
		default:
			_ = cmd.Process.Kill()
			<-p.exited
		}
	})
	return p
}

func (p *nodeProc) ready() node.Ready {
	p.t.Helper()
	timeout := time.After(60 * time.Second)
	for {
		select {
		case line, ok := <-p.lines:
			if !ok {
				p.t.Fatalf("node exited before ready:\n%s", p.output())
			}
			r, err := node.ParseReady([]byte(line))
			if errors.Is(err, node.ErrNotReadyRecord) {
				continue
			}
			if err != nil {
				p.t.Fatalf("stderr line %q: %v", line, err)
			}
			return r
		case <-timeout:
			p.t.Fatalf("no readiness record after 60s:\n%s", p.output())
		}
	}
}

func (p *nodeProc) output() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return strings.Join(p.stderr, "\n")
}

type result struct {
	code           int
	stdout, stderr string
}

func slipstream(t *testing.T, env []string, args ...string) result {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	var out, errOut strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	if _, exit := errors.AsType[*exec.ExitError](err); err != nil && !exit {
		t.Fatalf("slipstream %v: %v", args, err)
	}
	return result{cmd.ProcessState.ExitCode(), out.String(), errOut.String()}
}

func TestTwoNodesShareNothingAndRefuseAThird(t *testing.T) {
	t.Parallel()
	dirA, dirB := dataDir(t), dataDir(t)
	a, b := startNode(t, dirA, listenLocal), startNode(t, dirB, listenLocal)
	readyA, readyB := a.ready(), b.ready()

	for _, c := range []struct {
		dir   string
		ready node.Ready
	}{{dirA, readyA}, {dirB, readyB}} {
		if c.ready.DataDir != c.dir {
			t.Errorf("data dir = %s, want %s", c.ready.DataDir, c.dir)
		}
		if c.ready.Listeners.Torrent.Port() == 0 {
			t.Errorf("node on %s reports torrent port 0", c.dir)
		}
		want := []node.Key{{Key: "torrent.listen", Value: "127.0.0.1:0", Default: "0.0.0.0:6881", Source: "env"}}
		if !slices.Equal(c.ready.Overrides, want) {
			t.Errorf("overrides = %+v, want %+v", c.ready.Overrides, want)
		}
	}
	if readyA.Listeners.Torrent == readyB.Listeners.Torrent {
		t.Errorf("both nodes report torrent address %s", readyA.Listeners.Torrent)
	}

	res := slipstream(t, childEnv(listenLocal), "run", "--data-dir", dirA)
	if want := fmt.Sprintf("data dir %s is in use by SlipStream PID %d\n", dirA, readyA.PID); res.code != node.ExitInUse || res.stderr != want {
		t.Fatalf("third node: exit %d, stderr %q; want exit %d, stderr %q", res.code, res.stderr, node.ExitInUse, want)
	}
}

func TestBadConfigurationExitsWithTheUsageCode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		files map[string]string
		env   []string
		want  string
	}{
		{"misspelled yaml key", map[string]string{"config.yaml": "torrent:\n  listn: 127.0.0.1:0\n"}, nil, "config.yaml:2: torrent.listn: unknown key"},
		{"unknown variable", nil, []string{"SLIPSTREAM_TORRENT_LISTN=127.0.0.1:0"}, "env: SLIPSTREAM_TORRENT_LISTN: unknown key"},
		{".env line without prefix", map[string]string{".env": "TORRENT_LISTEN=127.0.0.1:0\n"}, nil, ".env:1: TORRENT_LISTEN: missing the SLIPSTREAM_ prefix"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			dir := dataDir(t)
			for name, body := range c.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			res := slipstream(t, childEnv(c.env...), "run", "--data-dir", dir)
			if res.code != node.ExitUsage || !strings.Contains(res.stderr, c.want) {
				t.Fatalf("exit %d, stderr:\n%s\nwant exit %d naming %q", res.code, res.stderr, node.ExitUsage, c.want)
			}
			logged, err := os.ReadFile(filepath.Join(dir, "logs", "slipstream.log"))
			if err != nil || !strings.Contains(string(logged), c.want) {
				t.Fatalf("log file does not name %q: %v\n%s", c.want, err, logged)
			}
		})
	}
}

func TestDataDirErrorsExitWithTheUsageCode(t *testing.T) {
	t.Parallel()
	res := slipstream(t, childEnv(), "run")
	if want := "slipstream: run: no data dir: pass --data-dir or set SLIPSTREAM_DATA_DIR\n"; res.code != node.ExitUsage || !strings.HasPrefix(res.stderr, want) {
		t.Fatalf("no data dir: exit %d, stderr %q; want exit %d starting %q", res.code, res.stderr, node.ExitUsage, want)
	}

	long := filepath.Join(dataDir(t), strings.Repeat("d", 100))
	res = slipstream(t, childEnv(), "run", "--data-dir", long)
	socket := filepath.Join(long, "run", "control.sock")
	want := fmt.Sprintf("control socket path %s is %d bytes, over the 103-byte limit", socket, len(socket))
	if res.code != node.ExitUsage || !strings.Contains(res.stderr, want) {
		t.Fatalf("long data dir: exit %d, stderr %q; want exit %d naming %q", res.code, res.stderr, node.ExitUsage, want)
	}
	if _, err := os.Stat(long); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("refused data dir was created: %v", err)
	}
}

func TestSeedIsReportedAndReplayable(t *testing.T) {
	t.Parallel()
	var seeds []string
	for _, env := range [][]string{{"SLIPSTREAM_SEED=fixed"}, {"SLIPSTREAM_SEED=fixed"}, nil} {
		p := startNode(t, dataDir(t), append([]string{listenLocal}, env...)...)
		seeds = append(seeds, p.ready().Seed)
	}
	if seeds[0] != "fixed" || seeds[1] != "fixed" {
		t.Fatalf("seeds with SLIPSTREAM_SEED=fixed = %q, %q", seeds[0], seeds[1])
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(seeds[2]) {
		t.Fatalf("drawn seed %q is not 32 hex characters", seeds[2])
	}
}
