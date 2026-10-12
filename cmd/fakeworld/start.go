package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jatassi/slipstream/fakes/world"
)

type report struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type outcome int

const (
	ready outcome = iota
	refused
	died
	timedOut
	interrupt
)

type handshake struct {
	outcome outcome
	detail  string
}

func readReport(stdout io.Reader) handshake {
	line, err := bufio.NewReader(stdout).ReadBytes('\n')
	if err != nil {
		return handshake{outcome: died}
	}
	var r report
	if err := json.Unmarshal(line, &r); err != nil {
		return handshake{outcome: died, detail: fmt.Sprintf("its report %q is not JSON", line)}
	}
	if !r.OK {
		return handshake{outcome: refused, detail: r.Error}
	}
	return handshake{outcome: ready}
}

func start(f startFlags) error {
	runDir, err := filepath.Abs(f.runDir)
	if err != nil {
		return fmt.Errorf("run dir: %w", err)
	}
	binary, err := filepath.Abs(f.binary)
	if err != nil {
		return fmt.Errorf("binary: %w", err)
	}
	if _, err := world.CheckAddresses(context.Background(), len(f.scenario.Nodes)); err != nil {
		return err
	}
	if err := world.CreateRunDir(runDir); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(runDir, "world.log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644) //nolint:gosec // world.log is evidence anyone on the machine may read
	if err != nil {
		return fmt.Errorf("open world.log: %w", err)
	}
	defer func() { _ = log.Close() }()
	cmd, err := supervisorCommand(runDir, binary, f.scenario.Name, log)
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("supervisor: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("supervisor: %w", err)
	}
	signals, stopSignals := interrupted()
	defer stopSignals()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("supervisor: %w", err)
	}
	reports := make(chan handshake, 1)
	go func() { reports <- readReport(stdout) }()

	h := await(signals, reports, f.timeout)
	_ = stdin.Close()
	switch h.outcome {
	case ready:
		fmt.Println(filepath.Join(runDir, "manifest.json"))
		return nil
	case timedOut, interrupt:
		abandon(runDir, <-reports)
	case refused, died:
	}
	_ = cmd.Wait()
	return h.err(f.timeout, cmd.ProcessState.ExitCode(), filepath.Join(runDir, "world.log"))
}

func supervisorCommand(runDir, binary, scenario string, log *os.File) (*exec.Cmd, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("supervisor: %w", err)
	}
	supervisor := filepath.Join(runDir, "bin", "fakeworld"+filepath.Ext(self))
	if _, err := world.CopyExecutable(self, supervisor); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(context.Background(), supervisor, "supervise", "--binary", binary, "--scenario", scenario, "--run-dir", runDir)
	cmd.Stderr = log
	cmd.SysProcAttr = detached()
	return cmd, nil
}

func await(signals context.Context, reports <-chan handshake, timeout time.Duration) handshake {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	select {
	case h := <-reports:
		return h
	case <-deadline.C:
		return handshake{outcome: timedOut}
	case <-signals.Done():
		return handshake{outcome: interrupt}
	}
}

func abandon(runDir string, late handshake) {
	if late.outcome != ready {
		return
	}
	if _, err := world.StopRun(context.Background(), runDir); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fakeworld start: stop the world it abandoned: %v\n", err)
	}
}

func (h handshake) err(timeout time.Duration, code int, worldLog string) error {
	switch h.outcome {
	case refused:
		return errors.New(h.detail)
	case died:
		msg := fmt.Sprintf("the supervisor exited with code %d before it reported", code)
		if h.detail != "" {
			msg += ": " + h.detail
		}
		return fmt.Errorf("%s; see %s", msg, worldLog)
	case timedOut:
		return fmt.Errorf("launch did not finish within %s", timeout)
	case interrupt:
		return errors.New("interrupted; the launch was abandoned")
	case ready:
	}
	return nil
}
