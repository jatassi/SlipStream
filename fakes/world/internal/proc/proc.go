// Package proc spawns the world's child processes and waits on their
// exits as events.
package proc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// TailLines is how much of a process's stderr a failure quotes.
const TailLines = 40

type Process struct {
	PID    int
	cmd    *exec.Cmd
	stderr *lineWriter
	exited chan struct{}
	code   int
}

// Start runs bin until it exits or is killed: it outlives ctx. Each whole
// line of its stderr goes to log and then to watch, on one goroutine in
// order. Its stdout is discarded.
func Start(ctx context.Context, bin string, args, env []string, log io.Writer, watch func(line []byte)) (*Process, error) {
	stderr := &lineWriter{log: log, watch: watch}
	cmd := exec.CommandContext(context.WithoutCancel(ctx), bin, args...)
	cmd.Env = env
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", bin, err)
	}
	p := &Process{PID: cmd.Process.Pid, cmd: cmd, stderr: stderr, exited: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		stderr.flush()
		p.code = cmd.ProcessState.ExitCode()
		close(p.exited)
	}()
	return p, nil
}

// Exited is closed once the process has exited and all its stderr is read.
func (p *Process) Exited() <-chan struct{} { return p.exited }

// ExitCode is valid once Exited is closed. It is -1 for a killed process.
func (p *Process) ExitCode() int { return p.code }

// Kill kills the process. One that has already exited is no error: once
// waited on, Unix reports it done and Windows reports it released (EINVAL).
func (p *Process) Kill() error {
	if err := p.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, syscall.EINVAL) {
		return fmt.Errorf("kill PID %d: %w", p.PID, err)
	}
	return nil
}

// Kill kills the process pid, which need not be this process's child.
func Kill(pid int) error {
	p, err := os.FindProcess(pid)
	if err == nil {
		err = p.Kill()
	}
	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("kill PID %d: %w", pid, err)
	}
	return nil
}

// Tail is the last TailLines lines of stderr.
func (p *Process) Tail() []string { return p.stderr.tail() }

type lineWriter struct {
	log   io.Writer
	watch func([]byte)

	partial []byte
	mu      sync.Mutex
	last    []string
}

func (w *lineWriter) Write(b []byte) (int, error) {
	_, _ = w.log.Write(b)
	w.partial = append(w.partial, b...)
	for {
		i := bytes.IndexByte(w.partial, '\n')
		if i < 0 {
			return len(b), nil
		}
		w.line(w.partial[:i])
		w.partial = w.partial[i+1:]
	}
}

func (w *lineWriter) flush() {
	if len(w.partial) > 0 {
		w.line(w.partial)
		w.partial = nil
	}
}

func (w *lineWriter) line(l []byte) {
	l = bytes.TrimSuffix(l, []byte("\r"))
	w.mu.Lock()
	w.last = append(w.last, string(l))
	if len(w.last) > TailLines {
		w.last = w.last[len(w.last)-TailLines:]
	}
	w.mu.Unlock()
	w.watch(l)
}

func (w *lineWriter) tail() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.last...)
}

type Result struct {
	Stdout, Stderr []byte
	Code           int
}

// Run runs bin to its exit under ctx.
func Run(ctx context.Context, bin string, args, env []string) (Result, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return Result{}, fmt.Errorf("%s %s: %w", bin, args[0], ctx.Err())
	}
	if _, exited := errors.AsType[*exec.ExitError](err); err != nil && !exited {
		return Result{}, fmt.Errorf("%s %s: %w", bin, args[0], err)
	}
	return Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Code: cmd.ProcessState.ExitCode()}, nil
}
