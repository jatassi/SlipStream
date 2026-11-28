// Package simconv is the simulation's Converting seams: a runner whose
// every run succeeds and proofs for the processor alone.
package simconv

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
)

// Succeeds writes a placeholder at the output, ffmpeg's last argument,
// and exits 0 at once.
type Succeeds struct{}

func (Succeeds) Start(_ context.Context, args []string) (converting.Run, error) {
	if err := os.WriteFile(args[len(args)-1], []byte("converted"), 0o600); err != nil {
		return nil, fmt.Errorf("write the output: %w", err)
	}
	progress := make(chan converting.Progress)
	close(progress)
	return exited{progress}, nil
}

type exited struct{ progress chan converting.Progress }

func (e exited) Progress() <-chan converting.Progress { return e.progress }
func (exited) Pause() error                           { return nil }
func (exited) Resume() error                          { return nil }
func (exited) Kill()                                  {}
func (exited) Wait() converting.Exit                  { return converting.Exit{} }
func (exited) Tail() string                           { return "" }

// ProcessorOnly passes every test run on the processor, at real time, and
// finds no graphics card.
type ProcessorOnly struct{}

// Signature is the one every simulated Node has.
var Signature = converting.Signature(strings.Repeat("5", 64))

func (ProcessorOnly) Signature(context.Context) (converting.Signature, []converting.Device, error) {
	return Signature, nil, nil
}

func (ProcessorOnly) Test(context.Context, encode.Key) (converting.Proved, error) {
	return converting.Proved{
		Passed: true, Pipeline: encode.Pipeline{Path: encode.Processor}, Speed: 1, Pauses: true, Signature: Signature,
	}, nil
}

func (ProcessorOnly) Runs() (role.Fit, role.ProbeReason) { return role.Weak, role.NoGPU }

func (ProcessorOnly) Warm(context.Context, func()) error { return nil }

// Default is node.Env.Converting for a simulated Node: Succeeds and
// ProcessorOnly.
func Default(converting.RealConfig) (converting.Converter, converting.Proofs) {
	return Succeeds{}, ProcessorOnly{}
}
