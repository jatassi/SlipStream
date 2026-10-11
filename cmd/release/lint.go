package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
	_ "github.com/jatassi/slipstream/cmd/release/internal/lint/rules"
)

func init() {
	register("lint", "run the architecture-lint rules", runLint)
}

func runLint(args []string) error {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	only := fs.String("rule", "", "run only this rule")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: go run ./cmd/release lint [-rule name]")
		for _, r := range lint.Rules() {
			fmt.Fprintf(fs.Output(), "  %-16s %s\n", r.Name, r.Doc)
		}
	}
	if err := fs.Parse(args); err != nil {
		return err
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
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return fmt.Errorf("git rev-parse: %w", err)
	}
	return lint.Run(os.Stdout, lint.Open(strings.TrimSpace(string(top))), rules)
}
