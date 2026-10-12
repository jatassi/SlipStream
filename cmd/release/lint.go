package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
	_ "github.com/jatassi/slipstream/cmd/release/internal/lint/rules"
	"github.com/jatassi/slipstream/cmd/release/internal/notices"
)

func init() {
	register("lint", "run the architecture-lint rules", runLint)
}

func runLint(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	only := fs.String("rule", "", "run only this rule")
	fs.Usage = func() {
		out := fs.Output()
		_, _ = fmt.Fprintln(out, "usage: go run ./cmd/release lint [-rule name]")
		for _, r := range lint.Rules() {
			_, _ = fmt.Fprintf(out, "  %-16s %s\n", r.Name, r.Doc)
		}
	}
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	rules := lint.Rules()
	if *only != "" {
		r, ok := lint.Lookup(*only)
		if !ok {
			fs.Usage()
			return fmt.Errorf("unknown rule %q", *only)
		}
		rules = []lint.Rule{r}
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := buildWeb(ctx, root); err != nil {
		return err
	}
	if err := lint.Run(os.Stdout, lint.Open(ctx, root), rules); err != nil || *only != "" {
		return err
	}
	return notices.Check(ctx, root)
}
