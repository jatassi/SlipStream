package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func init() {
	register("web", "install and build the web app into web/dist", runWeb)
	register("test", "run go test -race -shuffle=on against the native layers (default ./...)", runTest)
	register("check", "run lint, test and build, as CI does", runCheck)
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

// runTest passes its arguments to go test, so `release test -run Foo
// ./internal/media` narrows the run.
func runTest(ctx context.Context, args []string) error {
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	_, _, bundle, err := goTargetBundle(ctx)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		args = []string{"./..."}
	}
	gotest := exec.CommandContext(ctx, "go", append([]string{"test", "-race", "-shuffle=on"}, args...)...)
	gotest.Dir = root
	gotest.Stdout, gotest.Stderr = os.Stdout, os.Stderr
	gotest.Env = bundleEnv(bundle)
	if err := gotest.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}
	return nil
}

func runCheck(ctx context.Context, args []string) error {
	if err := flag.NewFlagSet("check", flag.ContinueOnError).Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	for _, step := range []struct {
		name string
		run  func(context.Context, []string) error
	}{{"lint", runLint}, {"test", runTest}, {"build", runBuild}} {
		if err := step.run(ctx, nil); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}
	return nil
}
