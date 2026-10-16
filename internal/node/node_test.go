package node

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/media/mediatest"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Write(p)
	return len(p), nil
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func testEnv(t *testing.T) (Env, *syncBuffer) {
	t.Helper()
	d := testDataDir(t)
	stderr := new(syncBuffer)
	return Env{
		DataDir: d,
		Environ: []string{"SLIPSTREAM_TORRENT_LISTEN=127.0.0.1:0", "SLIPSTREAM_HTTP_LISTEN=127.0.0.1:0", "SLIPSTREAM_LINK_LISTEN=127.0.0.1:0"},
		Stderr:  stderr,
		Version: "test",
		Net:     osNetwork{},
		Engine:  func(c engine.Config) (engine.Engine, error) { return libtorrent.Open(c) },
		OpenLog: logs.Open,
		Media:   mediatest.Prober{},
	}, stderr
}

func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ss") //nolint:usetesting // t.TempDir's path is too long for the socket limit on macOS
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func startTest(t *testing.T, env Env) *Node {
	t.Helper()
	n, err := Start(t.Context(), env)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		n.Stop()
		<-n.Done()
	})
	return n
}

func waitDone(t *testing.T, n *Node) {
	t.Helper()
	select {
	case <-n.Done():
	case <-time.After(30 * time.Second):
		t.Fatal("node did not stop within 30s")
	}
}

func TestTwoNodesInOneProcessShareNothing(t *testing.T) {
	envA, stderrA := testEnv(t)
	envB, _ := testEnv(t)
	a, b := startTest(t, envA), startTest(t, envB)

	if a.Ready().Listeners.Torrent == b.Ready().Listeners.Torrent {
		t.Fatalf("both nodes report torrent address %s", a.Ready().Listeners.Torrent)
	}
	for _, c := range []struct {
		n   *Node
		env Env
	}{{a, envA}, {b, envB}} {
		logged, err := os.ReadFile(filepath.Join(c.env.DataDir.LogsDir(), "slipstream.log"))
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(string(logged), `"msg":"ready"`); got != 1 || !strings.Contains(string(logged), `"data_dir":"`+jsonPath(c.env.DataDir.Path())+`"`) {
			t.Fatalf("log of %s holds %d ready records, want only its own:\n%s", c.env.DataDir, got, logged)
		}
	}

	a.Stop()
	waitDone(t, a)
	if a.Err() != nil {
		t.Fatalf("a stopped with %v", a.Err())
	}
	select {
	case <-b.Done():
		t.Fatal("stopping a stopped b")
	default:
	}
	if lines := strings.Split(strings.TrimSpace(stderrA.String()), "\n"); !strings.Contains(lines[len(lines)-1], `"msg":"stopped"`) {
		t.Fatalf("a's last record = %s, want stopped", lines[len(lines)-1])
	}

	again := startTest(t, envA)
	if again.Ready().DataDir != envA.DataDir.Path() {
		t.Fatalf("restart on a's data dir reports %s", again.Ready().DataDir)
	}
}

func TestStartRefusesAHeldDataDir(t *testing.T) {
	env, _ := testEnv(t)
	startTest(t, env)
	second := env
	stderr := new(syncBuffer)
	second.Stderr = stderr
	_, err := Start(t.Context(), second)
	want := "data dir " + env.DataDir.Path() + " is in use by SlipStream PID " + strconv.Itoa(os.Getpid())
	if inUse := new(InUseError); !errors.As(err, &inUse) || err.Error() != want || ExitCode(err) != ExitInUse {
		t.Fatalf("err = %v, want %s with exit %d", err, want, ExitInUse)
	}
	if stderr.String() != want+"\n" {
		t.Fatalf("stderr = %q, want the refusal as plain text", stderr.String())
	}
}

func TestStartRemovesTheLogArchivesAStoppedProcessLeft(t *testing.T) {
	env, _ := testEnv(t)
	run := env.DataDir.RunDir()
	if err := os.MkdirAll(run, 0o700); err != nil {
		t.Fatal(err)
	}
	stale, kept := filepath.Join(run, "logs-3141592653.zip"), filepath.Join(run, "notes.zip")
	writeFile(t, stale, "a zip its reader never finished")
	writeFile(t, kept, "not an archive")
	startTest(t, env)
	_, staleErr := os.Stat(stale)
	if _, keptErr := os.Stat(kept); !errors.Is(staleErr, fs.ErrNotExist) || keptErr != nil {
		t.Fatalf("after start, stat %s = %v and %s = %v; want the archive gone and the other file kept", stale, staleErr, kept, keptErr)
	}
}

func jsonPath(p string) string { return strings.ReplaceAll(p, `\`, `\\`) }
