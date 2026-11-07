package proc_test

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
// after stop must not mistake that process for the one that stopped. A
// watch keeps its PID from reuse, so that PID still names the process
// that exited.
func TestAWatchedProcessReadsAsGoneOnceItsPIDIsReaped(t *testing.T) {
	t.Parallel()
	watched := start(t)
	pid := watched.Process.Pid
	w, err := proc.Watch(pid)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	reap(t, watched)
	if !w.Gone() {
		t.Fatalf("the reaped PID %d reads as running", pid)
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid)) //nolint:gosec // a PID is positive and fits
	if err != nil {
		t.Fatalf("the watch let PID %d go: %v", pid, err)
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("PID %d holds exit code %d, want the 0 of the process the watch kept it for", pid, code)
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
