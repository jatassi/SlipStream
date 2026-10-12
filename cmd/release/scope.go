package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/changescope"
	"github.com/jatassi/slipstream/cmd/release/internal/nativetouch"
)

func init() {
	register("scope", "print how much CI runs for the changes from base to head: docs, pr or native", runScope)
}

func runScope(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("scope", flag.ContinueOnError)
	explain := fs.Bool("explain", false, "print which native clause matched which file, and for scope pr which files are not inert")
	base := fs.String("base", "origin/main", "revision the change merges into")
	head := fs.String("head", "HEAD", "revision of the change")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	changes, err := gitChanges(ctx, ".", *base, *head)
	if err != nil {
		return err
	}
	scope := changescope.Of(changes)
	if *explain {
		if err := explainScope(os.Stdout, changes, scope); err != nil {
			return err
		}
	}
	fmt.Println(scope)
	return githubOutput("scope", scope.String())
}

func explainScope(w io.Writer, changes []nativetouch.Change, scope changescope.Scope) error {
	names := make([]string, len(nativetouch.Clauses))
	for i, c := range nativetouch.Clauses {
		names[i] = c.Name
	}
	var b strings.Builder
	fmt.Fprintf(&b, "clauses: %s\n", strings.Join(names, ", "))
	for _, m := range nativetouch.Matches(changes) {
		fmt.Fprintf(&b, "%s: %s\n", m.Clause, m.Path)
	}
	if scope == changescope.PR {
		for _, p := range changescope.NotInert(changes) {
			fmt.Fprintf(&b, "not inert: %s\n", p)
		}
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("print the explanation: %w", err)
	}
	return nil
}

// githubOutput appends key=value to the step outputs file when the command
// runs as a GitHub Actions step.
func githubOutput(key, value string) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open the step outputs file: %w", err)
	}
	_, err = fmt.Fprintf(f, "%s=%s\n", key, value)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write the step output %s: %w", key, err)
	}
	return nil
}

// gitChanges lists the files changed between the merge base of base and head
// and head, with each side's content read at its own revision.
func gitChanges(ctx context.Context, dir, base, head string) ([]nativetouch.Change, error) {
	mergeBase, err := gitOutput(ctx, dir, nil, "merge-base", base, head)
	if err != nil {
		return nil, err
	}
	oldRev := strings.TrimSpace(string(mergeBase))
	diff, err := gitOutput(ctx, dir, nil, "diff", "--name-status", "-z", "-M", oldRev, head)
	if err != nil {
		return nil, err
	}
	changes, err := parseNameStatus(diff)
	if err != nil || len(changes) == 0 {
		return changes, err
	}
	var specs []string
	var sides []*nativetouch.Side
	addSide := func(rev string, side *nativetouch.Side) {
		if side.Path != "" {
			specs = append(specs, rev+":"+side.Path)
			sides = append(sides, side)
		}
	}
	for i := range changes {
		addSide(oldRev, &changes[i].Old)
		addSide(head, &changes[i].New)
	}
	contents, err := catFiles(ctx, dir, specs)
	if err != nil {
		return nil, err
	}
	for i, side := range sides {
		side.Content = contents[i]
	}
	return changes, nil
}

// parseNameStatus parses git diff --name-status -z: a status, then one path,
// or two for a rename (old, then new).
func parseNameStatus(out []byte) ([]nativetouch.Change, error) {
	if len(out) == 0 {
		return nil, nil
	}
	fields := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	var changes []nativetouch.Change
	for i := 0; i < len(fields); {
		status := fields[i]
		switch status[0] {
		case 'A':
			changes = append(changes, nativetouch.Change{New: nativetouch.Side{Path: fields[i+1]}})
			i += 2
		case 'D':
			changes = append(changes, nativetouch.Change{Old: nativetouch.Side{Path: fields[i+1]}})
			i += 2
		case 'M', 'T':
			changes = append(changes, nativetouch.Change{Old: nativetouch.Side{Path: fields[i+1]}, New: nativetouch.Side{Path: fields[i+1]}})
			i += 2
		case 'R':
			changes = append(changes, nativetouch.Change{Old: nativetouch.Side{Path: fields[i+1]}, New: nativetouch.Side{Path: fields[i+2]}})
			i += 3
		default:
			return nil, fmt.Errorf("git diff: unexpected status %q", status)
		}
	}
	return changes, nil
}

func catFiles(ctx context.Context, dir string, specs []string) ([][]byte, error) {
	out, err := gitOutput(ctx, dir, strings.NewReader(strings.Join(specs, "\n")+"\n"), "cat-file", "--batch")
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(bytes.NewReader(out))
	contents := make([][]byte, len(specs))
	for i, spec := range specs {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("git cat-file %s: %w", spec, err)
		}
		parts := strings.Fields(header)
		if len(parts) != 3 {
			return nil, fmt.Errorf("git cat-file %s: %q", spec, strings.TrimSpace(header))
		}
		size, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("git cat-file %s: %w", spec, err)
		}
		contents[i] = make([]byte, size)
		if _, err := io.ReadFull(r, contents[i]); err != nil {
			return nil, fmt.Errorf("git cat-file %s: %w", spec, err)
		}
		if _, err := r.Discard(1); err != nil {
			return nil, fmt.Errorf("git cat-file %s: %w", spec, err)
		}
	}
	return contents, nil
}

func gitOutput(ctx context.Context, dir string, stdin io.Reader, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdin = stdin
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
