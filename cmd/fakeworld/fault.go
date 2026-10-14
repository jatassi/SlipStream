package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world"
)

var faultCommands = map[string]func(runDir string, args []string) int{
	"set":   faultSet,
	"clear": faultClear,
	"list":  faultList,
	"wait":  faultWait,
}

func faultCommand(args []string) int {
	if len(args) < 2 {
		return usage(errors.New("fault: takes set, clear, list or wait and a run dir"))
	}
	command, ok := faultCommands[args[0]]
	if !ok {
		return usage(fmt.Errorf("fault: unknown command %q", args[0]))
	}
	return command(args[1], args[2:])
}

type params map[string]string

func (p params) String() string { return "" }

func (p params) Set(v string) error {
	k, val, ok := strings.Cut(v, "=")
	if !ok || k == "" {
		return fmt.Errorf("%q is not k=v", v)
	}
	p[k] = val
	return nil
}

// parseRule reads fault set's arguments after the run dir:
// <id> <target>[@<node>] <fault> [--route r] [--param k=v]… [--skip n]
// [--times n] [--after <kind>[,k=v…]].
func parseRule(args []string) (fault.Rule, error) {
	fs := newFlagSet("fault set")
	r := fault.Rule{Params: params{}}
	var after string
	fs.StringVar(&r.Route, "route", "", "")
	fs.Var(params(r.Params), "param", "")
	fs.IntVar(&r.Skip, "skip", 0, "")
	fs.IntVar(&r.Times, "times", 0, "")
	fs.StringVar(&after, "after", "", "")
	positional, err := parse(fs, args)
	if err != nil {
		return fault.Rule{}, err
	}
	if len(positional) != 3 {
		return fault.Rule{}, errors.New("fault set: takes a run dir, an id, a target[@node] and a fault")
	}
	r.ID = positional[0]
	target, origin, _ := strings.Cut(positional[1], "@")
	r.Target, r.Origin = fault.Target(target), origin
	if r.Fault, err = fault.Parse(positional[2]); err != nil {
		return fault.Rule{}, fmt.Errorf("fault set: %w", err)
	}
	if len(r.Params) == 0 {
		r.Params = nil
	}
	if after != "" {
		q, err := parseAfter(after)
		if err != nil {
			return fault.Rule{}, err
		}
		r.After = &q
	}
	return r, nil
}

func parseAfter(s string) (events.Query, error) {
	parts := strings.Split(s, ",")
	q := events.Query{Kind: parts[0]}
	if q.Kind == "" {
		return events.Query{}, fmt.Errorf("fault set: --after %q names no kind", s)
	}
	for _, p := range parts[1:] {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return events.Query{}, fmt.Errorf("fault set: --after %q: %q is not k=v", s, p)
		}
		if q.Match == nil {
			q.Match = map[string]string{}
		}
		q.Match[k] = v
	}
	return q, nil
}

func faultSet(runDir string, args []string) int {
	r, err := parseRule(args)
	if err != nil {
		return usage(err)
	}
	ctx, stop := interrupted()
	defer stop()
	return journal(ctx, runDir, "fault set", args, func(ctx context.Context, c world.Controls, _ world.Step) (int, map[string]string) {
		return printed("fault set", func() (any, error) { return c.SetFault(ctx, r) }), nil
	})
}

func faultClear(runDir string, args []string) int {
	if len(args) > 1 {
		return usage(errors.New("fault clear: takes a run dir and at most one id"))
	}
	id := ""
	if len(args) == 1 {
		id = args[0]
	}
	ctx, stop := interrupted()
	defer stop()
	return journal(ctx, runDir, "fault clear", args, func(ctx context.Context, c world.Controls, _ world.Step) (int, map[string]string) {
		return printed("fault clear", func() (any, error) { return c.ClearFault(ctx, id) }), nil
	})
}

func faultList(runDir string, args []string) int {
	if len(args) > 0 {
		return usage(errors.New("fault list: takes only a run dir"))
	}
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, "fault list", args, func(context.Context) int {
		live, err := world.LiveFaults(runDir)
		if err != nil {
			return failed("fault list", err)
		}
		for _, rec := range live {
			printJSON(rec)
		}
		return exitOK
	})
}

type waitFlags struct {
	q       events.Query
	timeout time.Duration
}

func parseFaultWait(args []string) (waitFlags, error) {
	fs := newFlagSet("fault wait")
	f := waitFlags{q: events.Query{Kind: events.FaultFired}}
	var rule string
	fs.StringVar(&rule, "rule", "", "")
	fs.StringVar(&f.q.Node, "node", "", "")
	fs.Uint64Var(&f.q.After, "after", 0, "")
	fs.DurationVar(&f.timeout, "timeout", defaultTimeout, "")
	if err := noArguments(fs, args); err != nil {
		return waitFlags{}, err
	}
	if rule != "" {
		f.q.Match = map[string]string{"rule": rule}
	}
	return f, positiveTimeout(fs, f.timeout)
}

// faultWait waits for a fault.fired record. Unless --after says
// otherwise, it waits for one after the last step's start.
func faultWait(runDir string, args []string) int {
	f, err := parseFaultWait(args)
	if err != nil {
		return usage(err)
	}
	if f.q.After == 0 {
		if f.q.After, err = lastStep(runDir); err != nil {
			return failed("fault wait", err)
		}
	}
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, "fault wait", args, func(ctx context.Context) int {
		ctx, cancel := context.WithTimeout(ctx, f.timeout)
		defer cancel()
		rec, err := events.Wait(ctx, runDir, f.q)
		if err == nil {
			printJSON(rec)
			return exitOK
		}
		_, _ = fmt.Fprintf(os.Stderr, "fakeworld fault wait: %v\n", err)
		fired, serr := events.Select(runDir, events.Query{Kind: events.FaultFired})
		if serr != nil {
			return failed("fault wait", serr)
		}
		if len(fired) > 0 {
			_, _ = fmt.Fprintln(os.Stderr, "the last fault.fired records:")
		}
		for _, r := range fired[max(0, len(fired)-5):] {
			line, _ := json.Marshal(r)
			_, _ = fmt.Fprintln(os.Stderr, string(line))
		}
		return exitFailed
	})
}

// lastStep is the seq of the run's last step.start, or 0 before any step.
func lastStep(runDir string) (uint64, error) {
	starts, err := events.Select(runDir, events.Query{Kind: events.StepStart})
	if err != nil || len(starts) == 0 {
		return 0, err
	}
	return starts[len(starts)-1].Seq, nil
}

func printed(verb string, call func() (any, error)) int {
	v, err := call()
	if err != nil {
		return failed(verb, err)
	}
	printJSON(v)
	return exitOK
}

func printJSON(v any) {
	fmt.Print(string(jsonLine(v)))
}

func jsonLine(v any) []byte {
	line, err := json.Marshal(v)
	if err != nil {
		panic("fakeworld: " + err.Error())
	}
	return append(line, '\n')
}
