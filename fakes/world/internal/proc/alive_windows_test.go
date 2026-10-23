package proc_test

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"

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

// Windows hands a reaped process's PID to a later process, so a check
// after stop must not mistake that process for the one that stopped. The
// watch keeps its own PID from reuse, so an unwatched process's PID shows
// that Windows reused freed PIDs meanwhile.
func TestAWatchedProcessReadsAsGoneOnceItsPIDIsReaped(t *testing.T) {
	t.Parallel()
	watched := start(t)
	w, err := proc.Watch(watched.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	reap(t, watched)
	unwatched := start(t)
	reap(t, unwatched)
	parkUntilReused(t, unwatched.Process.Pid)
	if !w.Gone() {
		t.Fatalf("the reaped PID %d reads as running once Windows has reused freed PIDs", watched.Process.Pid)
	}
}

func start(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.CommandContext(worldtest.Context(t), os.Args[0])
	cmd.Env = append(os.Environ(), "PROC_TEST_EXIT=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func reap(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}

// parkUntilReused starts suspended processes until one gets pid. Each is
// killed at once, freeing its memory, but its handle stays open until t
// ends, so none frees a PID that Windows would reuse before pid.
func parkUntilReused(t *testing.T, pid int) {
	t.Helper()
	const spawns = 1000
	for range spawns {
		cmd := exec.CommandContext(worldtest.Context(t), os.Args[0])
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		_ = cmd.Process.Kill()
		t.Cleanup(func() { _ = cmd.Wait() })
		if cmd.Process.Pid == pid {
			return
		}
	}
	t.Fatalf("no process got PID %d within %d spawns", pid, spawns)
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
