package main

import (
	"context"
	gobuildinfo "debug/buildinfo"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/worktree"
)

type replayFlags struct {
	bundle, binary, runDir string
	to                     int
	stop                   bool
	timeout                time.Duration
}

func parseReplay(args []string) (replayFlags, error) {
	if len(args) == 0 {
		return replayFlags{}, errors.New("replay: takes a bundle")
	}
	fs := newFlagSet("replay")
	f := replayFlags{bundle: args[0]}
	fs.StringVar(&f.binary, "binary", "", "")
	fs.StringVar(&f.runDir, "run-dir", "", "")
	fs.IntVar(&f.to, "to", 0, "")
	fs.BoolVar(&f.stop, "stop", false, "")
	fs.DurationVar(&f.timeout, "timeout", defaultTimeout, "")
	if err := noArguments(fs, args[1:]); err != nil {
		return replayFlags{}, err
	}
	if f.to < 0 {
		return replayFlags{}, fmt.Errorf("replay: --to %d is not a step", f.to)
	}
	return f, positiveTimeout(fs, f.timeout)
}

// replay runs the bundle's run again: a steps bundle's steps on a fresh
// world, or a test bundle's test against the bundle's binary.
func replay(args []string) int {
	f, err := parseReplay(args)
	if err != nil {
		return usage(err)
	}
	b, err := world.ReadBundle(f.bundle, f.binary)
	if err != nil {
		return failed("replay", err)
	}
	if b.Origin.Kind == world.FromTest {
		if f.to > 0 || f.stop {
			return usage(errors.New("replay: a test bundle replays by its command and takes no --to or --stop"))
		}
		return replayTest(f, b)
	}
	return replaySteps(f, b)
}

// replaySteps starts a world from the bundle's Scenario and dispatches
// steps 1 to --to, all of them by default, as the CLI would, stopping at
// the first whose exit differs from the one recorded.
func replaySteps(f replayFlags, b world.Bundle) int {
	steps := b.Steps
	if f.to > len(steps) {
		return usage(fmt.Errorf("replay: --to %d, and the bundle journals %d steps", f.to, len(steps)))
	}
	if f.to > 0 {
		steps = steps[:f.to]
	}
	for _, s := range steps {
		if s.Verb == "" {
			return failed("replay", fmt.Errorf("step %d is a Go test's note, which replays only with its test", s.N))
		}
		if s.Exit == nil {
			return failed("replay", fmt.Errorf("step %d never ended; replay --to %d", s.N, s.N-1))
		}
	}
	runDir, err := replayDir(f)
	if err != nil {
		return failed("replay", err)
	}
	scenarioFile, err := filepath.Abs(filepath.Join(b.Dir, "scenario.yaml"))
	if err != nil {
		return failed("replay", err)
	}
	if err := start(startFlags{binary: b.Binary, runDir: runDir, scenario: b.Scenario, scenarioFile: scenarioFile, timeout: f.timeout}); err != nil {
		return failed("replay", err)
	}
	exit := exitOK
	for _, s := range steps {
		fields := strings.Fields(s.Verb)
		argv := append(append(fields[:len(fields):len(fields)], runDir), s.Args...)
		if got := run(argv); got != *s.Exit {
			_, _ = fmt.Fprintf(os.Stderr, "fakeworld replay: step %d exited %d (recorded %d)\n", s.N, got, *s.Exit)
			exit = exitFailed
			break
		}
	}
	if f.stop {
		if err := stop(stopFlags{runDir: runDir, timeout: f.timeout}); err != nil {
			return failed("replay", err)
		}
	}
	return exit
}

// replayDir is --run-dir, or else the first of <bundle>-replay-1, -2 …
// that does not exist.
func replayDir(f replayFlags) (string, error) {
	if f.runDir != "" {
		return f.runDir, nil
	}
	base := strings.TrimRight(f.bundle, `/\`)
	for n := 1; ; n++ {
		dir := base + "-replay-" + strconv.Itoa(n)
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			return dir, nil
		} else if err != nil {
			return "", fmt.Errorf("replay dir: %w", err)
		}
	}
}

// replayTest runs the bundle's recorded test command from the root of the
// worktree that holds the current directory, against the bundle's binary
// and this fakeworld, which a browser test launches its world with. The
// test keeps its runs in the replay dir, since Playwright empties its
// output dir, which may hold the bundle, when it starts.
func replayTest(f replayFlags, b world.Bundle) int {
	argv, err := splitCommand(b.Origin.Command)
	if err != nil {
		return failed("replay", err)
	}
	ctx, stop := interrupted()
	defer stop()
	root, err := worktree.Root(ctx, ".")
	if err != nil {
		return failed("replay", err)
	}
	binary, err := filepath.Abs(b.Binary)
	if err != nil {
		return failed("replay", err)
	}
	variable, err := binaryVariable(binary)
	if err != nil {
		return failed("replay", err)
	}
	fakeworld, err := os.Executable()
	if err != nil {
		return failed("replay", err)
	}
	runs, err := replayDir(f)
	if err != nil {
		return failed("replay", err)
	}
	if runs, err = filepath.Abs(runs); err != nil {
		return failed("replay", err)
	}
	warnOnCommit(ctx, root, binary)
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), variable+"="+binary, "TEST_FAKEWORLD_BINARY="+fakeworld, "TEST_FAKEWORLD_RUNS="+runs)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = cmd.Run()
	if _, exited := errors.AsType[*exec.ExitError](err); err != nil && !exited {
		return failed("replay", err)
	}
	return cmd.ProcessState.ExitCode()
}

// binaryVariable is the test variable that names binary to the tests:
// the e2e binary's own for a build of cmd/slipstream-e2e.
func binaryVariable(binary string) (string, error) {
	info, err := gobuildinfo.ReadFile(binary)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", binary, err)
	}
	if strings.HasSuffix(info.Path, "/cmd/slipstream-e2e") {
		return "TEST_SLIPSTREAM_E2E_BINARY", nil
	}
	return "TEST_SLIPSTREAM_BINARY", nil
}

func warnOnCommit(ctx context.Context, root, binary string) {
	stamp, err := buildinfo.ReadFile(binary)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fakeworld replay: warning: %s: %v\n", binary, err)
		return
	}
	head, err := worktree.Stamp(ctx, root)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fakeworld replay: warning: %v\n", err)
		return
	}
	if head.Commit != stamp.Commit {
		_, _ = fmt.Fprintf(os.Stderr, "fakeworld replay: warning: the worktree's HEAD is %s, and the binary was built at %s\n", head.Commit, stamp.Commit)
	}
}

// splitCommand splits a recorded test command into words at spaces
// outside single quotes, the one quoting worldtest.Origin writes.
func splitCommand(s string) ([]string, error) {
	var words []string
	var word strings.Builder
	quoted, inWord := false, false
	for _, r := range s {
		switch {
		case r == '\'':
			quoted, inWord = !quoted, true
		case r == ' ' && !quoted:
			if inWord {
				words = append(words, word.String())
				word.Reset()
			}
			inWord = false
		default:
			word.WriteRune(r)
			inWord = true
		}
	}
	if quoted {
		return nil, fmt.Errorf("the recorded command %q has an unclosed quote", s)
	}
	if inWord {
		words = append(words, word.String())
	}
	if len(words) == 0 {
		return nil, errors.New("the bundle records no test command")
	}
	return words, nil
}
