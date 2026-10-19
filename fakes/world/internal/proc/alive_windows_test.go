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
// after stop must not mistake that process for the one that stopped.
func TestAReapedProcessReadsAsGoneOnceItsPIDIsReused(t *testing.T) {
	t.Parallel()
	cmd := exec.CommandContext(worldtest.Context(t), os.Args[0])
	cmd.Env = append(os.Environ(), "PROC_TEST_EXIT=1")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	parkUntilReused(t, pid)
	if proc.Alive(pid) {
		t.Fatalf("the reaped PID %d reads as running once another process has it", pid)
	}
}

// parkUntilReused starts suspended processes until one gets pid. They stay
// until t ends, so none frees a PID that Windows would reuse before pid.
func parkUntilReused(t *testing.T, pid int) {
	t.Helper()
	const spawns = 1000
	for range spawns {
		cmd := exec.CommandContext(worldtest.Context(t), os.Args[0])
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		})
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
