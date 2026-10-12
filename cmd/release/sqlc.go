package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"path"
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
		if err := sqlc(ctx, root, "generate", "-f", c); err != nil {
			return err
		}
	}
	return nil
}

// sqlcConfigs is every tracked sqlc.yaml outside the dirs the go command
// ignores (testdata, _*, .*), so lint fixtures are never regenerated.
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

// sqlc runs the sqlc that tools/go.mod pins with the host environment: it
// builds without cgo, so it needs none of the native bundle's toolchain.
func sqlc(ctx context.Context, root string, args ...string) error {
	return run(ctx, root, nil, "go", append([]string{"tool", "-modfile=tools/go.mod", "sqlc"}, args...)...)
}
