package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/world"
)

type execFlags struct {
	runDir string
	proof  string
	// args are exec's arguments after the run dir, as the journal keeps
	// them; argv is the command, unexpanded.
	args, argv []string
}

var proofShape = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9-]*$`)

func parseExec(args []string) (execFlags, error) {
	if len(args) == 0 {
		return execFlags{}, errors.New("exec: takes a run dir")
	}
	f := execFlags{runDir: args[0], args: args[1:]}
	sep := slices.Index(f.args, "--")
	if sep < 0 {
		return execFlags{}, errors.New("exec: put the command after --")
	}
	fs := newFlagSet("exec")
	fs.StringVar(&f.proof, "proof", "", "")
	if err := noArguments(fs, f.args[:sep]); err != nil {
		return execFlags{}, err
	}
	if f.argv = f.args[sep+1:]; len(f.argv) == 0 {
		return execFlags{}, errors.New("exec: no command after --")
	}
	if f.proof != "" && !proofShape.MatchString(f.proof) {
		return execFlags{}, fmt.Errorf("exec: --proof %q is not <feature>/<sub>", f.proof)
	}
	return f, nil
}

// execStep runs the command as one journaled step: no shell, its tokens
// expanded against the manifest now, its output kept in the journal or
// the proof dir, and its exit the step's and fakeworld's.
func execStep(f execFlags) int {
	m, err := world.ReadManifest(f.runDir)
	if err != nil {
		return failed("exec", err)
	}
	argv, err := world.Expand(m, f.argv)
	if err != nil {
		return usage(fmt.Errorf("exec: %w", err))
	}
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return usage(fmt.Errorf("exec: %w", err))
	}
	ctx, stop := interrupted()
	defer stop()
	return journal(ctx, f.runDir, "exec", f.args, func(ctx context.Context, _ world.Controls, s world.Step) (int, map[string]string) {
		files := world.StepOutputs(s.N, f.proof)
		exit, err := runCommand(ctx, f.runDir, files, path, argv[1:])
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "fakeworld exec: %v\n", err)
		}
		printNodeErrors(f.runDir, s)
		return exit, files
	})
}

func runCommand(ctx context.Context, runDir string, files map[string]string, path string, args []string) (int, error) {
	out, err := create(runDir, files["stdout"])
	if err != nil {
		return exitFailed, err
	}
	defer func() { _ = out.Close() }()
	errOut, err := create(runDir, files["stderr"])
	if err != nil {
		return exitFailed, err
	}
	defer func() { _ = errOut.Close() }()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = io.MultiWriter(out, os.Stdout)
	cmd.Stderr = io.MultiWriter(errOut, os.Stderr)
	err = cmd.Run()
	if _, exited := errors.AsType[*exec.ExitError](err); err != nil && !exited {
		return exitFailed, fmt.Errorf("%s: %w", path, err)
	}
	if code := cmd.ProcessState.ExitCode(); code >= 0 {
		return code, nil
	}
	return exitFailed, fmt.Errorf("%s was killed", path)
}

func create(runDir, rel string) (*os.File, error) {
	path := filepath.Join(runDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("step output: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("step output: %w", err)
	}
	return f, nil
}

// printNodeErrors names each ERROR a Node logged since step s started.
func printNodeErrors(runDir string, s world.Step) {
	recs, err := events.Select(runDir, events.Query{Kind: events.NodeError, After: s.Seq})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fakeworld exec: %v\n", err)
		return
	}
	for _, r := range recs {
		_, _ = fmt.Fprintf(os.Stderr, "fakeworld exec: node %s logged an ERROR during step %d: %s\n", r.Node, s.N, r.Data)
	}
}
