// Command fakeworld starts and stops Fake worlds from a shell: a thin
// shell over fakes/world, whose supervisor process holds each run.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
)

const usageText = `usage: fakeworld <command>
  start --binary <path> --scenario <name> --run-dir <dir> [--timeout 2m]
      launch the catalog Scenario against the binary and print the manifest's
      path once the world is ready
  stop <run-dir> [--timeout 2m]
      stop the run and wait until every process of it has exited
  exec <run-dir> [--proof <feature>/<sub>] -- <command> [<arg>...]
      run the command as a journaled step, its {path} tokens filled from the
      manifest ({{ is a literal brace), and exit with its exit code
  fault set <run-dir> <id> <target>[@<node>] <fault> [--route r] [--param k=v]...
            [--skip n] [--times n] [--after <kind>[,k=v...]]
      set a fault rule, replacing the one with its id; <fault> is one of
      refuse | reset | stall | latency=200ms | throttle=256KiB/s | cut=64KiB |
      tls=untrusted|expired|wrong-name | status=<code>[,retry-after=<dur>]
      [,reset-header=<name>] | hang | malformed | truncate | slow-body=<rate>
  fault clear <run-dir> [<id>]
      clear the rule with id, or every rule
  fault list <run-dir>
      print the fault.set record of each rule still set
  fault wait <run-dir> [--rule id] [--node n] [--after seq] [--timeout 2m]
      wait for a fault.fired record after the last step's start
  cue <run-dir> <name> [<args>]
      play a cue a fake declares, its arguments one JSON object
  node stop|kill|start|wipe <run-dir> <node>
      stop the Node with slipstream stop, kill its process, start it again
      on the same config.yaml, or empty a stopped Node's data dir but its
      config.yaml, and print its manifest entry
  fs stat <run-dir> <path>
      print the file's path, size, link count and BLAKE3
  fs match <run-dir> <path> --release <name>
      exit 0 when the file's size and BLAKE3 are the release's; a path
      outside the run dir and its scratch dir is refused
  doctor <run-dir> [--json] [--worktree <dir>] [--timeout 2m]
      run six read-only checks on the live run: it is ours, its build is the
      worktree's, its cluster is its Scenario's, every fake answers, every
      address an Admin entered is a fake's front, and its API key works;
      exit 1 when any fails, and list the fault rules still set
  summary <run-dir>
      write the run's summary.md, text for a PR's Verify section, from its
      files alone, and print its path
  replay <bundle> [--binary <path>] [--run-dir <dir>] [--to N] [--stop]
         [--timeout 2m]
      check the bundle's scenario.yaml and binary against its manifest, then
      for a run fakeworld start launched, start a fresh world from its
      Scenario and run its steps 1 to N again, stopping at the first whose
      exit differs from the one recorded; for a run a Go test launched, run
      the test again against the binary
  setup-loopback [--remove]
      on macOS, alias 127.0.0.2 to 127.0.0.32 now and at every boot (needs root)`

const (
	exitOK     = 0
	exitFailed = 1
	exitUsage  = 3

	defaultTimeout = 2 * time.Minute
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// verbs are fakeworld's static commands, before the capture readers.
var verbs = map[string]func(args []string) int{
	"start": func(args []string) int {
		f, err := parseStart(args)
		if err != nil {
			return usage(err)
		}
		return failed("start", start(f))
	},
	"stop": func(args []string) int {
		f, err := parseStop(args)
		if err != nil {
			return usage(err)
		}
		return failed("stop", stop(f))
	},
	"exec": func(args []string) int {
		f, err := parseExec(args)
		if err != nil {
			return usage(err)
		}
		return execStep(f)
	},
	"fault":   faultCommand,
	"cue":     cueCommand,
	"node":    nodeCommand,
	"fs":      fsCommand,
	"sse":     captureCommand(sseReader),
	"doctor":  doctor,
	"summary": summary,
	"replay":  replay,
	"setup-loopback": func(args []string) int {
		remove, err := parseSetupLoopback(args)
		if err != nil {
			return usage(err)
		}
		return failed("setup-loopback", setupLoopback(remove))
	},
	"supervise": func(args []string) int {
		f, err := parseSupervise(args)
		if err != nil {
			return usage(err)
		}
		return supervise(f)
	},
}

// commands are set in init, since replay dispatches through them.
var commands map[string]func(args []string) int

func init() { commands = withCaptures(verbs) }

func run(args []string) int {
	if len(args) == 0 {
		return usage(errors.New("no command"))
	}
	command, ok := commands[args[0]]
	if !ok {
		return usage(fmt.Errorf("unknown command %q", args[0]))
	}
	return command(args[1:])
}

func failed(name string, err error) int {
	if err == nil {
		return exitOK
	}
	_, _ = fmt.Fprintf(os.Stderr, "fakeworld %s: %v\n", name, err)
	return exitFailed
}

func usage(err error) int {
	_, _ = fmt.Fprintf(os.Stderr, "fakeworld: %v\n%s\n%s%s\n", err, usageText, captureUsage, captureNames())
	return exitUsage
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, fmt.Errorf("%s: %w", fs.Name(), err)
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func noArguments(fs *flag.FlagSet, args []string) error {
	positional, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(positional) > 0 {
		return fmt.Errorf("%s: unexpected argument %q", fs.Name(), positional[0])
	}
	return nil
}

func positiveTimeout(fs *flag.FlagSet, d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("%s: --timeout %s is not positive", fs.Name(), d)
	}
	return nil
}

func required(fs *flag.FlagSet, names ...string) error {
	for _, name := range names {
		if fs.Lookup(name).Value.String() == "" {
			return fmt.Errorf("%s: --%s is required", fs.Name(), name)
		}
	}
	return nil
}

type startFlags struct {
	binary, runDir string
	scenario       scenario.Scenario
	// scenarioFile holds the Scenario when it is not the catalog's, as
	// for a replay.
	scenarioFile string
	timeout      time.Duration
}

func parseStart(args []string) (startFlags, error) {
	fs := newFlagSet("start")
	var f startFlags
	var name string
	fs.StringVar(&f.binary, "binary", "", "")
	fs.StringVar(&name, "scenario", "", "")
	fs.StringVar(&f.runDir, "run-dir", "", "")
	fs.DurationVar(&f.timeout, "timeout", defaultTimeout, "")
	if err := noArguments(fs, args); err != nil {
		return startFlags{}, err
	}
	if err := required(fs, "binary", "scenario", "run-dir"); err != nil {
		return startFlags{}, err
	}
	s, err := scenario.Load(name)
	if err != nil {
		return startFlags{}, fmt.Errorf("start: %w", err)
	}
	f.scenario = s
	return f, positiveTimeout(fs, f.timeout)
}

type stopFlags struct {
	runDir  string
	timeout time.Duration
}

func parseStop(args []string) (stopFlags, error) {
	fs := newFlagSet("stop")
	var f stopFlags
	fs.DurationVar(&f.timeout, "timeout", defaultTimeout, "")
	positional, err := parse(fs, args)
	if err != nil {
		return stopFlags{}, err
	}
	if len(positional) != 1 {
		return stopFlags{}, errors.New("stop: takes one run dir")
	}
	f.runDir = positional[0]
	return f, positiveTimeout(fs, f.timeout)
}

func parseSetupLoopback(args []string) (bool, error) {
	fs := newFlagSet("setup-loopback")
	remove := fs.Bool("remove", false, "")
	return *remove, noArguments(fs, args)
}

type superviseFlags struct {
	binary, scenario, scenarioFile, runDir string
}

func parseSupervise(args []string) (superviseFlags, error) {
	fs := newFlagSet("supervise")
	var f superviseFlags
	fs.StringVar(&f.binary, "binary", "", "")
	fs.StringVar(&f.scenario, "scenario", "", "")
	fs.StringVar(&f.scenarioFile, "scenario-file", "", "")
	fs.StringVar(&f.runDir, "run-dir", "", "")
	if err := noArguments(fs, args); err != nil {
		return superviseFlags{}, err
	}
	return f, required(fs, "binary", "scenario", "run-dir")
}

func interrupted() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
