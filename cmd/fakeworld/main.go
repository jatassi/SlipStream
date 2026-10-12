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

var commands = map[string]func(args []string) int{
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
	_, _ = fmt.Fprintf(os.Stderr, "fakeworld: %v\n%s\n", err, usageText)
	return exitUsage
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// parse parses flags anywhere among args and returns the other arguments.
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
	timeout        time.Duration
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
	binary, scenario, runDir string
}

func parseSupervise(args []string) (superviseFlags, error) {
	fs := newFlagSet("supervise")
	var f superviseFlags
	fs.StringVar(&f.binary, "binary", "", "")
	fs.StringVar(&f.scenario, "scenario", "", "")
	fs.StringVar(&f.runDir, "run-dir", "", "")
	if err := noArguments(fs, args); err != nil {
		return superviseFlags{}, err
	}
	return f, required(fs, "binary", "scenario", "run-dir")
}

// interrupted ends on Ctrl+C or SIGTERM. Until stop is called, those
// signals no longer end the process.
func interrupted() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
