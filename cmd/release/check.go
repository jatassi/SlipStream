package main

import (
	"context"
	"flag"
	"fmt"
)

func init() {
	register("web", "install and build the web app into web/dist", runWeb)
	register("check", "run lint, build and test, as CI does", runCheck)
}

func runWeb(ctx context.Context, args []string) error {
	if err := flag.NewFlagSet("web", flag.ContinueOnError).Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	return buildWeb(ctx, root)
}

func runCheck(ctx context.Context, args []string) error {
	if err := flag.NewFlagSet("check", flag.ContinueOnError).Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	for _, step := range []struct {
		name string
		run  func(context.Context, []string) error
	}{{"lint", runLint}, {"build", runBuild}, {"test", runTest}} {
		if err := step.run(ctx, nil); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}
	return nil
}
