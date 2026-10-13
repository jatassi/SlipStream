package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

func init() {
	register("test", "run the test tiers: -tier go|component|browser|all (default all); -tier go passes the rest to go test -race -shuffle=on", runTest)
}

type testTier struct {
	name string
	run  func(ctx context.Context, root string, goArgs []string) error
}

var testTiers = []testTier{
	{"go", testGo},
	{"component", testComponent},
	{"browser", testBrowser},
}

// runTest reads -tier ahead of go test's own flags, so `release test -tier
// go -run Foo ./e2e` hands `-run Foo ./e2e` to go test.
func runTest(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	tier := fs.String("tier", "all", "go, component, browser or all")
	tierArgs, goArgs := splitTierFlag(args)
	if err := fs.Parse(tierArgs); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	tiers := testTiers
	if *tier != "all" {
		i := slices.IndexFunc(testTiers, func(t testTier) bool { return t.name == *tier })
		if i < 0 {
			return fmt.Errorf("-tier %q is not one of go, component, browser, all", *tier)
		}
		tiers = testTiers[i : i+1]
	}
	if len(goArgs) > 0 && *tier != "go" {
		return fmt.Errorf("go test arguments %q need -tier go", goArgs)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	var failed []string
	for _, t := range tiers {
		if err := t.run(ctx, root, goArgs); err != nil {
			fmt.Fprintf(os.Stderr, "%s tier: %v\n", t.name, err)
			failed = append(failed, t.name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed: %s", strings.Join(failed, ", "))
	}
	return nil
}

// splitTierFlag splits the leading -tier flags from the arguments that
// follow, since a FlagSet refuses go test's flags.
func splitTierFlag(args []string) (tierArgs, rest []string) {
	i := 0
	for i < len(args) {
		name, _, hasValue := strings.Cut(strings.TrimLeft(args[i], "-"), "=")
		if !strings.HasPrefix(args[i], "-") || name != "tier" {
			break
		}
		i++
		if !hasValue {
			i++
		}
	}
	i = min(i, len(args))
	return args[:i], args[i:]
}

func testGo(ctx context.Context, root string, goArgs []string) error {
	_, _, bundle, err := goTargetBundle(ctx)
	if err != nil {
		return err
	}
	if len(goArgs) == 0 {
		goArgs = []string{"./..."}
	}
	gotest := exec.CommandContext(ctx, "go", append([]string{"test", "-race", "-shuffle=on"}, goArgs...)...)
	gotest.Dir = root
	gotest.Stdout, gotest.Stderr = os.Stdout, os.Stderr
	gotest.Env = bundleEnv(bundle)
	if err := gotest.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}
	return nil
}

// testComponent installs the target's browsers first, because Vitest's
// browser mode launches the same Chromium the browser tier does.
func testComponent(ctx context.Context, root string, _ []string) error {
	if err := installWeb(ctx, root); err != nil {
		return err
	}
	web := filepath.Join(root, "web")
	if err := run(ctx, web, nil, "bun", "run", "e2e:install"); err != nil {
		return err
	}
	return run(ctx, web, nil, "bun", "run", "test")
}

func testBrowser(ctx context.Context, root string, _ []string) error {
	_, target, bundle, err := goTargetBundle(ctx)
	if err != nil {
		return err
	}
	dist := filepath.Join(root, "dist")
	slipstream := testBinary(root, "TEST_SLIPSTREAM_BINARY", executable(target, dist, "slipstream"))
	if _, err := os.Stat(slipstream); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s: run release build first: the browser tier needs the web app embedded", slipstream)
	} else if err != nil {
		return fmt.Errorf("release binary: %w", err)
	}
	fakeworld := testBinary(root, "TEST_FAKEWORLD_BINARY", "")
	if fakeworld == "" {
		fakeworld = executable(target, dist, "fakeworld")
		if err := run(ctx, root, bundleEnv(bundle), "go", "build", "-o", fakeworld, "./cmd/fakeworld"); err != nil {
			return err
		}
	}
	if err := installWeb(ctx, root); err != nil {
		return err
	}
	web := filepath.Join(root, "web")
	env := append(os.Environ(), "TEST_SLIPSTREAM_BINARY="+slipstream, "TEST_FAKEWORLD_BINARY="+fakeworld)
	if err := run(ctx, web, env, "bun", "run", "e2e:install"); err != nil {
		return err
	}
	return run(ctx, web, env, "bun", "run", "e2e")
}

// testBinary is the variable's path, read from the repo root as testbin
// reads it, or def when the variable is unset.
func testBinary(root, variable, def string) string {
	p, ok := os.LookupEnv(variable)
	if !ok {
		return def
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return p
}

func executable(target native.Target, dir, name string) string {
	if target.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}
