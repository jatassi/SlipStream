package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/example/shelf"
)

// localListeners keep parallel Nodes off each other's ports. startNode
// always sets them.
var localListeners = []string{"SLIPSTREAM_TORRENT_LISTEN=127.0.0.1:0", "SLIPSTREAM_HTTP_LISTEN=127.0.0.1:0", "SLIPSTREAM_LINK_LISTEN=127.0.0.1:0"}

func TestMain(m *testing.M) {
	os.Exit(testbin.Run(m))
}

// childEnv is explicit, so a stray SLIPSTREAM_ variable in the developer's
// environment never reaches the strict loader. Windows needs SystemRoot to
// initialize sockets, and PowerShell needs PATHEXT to run an .exe and wait
// for its exit code: without it PowerShell opens the file through
// ShellExecute, does not wait, and leaves $LASTEXITCODE unset.
func childEnv(extra ...string) []string {
	var env []string
	for _, name := range []string{"PATH", "PATHEXT", "SystemRoot", "TEMP", "TMP", "TMPDIR", "HOME"} {
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
	return startBinary(t, testbin.Slipstream(t), dir, env...)
}

func startBinary(t *testing.T, bin, dir string, env ...string) *nodeProc {
	t.Helper()
	args := []string{"run"}
	if dir != "" {
		args = append(args, "--data-dir", dir)
	}
	return startArgs(t, bin, args, env...)
}

func startArgs(t *testing.T, bin string, args []string, env ...string) *nodeProc {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), bin, args...)
	cmd.Env = childEnv(append(slices.Clone(localListeners), env...)...)
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

func (p *nodeProc) wait() int {
	p.t.Helper()
	select {
	case <-p.exited:
		return p.cmd.ProcessState.ExitCode()
	case <-time.After(60 * time.Second):
		p.t.Fatalf("node still running after 60s:\n%s", p.output())
		return -1
	}
}

func (p *nodeProc) record(msg string) map[string]any {
	p.t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, line := range p.stderr {
		var rec map[string]any
		if json.Unmarshal([]byte(line), &rec) == nil && rec["msg"] == msg {
			return rec
		}
	}
	p.t.Fatalf("no %q record in:\n%s", msg, strings.Join(p.stderr, "\n"))
	return nil
}

func (p *nodeProc) lastRecord() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	var rec struct {
		Msg string `json:"msg"`
	}
	if len(p.stderr) == 0 || json.Unmarshal([]byte(p.stderr[len(p.stderr)-1]), &rec) != nil {
		return ""
	}
	return rec.Msg
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
	cmd := exec.CommandContext(ctx, testbin.Slipstream(t), args...)
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
	a, b := startNode(t, dirA), startNode(t, dirB)
	readyA, readyB := a.ready(), b.ready()

	for _, c := range []struct {
		dir   string
		ready node.Ready
	}{{dirA, readyA}, {dirB, readyB}} {
		if c.ready.DataDir != c.dir {
			t.Errorf("data dir = %s, want %s", c.ready.DataDir, c.dir)
		}
		if want := filepath.Join(c.dir, "run", "control.sock"); c.ready.Listeners.Control != want {
			t.Errorf("control socket = %s, want %s", c.ready.Listeners.Control, want)
		}
		if c.ready.Listeners.Torrent.Port() == 0 {
			t.Errorf("node on %s reports torrent port 0", c.dir)
		}
		want := []node.Key{
			{Key: "torrent.listen", Value: "127.0.0.1:0", Default: "0.0.0.0:6881", Source: "env", Env: "SLIPSTREAM_TORRENT_LISTEN"},
			{Key: "http.listen", Value: "127.0.0.1:0", Default: "[::]:8484", Source: "env", Env: "SLIPSTREAM_HTTP_LISTEN"},
			{Key: "link.listen", Value: "127.0.0.1:0", Default: "[::]:8485", Source: "env", Env: "SLIPSTREAM_LINK_LISTEN"},
		}
		if !slices.Equal(c.ready.Overrides, want) {
			t.Errorf("overrides = %+v, want %+v", c.ready.Overrides, want)
		}
	}
	if readyA.Listeners.Torrent == readyB.Listeners.Torrent {
		t.Errorf("both nodes report torrent address %s", readyA.Listeners.Torrent)
	}

	res := slipstream(t, childEnv(localListeners...), "run", "--data-dir", dirA)
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

func TestRefusedLibraryExitsWithItsOwnCode(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	lib, err := storage.OpenLibrary(t.Context(), filepath.Join(dir, storage.LibraryFile), []storage.Schema[storage.Library]{shelf.Schema()}, nil, storage.Timings{BusyTimeout: time.Second}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.Close(); err != nil {
		t.Fatal(err)
	}
	res := slipstream(t, childEnv(localListeners...), "run", "--data-dir", dir)
	want := "library.db holds schema shelf, which this binary does not have"
	if got := startFailed(res.stderr); res.code != node.ExitLibraryRefused || got != want {
		t.Fatalf("exit %d, start failed with %q; want exit %d with %q\n%s", res.code, got, node.ExitLibraryRefused, want, res.stderr)
	}
}

func TestALibraryTheE2EBinaryClaimedRefusesTheReleaseBinary(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startBinary(t, testbin.E2E(t), dir)
	ready := p.ready()
	if claimed := claimController(t, ready, ready.SetupCode); claimed.status != http.StatusCreated {
		t.Fatalf("claimController on the e2e binary = %d %s", claimed.status, claimed.body)
	}
	stopNode(t, p, dir)
	res := slipstream(t, childEnv(localListeners...), "run", "--data-dir", dir)
	want := "library.db holds schema music, which this binary does not have"
	if got := startFailed(res.stderr); res.code != node.ExitLibraryRefused || got != want {
		t.Fatalf("exit %d, start failed with %q; want exit %d with %q\n%s", res.code, got, node.ExitLibraryRefused, want, res.stderr)
	}
}

// startFailed is the err of the start failed record on stderr.
func startFailed(stderr string) string {
	var failed struct{ Msg, Err string }
	for line := range strings.SplitSeq(strings.TrimSpace(stderr), "\n") {
		if err := json.Unmarshal([]byte(line), &failed); err == nil && failed.Msg == "start failed" {
			return failed.Err
		}
	}
	return ""
}

func TestAMalformedNodeKeyExitsWithItsOwnCode(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	key := filepath.Join(dir, "node.key")
	if err := os.WriteFile(key, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := slipstream(t, childEnv(localListeners...), "run", "--data-dir", dir)
	want := "node key " + key + " is not 64 hex characters"
	if got := startFailed(res.stderr); res.code != node.ExitRecordRefused || got != want {
		t.Fatalf("exit %d, start failed with %q; want exit %d with %q\n%s", res.code, got, node.ExitRecordRefused, want, res.stderr)
	}
}

// A node.key the Node cannot read, here a directory on every OS, exits 6
// at start and in an offline join.
func TestAnUnreadableNodeKeyExitsWithTheRecordCode(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	if err := os.Mkdir(filepath.Join(dir, "node.key"), 0o700); err != nil {
		t.Fatal(err)
	}
	res := slipstream(t, childEnv(localListeners...), "run", "--data-dir", dir)
	if got := startFailed(res.stderr); res.code != node.ExitRecordRefused || !strings.HasPrefix(got, "read node key: ") {
		t.Fatalf("run: exit %d, start failed with %q; want exit %d with read node key: ...\n%s", res.code, got, node.ExitRecordRefused, res.stderr)
	}
	res = slipstream(t, childEnv(), "join", "--data-dir", dir, joinCode)
	if res.code != node.ExitRecordRefused || !strings.HasPrefix(res.stderr, "slipstream join: read node key: ") {
		t.Fatalf("join: exit %d, stderr %q; want exit %d with read node key: ...", res.code, res.stderr, node.ExitRecordRefused)
	}
}

// joinCode is the SLP1 golden code; no output may hold its text.
const joinCode = "SLP1-CC6WT-AB6RG-SPD48-JAH4B-QCJV9-XJ6AS-K7D1M-PMTVC-DNQ6Y-W3HE9-SG213-Z0000-58D1S-QG852-G"

func holdsTheCode(out string) bool { return holdsCode(out, joinCode) }

// holdsCode is whether out holds code's text past its SLP1 prefix, in any
// case.
func holdsCode(out, code string) bool {
	return strings.Contains(strings.ToUpper(out), strings.ToUpper(code[len("SLP1-"):len("SLP1-")+11]))
}

func TestAJoinCodeNeverReachesALogOrTheConfigOutput(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	damaged := strings.ToLower(joinCode[:len(joinCode)-1] + "H")
	res := slipstream(t, childEnv(append(slices.Clone(localListeners), "SLIPSTREAM_JOIN_CODE="+damaged)...), "run", "--data-dir", dir)
	want := "env: SLIPSTREAM_JOIN_CODE: the Join Code is mistyped or cut short"
	if res.code != node.ExitUsage || startFailed(res.stderr) != want {
		t.Fatalf("exit %d, start failed with %q; want exit %d with %q\n%s", res.code, startFailed(res.stderr), node.ExitUsage, want, res.stderr)
	}
	logged, err := os.ReadFile(filepath.Join(dir, "logs", "slipstream.log"))
	if err != nil {
		t.Fatal(err)
	}
	if holdsTheCode(res.stderr) || holdsTheCode(string(logged)) {
		t.Fatalf("a log holds the Join Code:\n%s\n%s", res.stderr, logged)
	}

	res = slipstream(t, childEnv("SLIPSTREAM_JOIN_CODE="+damaged), "config", "--data-dir", dir)
	if res.code != node.ExitUsage || res.stderr != "slipstream config: "+want+"\n" {
		t.Fatalf("config with a damaged code: exit %d, stderr %q; want exit %d, %q", res.code, res.stderr, node.ExitUsage, want)
	}
	res = slipstream(t, childEnv("SLIPSTREAM_JOIN_CODE="+joinCode), "config", "--data-dir", dir)
	var got node.Resolved
	if err := json.Unmarshal([]byte(res.stdout), &got); err != nil || res.code != node.ExitOK {
		t.Fatalf("config: exit %d, stdout %q, stderr %q", res.code, res.stdout, res.stderr)
	}
	wantKey := node.Key{Key: "join.code", Value: "SLP1-[redacted]", Default: "", Source: "env", Env: "SLIPSTREAM_JOIN_CODE"}
	if !slices.Contains(got.Keys, wantKey) || holdsTheCode(res.stdout) || holdsTheCode(res.stderr) {
		t.Fatalf("config = %s, want %+v and no code", res.stdout, wantKey)
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
		p := startNode(t, dataDir(t), env...)
		seeds = append(seeds, p.ready().Seed)
	}
	if seeds[0] != "fixed" || seeds[1] != "fixed" {
		t.Fatalf("seeds with SLIPSTREAM_SEED=fixed = %q, %q", seeds[0], seeds[1])
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(seeds[2]) {
		t.Fatalf("drawn seed %q is not 32 hex characters", seeds[2])
	}
}

func TestTimingsOverrideIsReportedAndWarned(t *testing.T) {
	t.Parallel()
	p := startNode(t, dataDir(t), "SLIPSTREAM_TIMINGS_NODE_STOP_DEADLINE=5s")
	ready := p.ready()
	want := node.Key{Key: "timings.node.stop_deadline", Value: "5s", Default: "30s", Source: "env", Env: "SLIPSTREAM_TIMINGS_NODE_STOP_DEADLINE"}
	if !slices.Contains(ready.Overrides, want) {
		t.Fatalf("overrides = %+v, want %+v among them", ready.Overrides, want)
	}
	warning := p.record("timings differ from their defaults")
	got, err := json.Marshal([]any{warning["level"], warning["keys"]})
	if err != nil {
		t.Fatal(err)
	}
	if want := `["WARN",[{"default":"30s","env":"SLIPSTREAM_TIMINGS_NODE_STOP_DEADLINE","key":"timings.node.stop_deadline","source":"env","value":"5s"}]]`; string(got) != want {
		t.Fatalf("warning = %s, want %s", got, want)
	}
}

func TestStopWaitsUntilTheDataDirIsFree(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	p := startNode(t, dir)
	p.ready()

	res := slipstream(t, childEnv(), "stop", "--data-dir", dir)
	if res.code != node.ExitOK || res.stderr != "" {
		t.Fatalf("stop: exit %d, stderr %q; want exit 0 and no output", res.code, res.stderr)
	}
	if code := p.wait(); code != node.ExitOK {
		t.Fatalf("node exit %d after stop, want 0:\n%s", code, p.output())
	}
	if last := p.lastRecord(); last != "stopped" {
		t.Fatalf("last record = %q, want stopped:\n%s", last, p.output())
	}
	if _, err := os.Stat(filepath.Join(dir, "run", "node.pid")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("PID file after a clean stop: %v, want removed", err)
	}

	again := startNode(t, dir)
	again.ready()
	if res := slipstream(t, childEnv(), "stop", "--data-dir", dir); res.code != node.ExitOK {
		t.Fatalf("second stop: exit %d, stderr %q", res.code, res.stderr)
	}
	if code := again.wait(); code != node.ExitOK {
		t.Fatalf("restarted node exit %d, want 0:\n%s", code, again.output())
	}
}

func TestStopWithNoNodeFails(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	res := slipstream(t, childEnv(), "stop", "--data-dir", dir)
	if want := "slipstream stop: no Node answered on " + dir + ": "; res.code != node.ExitFailed || !strings.HasPrefix(res.stderr, want) {
		t.Fatalf("exit %d, stderr %q; want exit %d starting %q", res.code, res.stderr, node.ExitFailed, want)
	}
}

func TestConfigPrintsResolvedKeysAndCensus(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("timings:\n  node:\n    stop_deadline: 5s\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := slipstream(t, childEnv("SLIPSTREAM_SEED=fixed"), "config", "--data-dir", dir)
	if res.code != node.ExitOK {
		t.Fatalf("exit %d, stderr %q", res.code, res.stderr)
	}
	var got node.Resolved
	if err := json.Unmarshal([]byte(res.stdout), &got); err != nil {
		t.Fatalf("stdout %q: %v", res.stdout, err)
	}
	host, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	want := node.Resolved{
		Keys: []node.Key{
			{Key: "seed", Value: "fixed", Default: "", Source: "env", Env: "SLIPSTREAM_SEED"},
			{Key: "name", Value: host, Default: host, Source: "default", Env: "SLIPSTREAM_NAME"},
			{Key: "torrent.listen", Value: "0.0.0.0:6881", Default: "0.0.0.0:6881", Source: "default", Env: "SLIPSTREAM_TORRENT_LISTEN"},
			{Key: "torrent.downloads", Value: "downloads", Default: "downloads", Source: "default", Env: "SLIPSTREAM_TORRENT_DOWNLOADS"},
			{Key: "torrent.outgoing_utp", Value: "true", Default: "true", Source: "default", Env: "SLIPSTREAM_TORRENT_OUTGOING_UTP"},
			{Key: "torrent.loopback_peers", Value: "false", Default: "false", Source: "default", Env: "SLIPSTREAM_TORRENT_LOOPBACK_PEERS"},
			{Key: "http.listen", Value: "[::]:8484", Default: "[::]:8484", Source: "default", Env: "SLIPSTREAM_HTTP_LISTEN"},
			{Key: "http.trusted_origins", Value: "", Default: "", Source: "default", Env: "SLIPSTREAM_HTTP_TRUSTED_ORIGINS"},
			{Key: "http.ports_to_try", Value: "10", Default: "10", Source: "default", Env: "SLIPSTREAM_HTTP_PORTS_TO_TRY"},
			{Key: "http.trusted_proxies", Value: "", Default: "", Source: "default", Env: "SLIPSTREAM_HTTP_TRUSTED_PROXIES"},
			{Key: "link.listen", Value: "[::]:8485", Default: "[::]:8485", Source: "default", Env: "SLIPSTREAM_LINK_LISTEN"},
			{Key: "link.advertise", Value: "", Default: "", Source: "default", Env: "SLIPSTREAM_LINK_ADVERTISE"},
			{Key: "link.stream_window", Value: "16777216", Default: "16777216", Source: "default", Env: "SLIPSTREAM_LINK_STREAM_WINDOW"},
			{Key: "join.code", Value: "", Default: "", Source: "default", Env: "SLIPSTREAM_JOIN_CODE"},
			{Key: "log.level", Value: "info", Default: "info", Source: "default", Env: "SLIPSTREAM_LOG_LEVEL"},
			{Key: "log.max_size_mb", Value: "10", Default: "10", Source: "default", Env: "SLIPSTREAM_LOG_MAX_SIZE_MB"},
			{Key: "log.max_files", Value: "5", Default: "5", Source: "default", Env: "SLIPSTREAM_LOG_MAX_FILES"},
			{Key: "log.max_age_days", Value: "30", Default: "30", Source: "default", Env: "SLIPSTREAM_LOG_MAX_AGE_DAYS"},
			{Key: "log.compress", Value: "true", Default: "true", Source: "default", Env: "SLIPSTREAM_LOG_COMPRESS"},
			{Key: "log.viewer_lines", Value: "2000", Default: "2000", Source: "default", Env: "SLIPSTREAM_LOG_VIEWER_LINES"},
			{Key: "endpoints.tmdb.api", Value: "https://api.themoviedb.org/3", Default: "https://api.themoviedb.org/3", Source: "default", Env: "SLIPSTREAM_ENDPOINTS_TMDB_API"},
			{Key: "timings.node.stop_deadline", Value: "5s", Default: "30s", Source: "config.yaml", Env: "SLIPSTREAM_TIMINGS_NODE_STOP_DEADLINE"},
			{Key: "timings.node.restart_lock_wait", Value: "30s", Default: "30s", Source: "default", Env: "SLIPSTREAM_TIMINGS_NODE_RESTART_LOCK_WAIT"},
			{Key: "timings.storage.busy_timeout", Value: "5s", Default: "5s", Source: "default", Env: "SLIPSTREAM_TIMINGS_STORAGE_BUSY_TIMEOUT"},
			{Key: "timings.http.read_header", Value: "10s", Default: "10s", Source: "default", Env: "SLIPSTREAM_TIMINGS_HTTP_READ_HEADER"},
			{Key: "timings.http.idle", Value: "2m0s", Default: "2m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_HTTP_IDLE"},
			{Key: "timings.live.heartbeat", Value: "25s", Default: "25s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LIVE_HEARTBEAT"},
			{Key: "timings.live.retry", Value: "3s", Default: "3s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LIVE_RETRY"},
			{Key: "timings.link.handshake", Value: "10s", Default: "10s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LINK_HANDSHAKE"},
			{Key: "timings.link.keepalive", Value: "25s", Default: "25s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LINK_KEEPALIVE"},
			{Key: "timings.link.write_timeout", Value: "10s", Default: "10s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LINK_WRITE_TIMEOUT"},
			{Key: "timings.link.redial_min", Value: "1s", Default: "1s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LINK_REDIAL_MIN"},
			{Key: "timings.link.redial_max", Value: "1m0s", Default: "1m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LINK_REDIAL_MAX"},
			{Key: "timings.link.fallback_delay", Value: "300ms", Default: "300ms", Source: "default", Env: "SLIPSTREAM_TIMINGS_LINK_FALLBACK_DELAY"},
			{Key: "timings.link.log_poll", Value: "20s", Default: "20s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LINK_LOG_POLL"},
			{Key: "timings.link.log_archive_ttl", Value: "10m0s", Default: "10m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LINK_LOG_ARCHIVE_TTL"},
			{Key: "timings.link.join_dial", Value: "20s", Default: "20s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LINK_JOIN_DIAL"},
			{Key: "timings.link.await_dial", Value: "10m0s", Default: "10m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_LINK_AWAIT_DIAL"},
			{Key: "timings.role.restart_min", Value: "1s", Default: "1s", Source: "default", Env: "SLIPSTREAM_TIMINGS_ROLE_RESTART_MIN"},
			{Key: "timings.role.restart_max", Value: "1m0s", Default: "1m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_ROLE_RESTART_MAX"},
			{Key: "timings.role.beat", Value: "15s", Default: "15s", Source: "default", Env: "SLIPSTREAM_TIMINGS_ROLE_BEAT"},
			{Key: "timings.role.stall", Value: "45s", Default: "45s", Source: "default", Env: "SLIPSTREAM_TIMINGS_ROLE_STALL"},
			{Key: "timings.role.probe", Value: "10m0s", Default: "10m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_ROLE_PROBE"},
			{Key: "timings.nodes.seen_persist", Value: "1m0s", Default: "1m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_NODES_SEEN_PERSIST"},
			{Key: "timings.nodes.blip", Value: "1m0s", Default: "1m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_NODES_BLIP"},
			{Key: "timings.nodes.settle", Value: "1m0s", Default: "1m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_NODES_SETTLE"},
			{Key: "timings.nodes.dial_invite", Value: "10m0s", Default: "10m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_NODES_DIAL_INVITE"},
			{Key: "timings.nodes.dial_report", Value: "30s", Default: "30s", Source: "default", Env: "SLIPSTREAM_TIMINGS_NODES_DIAL_REPORT"},
			{Key: "timings.nodes.traffic", Value: "1s", Default: "1s", Source: "default", Env: "SLIPSTREAM_TIMINGS_NODES_TRAFFIC"},
			{Key: "timings.nodes.reach", Value: "10s", Default: "10s", Source: "default", Env: "SLIPSTREAM_TIMINGS_NODES_REACH"},
			{Key: "timings.works.fetch", Value: "30s", Default: "30s", Source: "default", Env: "SLIPSTREAM_TIMINGS_WORKS_FETCH"},
			{Key: "timings.indexers.request", Value: "30s", Default: "30s", Source: "default", Env: "SLIPSTREAM_TIMINGS_INDEXERS_REQUEST"},
			{Key: "timings.transfers.chunk", Value: "30s", Default: "30s", Source: "default", Env: "SLIPSTREAM_TIMINGS_TRANSFERS_CHUNK"},
			{Key: "timings.grabs.retry_min", Value: "1s", Default: "1s", Source: "default", Env: "SLIPSTREAM_TIMINGS_GRABS_RETRY_MIN"},
			{Key: "timings.grabs.retry_max", Value: "1m0s", Default: "1m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_GRABS_RETRY_MAX"},
			{Key: "timings.grabs.progress", Value: "1s", Default: "1s", Source: "default", Env: "SLIPSTREAM_TIMINGS_GRABS_PROGRESS"},
			{Key: "timings.files.retry_min", Value: "1s", Default: "1s", Source: "default", Env: "SLIPSTREAM_TIMINGS_FILES_RETRY_MIN"},
			{Key: "timings.files.retry_max", Value: "1m0s", Default: "1m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_FILES_RETRY_MAX"},
			{Key: "timings.downloading.facts", Value: "1s", Default: "1s", Source: "default", Env: "SLIPSTREAM_TIMINGS_DOWNLOADING_FACTS"},
			{Key: "timings.downloading.announce_floor", Value: "5m0s", Default: "5m0s", Source: "default", Env: "SLIPSTREAM_TIMINGS_DOWNLOADING_ANNOUNCE_FLOOR"},
			{Key: "timings.downloading.route_check", Value: "30s", Default: "30s", Source: "default", Env: "SLIPSTREAM_TIMINGS_DOWNLOADING_ROUTE_CHECK"},
			{Key: "timings.media.probe", Value: "30s", Default: "30s", Source: "default", Env: "SLIPSTREAM_TIMINGS_MEDIA_PROBE"},
			{Key: "timings.mail.send", Value: "15s", Default: "15s", Source: "default", Env: "SLIPSTREAM_TIMINGS_MAIL_SEND"},
			{Key: "timings.serving.drain", Value: "30s", Default: "30s", Source: "default", Env: "SLIPSTREAM_TIMINGS_SERVING_DRAIN"},
			{Key: "timings.storing.stat", Value: "10s", Default: "10s", Source: "default", Env: "SLIPSTREAM_TIMINGS_STORING_STAT"},
		},
		Census:       []string{"endpoints.tmdb.api"},
		ContentTypes: []string{"movie"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("config = %+v, want %+v", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "run")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("config started a Node: run dir %v", err)
	}

	res = slipstream(t, childEnv("SLIPSTREAM_SEDE=fixed"), "config", "--data-dir", dir)
	if want := "slipstream config: env: SLIPSTREAM_SEDE: unknown key\n"; res.code != node.ExitUsage || res.stderr != want {
		t.Fatalf("bad config: exit %d, stderr %q; want exit %d, %q", res.code, res.stderr, node.ExitUsage, want)
	}
}

func TestTheReleaseBinaryHasNoKeyForAContentTypeItLacks(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	res := slipstream(t, childEnv("SLIPSTREAM_ENDPOINTS_CONTENT_TYPES_MUSIC_METADATA=https://x.invalid"), "config", "--data-dir", dir)
	if want := "slipstream config: env: SLIPSTREAM_ENDPOINTS_CONTENT_TYPES_MUSIC_METADATA: unknown key\n"; res.code != node.ExitUsage || res.stderr != want {
		t.Fatalf("exit %d, stderr %q; want exit %d, %q", res.code, res.stderr, node.ExitUsage, want)
	}
}

func TestNodeOutlivesItsStderrReader(t *testing.T) {
	t.Parallel()
	dir := dataDir(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(context.Background(), testbin.Slipstream(t), "run", "--data-dir", dir)
	cmd.Env = childEnv(localListeners...)
	cmd.Stderr = w
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-exited
	})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if _, err := node.ParseReady(sc.Bytes()); err == nil {
			break
		}
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	res := slipstream(t, childEnv(), "stop", "--data-dir", dir)
	if res.code != node.ExitOK || res.stderr != "" {
		t.Fatalf("stop after the stderr reader closed: exit %d, stderr %q; want exit 0 and no output", res.code, res.stderr)
	}
	select {
	case <-exited:
	case <-time.After(60 * time.Second):
		t.Fatal("node still running 60s after stop")
	}
	if code := cmd.ProcessState.ExitCode(); code != node.ExitOK {
		t.Fatalf("node exit %d (%s), want 0", code, cmd.ProcessState)
	}
	logged, err := os.ReadFile(filepath.Join(dir, "logs", "slipstream.log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(logged)), "\n")
	var last struct{ Msg string }
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil || last.Msg != "stopped" {
		t.Fatalf("last log file record = %q (%v), want stopped:\n%s", last.Msg, err, logged)
	}
}

func TestRunRefusesToStartWithoutFFprobeBesideIt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bin := filepath.Join(dir, filepath.Base(testbin.Slipstream(t)))
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
	p := startBinary(t, bin, dataDir(t))
	if code := p.wait(); code != node.ExitFailed || p.output() != "ffprobe not found beside "+bin {
		t.Fatalf("run: exit %d, stderr %q; want exit %d, ffprobe not found beside %s", code, p.output(), node.ExitFailed, bin)
	}
	cmd := exec.CommandContext(t.Context(), bin, "config", "--data-dir", dataDir(t))
	cmd.Env = childEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("config without ffprobe: %v\n%s", err, out)
	}
}
