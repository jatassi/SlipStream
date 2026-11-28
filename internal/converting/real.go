package converting

import (
	"bufio"
	"cmp"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/internal/converting/clip"
	"github.com/jatassi/slipstream/internal/converting/internal/drivers"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/proctree"
)

type RealConfig struct {
	// Tools are the bundled programs. Their Environ is the Node's Env.Environ.
	Tools   clip.Tools
	Scratch func() string
	// Override is the configured ffmpeg path, empty for the bundled one.
	Override func() string
	Media    media.Prober
	Timings  Timings
}

// Real is the production Converter and Proofs: the bundled ffmpeg, ffprobe
// and dovi_tool (or the override ffmpeg path), under proctree, with
// Env.Environ minus every SLIPSTREAM_ variable.
func Real(cfg RealConfig) (Converter, Proofs) {
	cfg.Tools.Environ = environ(cfg.Tools.Environ)
	c := &realConverter{cfg: cfg}
	return c, &realProofs{cfg: cfg, conv: c, adapters: func(ctx context.Context) ([]drivers.Adapter, error) {
		return drivers.Local(c.ffmpeg(), cfg.Tools.Environ)(ctx)
	}}
}

type realConverter struct{ cfg RealConfig }

func (c *realConverter) ffmpeg() string { return cmp.Or(c.cfg.Override(), c.cfg.Tools.FFmpeg) }

const tailSize = 2 << 10

// Start runs ffmpeg under proctree. Kill cancels the run's own context, so
// os/exec ends the tree with proctree's cancel, which leaves a nonzero
// exit code on every OS, and never after the run was waited for.
func (c *realConverter) Start(ctx context.Context, args []string) (Run, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, c.ffmpeg(), args...) //nolint:gosec // the Node's ffmpeg on arguments pipeline.Build made
	cmd.Env = c.cfg.Tools.Environ
	cmd.WaitDelay = c.cfg.Timings.KillGrace
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open ffmpeg's input: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open ffmpeg's output: %w", err)
	}
	r := &realRun{stdin: stdin, progress: make(chan Progress, 1), done: make(chan struct{}), kill: cancel}
	cmd.Stderr = &r.tail
	end, err := proctree.Start(cmd)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("start ffmpeg: %w", err)
	}
	r.end = end
	go r.follow(cmd, stdout)
	return r, nil
}

type realRun struct {
	stdin    io.WriteCloser
	progress chan Progress
	kill     context.CancelFunc
	end      func()
	tail     tail
	done     chan struct{}
	exit     Exit
}

// follow reads ffmpeg's progress until it closes its output, then waits
// for it and ends its tree, so nothing it started outlives it.
func (r *realRun) follow(cmd *exec.Cmd, stdout io.Reader) {
	lines := bufio.NewScanner(stdout)
	var p Progress
	for lines.Scan() {
		key, value, _ := strings.Cut(lines.Text(), "=")
		switch key {
		case "out_time_us":
			if us, err := strconv.ParseInt(value, 10, 64); err == nil {
				p.OutTimeUs = us
			}
		case "speed":
			if x, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(value), "x"), 64); err == nil {
				p.Speed = x
			}
		case "progress":
			r.latest(p)
		}
	}
	_ = cmd.Wait()
	r.end()
	r.kill()
	r.exit = Exit{Code: cmd.ProcessState.ExitCode(), Tail: r.tail.String()}
	close(r.progress)
	close(r.done)
}

// latest replaces any progress the reader hasn't taken, so a reader that
// stops reading never holds ffmpeg up.
func (r *realRun) latest(p Progress) {
	select {
	case r.progress <- p:
	default:
		select {
		case <-r.progress:
		default:
		}
		r.progress <- p
	}
}

func (r *realRun) Progress() <-chan Progress { return r.progress }

func (r *realRun) Pause() error { return r.key("p") }

func (r *realRun) Resume() error { return r.key("u") }

func (r *realRun) key(k string) error {
	if _, err := io.WriteString(r.stdin, k); err != nil {
		return fmt.Errorf("write %q to ffmpeg: %w", k, err)
	}
	return nil
}

func (r *realRun) Kill() { r.kill() }

func (r *realRun) Wait() Exit {
	<-r.done
	return r.exit
}

func (r *realRun) Tail() string { return r.tail.String() }

// tail is the last tailSize bytes written to it.
type tail struct {
	mu  sync.Mutex
	buf []byte
}

func (t *tail) Write(b []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, b...)
	if over := len(t.buf) - tailSize; over > 0 {
		t.buf = append(t.buf[:0], t.buf[over:]...)
	}
	return len(b), nil
}

func (t *tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}
