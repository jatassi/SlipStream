package proc_test

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/internal/proc"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// A test removes a run dir once its Nodes are gone, so a process Alive
// reports gone must no longer hold its executable.
func TestAProcessThatIsNotAliveReleasesItsExecutable(t *testing.T) {
	t.Parallel()
	for range 20 {
		exe := filepath.Join(t.TempDir(), "child.exe")
		copyFile(t, os.Args[0], exe)
		cmd := exec.CommandContext(worldtest.Context(t), exe)
		cmd.Env = append(os.Environ(), "PROC_TEST_UNTIL_EOF=1")
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		for proc.Alive(cmd.Process.Pid) {
			runtime.Gosched()
		}
		if err := os.Remove(exe); err != nil {
			t.Fatalf("remove the executable of an exited process: %v", err)
		}
		_ = cmd.Wait()
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, in); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}
