package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func init() {
	register("sqlc", "regenerate every table owner's queries with the sqlc that tools/go.mod pins", runSqlc)
}

func runSqlc(ctx context.Context, args []string) error {
	if err := flag.NewFlagSet("sqlc", flag.ContinueOnError).Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	configs, err := sqlcConfigs(ctx, root)
	if err != nil {
		return err
	}
	for _, c := range configs {
		cmd := hostSqlc(ctx, root, "generate", "-f", c)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("sqlc generate -f %s: %w", c, err)
		}
	}
	return nil
}

func sqlcConfigs(ctx context.Context, root string) ([]string, error) {
	out, err := gitOutput(ctx, root, nil, "ls-files", "-z", "--", "sqlc.yaml", "*/sqlc.yaml")
	if err != nil {
		return nil, err
	}
	var configs []string
	for f := range bytes.SplitSeq(bytes.TrimRight(out, "\x00"), []byte{0}) {
		if len(f) > 0 && !ignoredByGo(string(f)) {
			configs = append(configs, string(f))
		}
	}
	return configs, nil
}

func ignoredByGo(file string) bool {
	for elem := range strings.SplitSeq(path.Dir(file), "/") {
		if elem == "testdata" || strings.HasPrefix(elem, "_") || strings.HasPrefix(elem, ".") {
			return true
		}
	}
	return false
}

func hostSqlc(ctx context.Context, root string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "go", append([]string{"tool", "-modfile=tools/go.mod", "sqlc"}, args...)...)
	cmd.Dir = root
	cmd.Env = os.Environ()
	return cmd
}

func sqlcDrift(ctx context.Context, root, dir string, configs []string, w io.Writer) error {
	var stale []string
	for _, c := range configs {
		out, err := hostSqlc(ctx, root, "diff", "-f", filepath.Join(dir, filepath.FromSlash(c))).CombinedOutput()
		if err == nil {
			continue
		}
		// sqlc exits 1 both for a diff and for a broken config or query.
		if exit := (*exec.ExitError)(nil); !errors.As(err, &exit) || !bytes.HasPrefix(out, []byte("--- a/")) {
			return fmt.Errorf("sqlc diff -f %s: %w\n%s", c, err, out)
		}
		owner := path.Dir(c)
		_, _ = fmt.Fprintf(w, "%s: generated code is stale; run go run ./cmd/release sqlc\n%s", owner, out)
		stale = append(stale, owner)
	}
	if len(stale) > 0 {
		return fmt.Errorf("stale: %s", strings.Join(stale, ", "))
	}
	return nil
}
