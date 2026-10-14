package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/world"
)

const captureUsage = `  <capture> list <run-dir> [--node n] [--where k=v]... [--after seq]
      print the capture's records
  <capture> wait <run-dir> [--node n] [--where k=v]... [--after seq] [--timeout 2m]
      wait for a record of the capture after the last step's start
  <capture> open <run-dir> <seq> [--proof <feature>/<sub>]
      copy the record's files into the proof dir and print their paths
  captures: `

// withCaptures adds one reader verb for each capture the world's fakes
// declare to the static verbs.
func withCaptures(static map[string]func(args []string) int) map[string]func(args []string) int {
	for _, c := range world.Captures() {
		if _, taken := static[c.Name]; taken {
			panic(fmt.Sprintf("fakeworld: capture %s shadows a verb or another capture", c.Name))
		}
		static[c.Name] = captureCommand(c)
	}
	return static
}

func captureNames() string {
	var names []string
	for _, c := range world.Captures() {
		names = append(names, c.Name)
	}
	return strings.Join(names, ", ")
}

func captureCommand(c fake.CaptureSpec) func(args []string) int {
	verbs := map[string]func(c fake.CaptureSpec, runDir string, args []string) int{
		"list": captureList,
		"wait": captureWait,
		"open": captureOpen,
	}
	return func(args []string) int {
		if len(args) < 2 {
			return usage(fmt.Errorf("%s: takes list, wait or open and a run dir", c.Name))
		}
		verb, ok := verbs[args[0]]
		if !ok {
			return usage(fmt.Errorf("%s: unknown command %q", c.Name, args[0]))
		}
		return verb(c, args[1], args[2:])
	}
}

func parseCaptureQuery(c fake.CaptureSpec, verb string, args []string, timeout *time.Duration) (events.Query, error) {
	fs := newFlagSet(c.Name + " " + verb)
	q := events.Query{Kind: c.Kind, Match: params{}}
	fs.StringVar(&q.Node, "node", "", "")
	fs.Var(params(q.Match), "where", "")
	fs.Uint64Var(&q.After, "after", 0, "")
	if timeout != nil {
		fs.DurationVar(timeout, "timeout", defaultTimeout, "")
	}
	if err := noArguments(fs, args); err != nil {
		return events.Query{}, err
	}
	if len(q.Match) == 0 {
		q.Match = nil
	}
	if timeout != nil {
		return q, positiveTimeout(fs, *timeout)
	}
	return q, nil
}

func captureList(c fake.CaptureSpec, runDir string, args []string) int {
	q, err := parseCaptureQuery(c, "list", args, nil)
	if err != nil {
		return usage(err)
	}
	verb := c.Name + " list"
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, verb, args, func(context.Context) int {
		recs, err := events.Select(runDir, q)
		if err != nil {
			return failed(verb, err)
		}
		for _, r := range recs {
			printJSON(r)
		}
		return exitOK
	})
}

// captureWait waits for a record of the capture. Unless --after says
// otherwise, it waits for one after the last step's start.
func captureWait(c fake.CaptureSpec, runDir string, args []string) int {
	var timeout time.Duration
	q, err := parseCaptureQuery(c, "wait", args, &timeout)
	if err != nil {
		return usage(err)
	}
	verb := c.Name + " wait"
	if q.After == 0 {
		if q.After, err = lastStep(runDir); err != nil {
			return failed(verb, err)
		}
	}
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, verb, args, func(ctx context.Context) int {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		rec, err := events.Wait(ctx, runDir, q)
		if err == nil {
			printJSON(rec)
			return exitOK
		}
		_, _ = fmt.Fprintf(os.Stderr, "fakeworld %s: %v\n", verb, err)
		recs, serr := events.Select(runDir, events.Query{Kind: c.Kind})
		if serr != nil {
			return failed(verb, serr)
		}
		if len(recs) > 0 {
			_, _ = fmt.Fprintf(os.Stderr, "the last %s records:\n", c.Name)
		}
		for _, r := range recs[max(0, len(recs)-5):] {
			line, _ := json.Marshal(r)
			_, _ = fmt.Fprintln(os.Stderr, string(line))
		}
		return exitFailed
	})
}

// captureOpen prints the paths of the files of the record at seq, copied
// into the proof dir when --proof names one.
func captureOpen(c fake.CaptureSpec, runDir string, args []string) int {
	verb := c.Name + " open"
	fs := newFlagSet(verb)
	var proof string
	fs.StringVar(&proof, "proof", "", "")
	positional, err := parse(fs, args)
	if err != nil {
		return usage(err)
	}
	if len(positional) != 1 {
		return usage(fmt.Errorf("%s: takes a run dir and a seq", verb))
	}
	seq, err := strconv.ParseUint(positional[0], 10, 64)
	if err != nil {
		return usage(fmt.Errorf("%s: seq %q is not a number", verb, positional[0]))
	}
	if proof != "" && !proofShape.MatchString(proof) {
		return usage(fmt.Errorf("%s: --proof %q is not <feature>/<sub>", verb, proof))
	}
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, verb, args, func(context.Context) int {
		recs, err := events.Select(runDir, events.Query{Kind: c.Kind, After: seq - 1})
		if err != nil {
			return failed(verb, err)
		}
		if len(recs) == 0 || recs[0].Seq != seq {
			return failed(verb, fmt.Errorf("seq %d is no %s record", seq, c.Kind))
		}
		paths, err := openFiles(runDir, proof, recs[0])
		if err != nil {
			return failed(verb, err)
		}
		for _, p := range paths {
			fmt.Println(p)
		}
		return exitOK
	})
}

func openFiles(runDir, proof string, r events.Record) ([]string, error) {
	var paths []string
	for _, name := range slices.Sorted(maps.Keys(r.Files)) {
		rel := r.Files[name]
		src := filepath.Join(runDir, filepath.FromSlash(rel))
		if proof == "" {
			paths = append(paths, src)
			continue
		}
		dst := filepath.Join(runDir, "proof", filepath.FromSlash(proof), fmt.Sprintf("%d-%s", r.Seq, path.Base(rel)))
		if err := copyFile(src, dst); err != nil {
			return nil, err
		}
		paths = append(paths, dst)
	}
	return paths, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open capture file: %w", err)
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("proof dir: %w", err)
	}
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("proof file: %w", err)
	}
	_, err = io.Copy(out, in)
	return errors.Join(err, out.Close())
}
