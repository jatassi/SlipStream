package lifecycle_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world/internal/lifecycle"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
)

// A restart on Windows leaves a successor holding the Node's stderr, which
// closes before the successor counts as exited.
func TestStoppingARestartedNodeWaitsForTheSuccessorToExit(t *testing.T) {
	t.Parallel()
	bin := testbin.Build(t, t.TempDir(), "./fakes/world/testdata/stubnode", "-ldflags", "-X=main.successor=1 -X=main.linger=1")
	dir := shortDataDir(t)
	const config = "torrent: {listen: 127.0.0.1:0}\nhttp: {listen: 127.0.0.1:0}\nlink: {listen: 127.0.0.1:0}\n"
	if err := os.WriteFile(filepath.Join(dir.Path(), "config.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	n := lifecycle.New(lifecycle.Config{
		Name:    "home",
		Binary:  bin,
		DataDir: dir,
		Log:     filepath.Join(t.TempDir(), "home.log"),
		Record:  func(context.Context, events.Record) {},
		Append:  func(events.Record) {},
		Changed: func() {},
	})
	ctx := worldtest.Context(t)
	if err := n.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := n.Restart(ctx); err != nil {
		t.Fatal(err)
	}
	successor := n.PID()
	if err := n.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	exited, err := os.ReadFile(filepath.Join(dir.Path(), "exited"))
	if err != nil || string(exited) != strconv.Itoa(successor) {
		t.Fatalf("stop returned before the successor PID %d exited: its exited file holds %q, %v", successor, exited, err)
	}
}

func shortDataDir(t *testing.T) node.DataDir {
	t.Helper()
	path, err := os.MkdirTemp("", "ss") //nolint:usetesting // t.TempDir's path is too long for the socket limit on macOS
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(path) })
	dir, err := node.ParseDataDir(path)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
