package main

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/internal/node"
)

// TestTheTrampolineStartsTheAppAndTheConsoleKeepsItsExitCodes starts a
// ZIP install's slipstream-app.exe, built as a GUI program, the way a
// shortcut does, then stops the Node it started through cmd and
// PowerShell, which only wait for and report the exit code of a console
// program.
func TestTheTrampolineStartsTheAppAndTheConsoleKeepsItsExitCodes(t *testing.T) {
	t.Parallel()
	bin := installedCopy(t, "zip\n")
	trampoline := testbin.Build(t, filepath.Dir(bin), "./cmd/slipstream-app", "-ldflags=-H windowsgui")
	dir := filepath.Join(filepath.Dir(bin), "data")
	launch := exec.CommandContext(t.Context(), trampoline)
	launch.Env = childEnv(localListeners...)
	if out, err := launch.CombinedOutput(); err != nil {
		t.Fatalf("slipstream-app: %v\n%s", err, out)
	}
	t.Cleanup(func() { _ = exec.CommandContext(context.Background(), bin, "stop", "--data-dir", dir).Run() })
	ready := awaitReadyLog(t, filepath.Join(dir, "logs", "slipstream.log"))

	got := statusOf(t, dir)
	t.Logf("tray %v", got["tray"])
	if login, ok := got["start_at_login"].(bool); !ok || login || got["pid"] != float64(ready.PID) || got["mode"] != "app" || got["install"] != "zip" {
		t.Fatalf("status = %v, want the trampoline's Node in app mode on a zip install with start_at_login false", got)
	}

	stop := []string{bin, "stop", "--data-dir", dir}
	powershell := func() *exec.Cmd {
		return exec.CommandContext(t.Context(), "powershell", "-NoProfile", "-NonInteractive", "-Command",
			"& '"+bin+"' stop --data-dir '"+dir+"'; exit $LASTEXITCODE")
	}
	for _, c := range []struct {
		name string
		cmd  *exec.Cmd
		want int
	}{
		{"PowerShell stop", powershell(), node.ExitOK},
		{"PowerShell stop again", powershell(), node.ExitFailed},
		{"cmd stop again", exec.CommandContext(t.Context(), "cmd", slices.Concat([]string{"/c"}, stop)...), node.ExitFailed},
	} {
		c.cmd.Env = childEnv()
		out, err := c.cmd.CombinedOutput()
		if _, exit := errors.AsType[*exec.ExitError](err); err != nil && !exit {
			t.Fatalf("%s: %v", c.name, err)
		}
		if code := c.cmd.ProcessState.ExitCode(); code != c.want {
			t.Fatalf("%s: exit %d, want %d\n%s", c.name, code, c.want, out)
		}
	}
}

// awaitReadyLog is the readiness record once the log at path holds it.
func awaitReadyLog(t *testing.T, path string) node.Ready {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if f, err := os.Open(path); err == nil {
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				if r, err := node.ParseReady(sc.Bytes()); err == nil {
					_ = f.Close()
					return r
				}
			}
			_ = f.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no readiness record in %s after 60s", path)
	return node.Ready{}
}
