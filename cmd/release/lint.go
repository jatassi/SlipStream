package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/icons"
	"github.com/jatassi/slipstream/cmd/release/internal/lint"
	_ "github.com/jatassi/slipstream/cmd/release/internal/lint/rules"
	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/cmd/release/internal/notices"
	"github.com/jatassi/slipstream/cmd/release/internal/tokens"
)

func init() {
	register("lint", "run gofmt, go vet and golangci-lint (-part target), and the architecture lint, the cgo-free openapi.json check, the sqlc drift check, the proto check, the web generated check, the tokens generated check, the contrast proof, the icons generated check, the web checks and the notices check (-part repo); -part all runs both", runLint)
	register("fmt", "format the Go code with gofmt and the web app with Oxfmt", runFmt)
}

type lintCheck struct {
	name string
	run  func() error
}

func runLint(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	only := fs.String("rule", "", "run only this architecture-lint rule")
	part := fs.String("part", "all", "target runs the checks each OS must prove: gofmt, go vet and golangci-lint; repo runs the rest, which no OS changes; all runs both")
	fs.Usage = func() {
		out := fs.Output()
		_, _ = fmt.Fprintln(out, "usage: go run ./cmd/release lint [-part all|target|repo] [-rule name]")
		for _, r := range lint.Rules() {
			_, _ = fmt.Fprintf(out, "  %-16s %s\n", r.Name, r.Doc)
		}
	}
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if *part != "all" && *part != "target" && *part != "repo" {
		return fmt.Errorf("-part %q is not one of all, target, repo", *part)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	if *only != "" {
		r, ok := lint.Lookup(*only)
		if !ok {
			fs.Usage()
			return fmt.Errorf("unknown rule %q", *only)
		}
		return lint.Run(os.Stdout, lint.Open(ctx, root), []lint.Rule{r})
	}
	var checks []lintCheck
	if *part != "repo" {
		target, err := targetChecks(ctx, root)
		if err != nil {
			return err
		}
		checks = append(checks, target...)
	}
	if *part != "target" {
		repo, err := repoChecks(ctx, root)
		if err != nil {
			return err
		}
		checks = append(checks, repo...)
	}
	var failed []string
	for _, c := range checks {
		if err := c.run(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.name, err)
			failed = append(failed, c.name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed: %s", strings.Join(failed, ", "))
	}
	return nil
}

// targetChecks prove what each OS changes: line endings, and the cgo
// packages that only type-check against the target's bundle.
func targetChecks(ctx context.Context, root string) ([]lintCheck, error) {
	_, _, bundle, err := goTargetBundle(ctx)
	if err != nil {
		return nil, err
	}
	goEnv := bundleEnv(bundle)
	return []lintCheck{
		{"gofmt", func() error { return golangciLint(ctx, root, nil, "fmt", "--diff") }},
		{"go vet", func() error { return run(ctx, root, goEnv, "go", "vet", "./...") }},
		{"golangci-lint", func() error { return golangciLint(ctx, root, goEnv, "run", "./...") }},
	}, nil
}

func repoChecks(ctx context.Context, root string) ([]lintCheck, error) {
	// The web build type-checks the app and installs the packages that the
	// web checks run and the notices check reads.
	if err := buildWeb(ctx, root); err != nil {
		return nil, err
	}
	// The document must build without libtorrent, so a web-only change
	// never needs the native toolchain.
	cgoFree := append(bundleEnv(native.Bundle{}), "CGO_ENABLED=0")
	web := filepath.Join(root, "web")
	return []lintCheck{
		{"architecture lint", func() error { return lint.Run(os.Stdout, lint.Open(ctx, root), lint.Rules()) }},
		{"openapi.json without cgo", func() error {
			return run(ctx, root, cgoFree, "go", "test", "-count=1", "-run", "^TestDocument$", "./internal/node/surface")
		}},
		{"sqlc drift", func() error {
			configs, err := sqlcConfigs(ctx, root)
			if err != nil {
				return err
			}
			return sqlcDrift(ctx, root, root, configs, os.Stdout)
		}},
		{"proto", func() error { return protoCheck(ctx, root, os.Stdout) }},
		{"web generated", func() error { return webGenerated(ctx, root, os.Stdout) }},
		{"tokens generated", func() error { return tokens.Check(root) }},
		{"contrast", func() error { return tokens.CheckContrast(root) }},
		{"icons generated", func() error { return icons.Check(root) }},
		{"oxlint", func() error { return run(ctx, web, nil, "bun", "run", "lint") }},
		{"oxfmt", func() error { return run(ctx, web, nil, "bun", "run", "fmt:check") }},
		{"notices", func() error { return notices.Check(ctx, root) }},
	}, nil
}

func runFmt(ctx context.Context, args []string) error {
	if err := flag.NewFlagSet("fmt", flag.ContinueOnError).Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := golangciLint(ctx, root, nil, "fmt"); err != nil {
		return err
	}
	if err := installWeb(ctx, root); err != nil {
		return err
	}
	return run(ctx, filepath.Join(root, "web"), nil, "bun", "run", "fmt")
}

// golangciLint runs the golangci-lint that tools/go.mod pins, so the
// linter's dependencies never enter the product's module graph.
func golangciLint(ctx context.Context, root string, env []string, args ...string) error {
	return run(ctx, root, env, "go", append([]string{"tool", "-modfile=tools/go.mod", "golangci-lint"}, args...)...)
}
