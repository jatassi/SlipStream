package proc_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/internal/proc"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func TestMain(m *testing.M) {
	if os.Getenv("PROC_TEST_UNTIL_EOF") == "1" {
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	if os.Getenv("PROC_TEST_EXIT") == "1" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestWaitExit(t *testing.T) {
	t.Parallel()
	cmd := exec.CommandContext(worldtest.Context(t), os.Args[0])
	cmd.Env = append(os.Environ(), "PROC_TEST_UNTIL_EOF=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid

	ended, cancel := context.WithCancel(worldtest.Context(t))
	cancel()
	if err, want := proc.WaitExit(ended, pid), fmt.Sprintf("PID %d did not exit: context canceled", pid); err == nil || err.Error() != want {
		t.Fatalf("WaitExit on a running process with an ended context = %v, want %s", err, want)
	}
	_ = stdin.Close()
	if err := proc.WaitExit(worldtest.Context(t), pid); err != nil {
		t.Fatalf("WaitExit on a process that exits = %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := proc.WaitExit(worldtest.Context(t), pid); err != nil {
		t.Fatalf("WaitExit on a reaped process = %v", err)
	}
}

func TestKillingAProcessThatHasExitedSucceeds(t *testing.T) {
	t.Parallel()
	p, err := proc.Start(worldtest.Context(t), os.Args[0], nil, append(os.Environ(), "PROC_TEST_EXIT=1"), io.Discard, func([]byte) {})
	if err != nil {
		t.Fatal(err)
	}
	<-p.Exited()
	if err := p.Kill(); err != nil {
		t.Fatalf("Kill after the process exited = %v", err)
	}
}
