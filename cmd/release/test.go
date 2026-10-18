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
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

func init() {
	register("test", "run the test tiers: -tier go|e2e|component|browser|online|all (default all, which leaves out online); -tier go and e2e pass the rest to go test -race -shuffle=on, -tier online to go test -tags online ./conformance/...", runTest)
}

type testTier struct {
	name string
	run  func(ctx context.Context, root string, o testOptions) error
	// goArgs says the tier hands the arguments after its flags to go test.
	goArgs bool
	// sharded tiers take -shard, -filter and -repeat.
	sharded bool
	// timed tiers split their shards by -timings and write -record.
	timed bool
	// online tiers reach real services, so all leaves them out.
	online bool
}

var testTiers = []testTier{
	{name: "go", run: testGo, goArgs: true, sharded: true, timed: true},
	{name: "e2e", run: testE2E, goArgs: true, sharded: true, timed: true},
	{name: "component", run: testComponent},
	{name: "browser", run: testBrowser, sharded: true},
	{name: "online", run: testOnline, goArgs: true, online: true},
}

type testOptions struct {
	shard   shard
	filter  string
	repeat  int
	timings string
	record  string
	goArgs  []string
}

// shard is the index-th of count, from 1. The zero shard is the whole tier.
type shard struct{ index, count int }

func (s *shard) String() string { return fmt.Sprintf("%d/%d", s.index, s.count) }

func (s *shard) Set(v string) error {
	i, n, ok := strings.Cut(v, "/")
	index, err1 := strconv.Atoi(i)
	count, err2 := strconv.Atoi(n)
	if !ok || err1 != nil || err2 != nil || index < 1 || index > count {
		return fmt.Errorf("shard %q is not i/n with 1 <= i <= n", v)
	}
	*s = shard{index, count}
	return nil
}

// runTest reads its own flags ahead of go test's, so `release test -tier
// go -run Foo ./e2e` hands `-run Foo ./e2e` to go test.
func runTest(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	tier := fs.String("tier", "all", "go, e2e, component, browser, online or all")
	var o testOptions
	fs.Var(&o.shard, "shard", "run the i-th of n shards of the tier, as i/n")
	fs.StringVar(&o.filter, "filter", "", "run only the tests that match, as go test -run or playwright --grep")
	fs.IntVar(&o.repeat, "repeat", 1, "run each test this many times, as go test -count or playwright --repeat-each")
	fs.StringVar(&o.timings, "timings", "", "directory of earlier runs' timings to split the shards by")
	fs.StringVar(&o.record, "record", "", "write this run's timings to this file")
	ownArgs, goArgs := splitOwnFlags(fs, args)
	if err := fs.Parse(ownArgs); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	o.goArgs = goArgs
	tiers := slices.DeleteFunc(slices.Clone(testTiers), func(t testTier) bool { return t.online })
	if *tier != "all" {
		i := slices.IndexFunc(testTiers, func(t testTier) bool { return t.name == *tier })
		if i < 0 {
			return fmt.Errorf("-tier %q is not one of go, e2e, component, browser, online, all", *tier)
		}
		tiers = testTiers[i : i+1]
	}
	if err := o.fits(tiers); err != nil {
		return err
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	var failed []string
	for _, t := range tiers {
		if err := t.run(ctx, root, o); err != nil {
			fmt.Fprintf(os.Stderr, "%s tier: %v\n", t.name, err)
			failed = append(failed, t.name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed: %s", strings.Join(failed, ", "))
	}
	return nil
}

func (o testOptions) fits(tiers []testTier) error {
	one := len(tiers) == 1
	switch {
	case len(o.goArgs) > 0 && (!one || !tiers[0].goArgs):
		return fmt.Errorf("go test arguments %q need -tier go, e2e or online", o.goArgs)
	case (o.shard != shard{} || o.filter != "" || o.repeat != 1) && (!one || !tiers[0].sharded):
		return errors.New("-shard, -filter and -repeat need -tier go, e2e or browser")
	case (o.timings != "" || o.record != "") && (!one || !tiers[0].timed):
		return errors.New("-timings and -record need -tier go or e2e")
	case len(o.goArgs) > 0 && (o.shard != shard{} || o.filter != "" || o.repeat != 1 || o.timings != "" || o.record != ""):
		return fmt.Errorf("go test arguments %q pick the tests themselves, so they take none of -shard, -filter, -repeat, -timings or -record", o.goArgs)
	case o.repeat < 1:
		return fmt.Errorf("-repeat %d is not a positive count", o.repeat)
	}
	return nil
}

// splitOwnFlags splits the leading flags fs defines from the arguments
// that follow, since a FlagSet refuses go test's flags. Each of fs's flags
// takes a value.
func splitOwnFlags(fs *flag.FlagSet, args []string) (own, rest []string) {
	i := 0
	for i < len(args) {
		name, _, hasValue := strings.Cut(strings.TrimLeft(args[i], "-"), "=")
		if !strings.HasPrefix(args[i], "-") || fs.Lookup(name) == nil {
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

func testGo(ctx context.Context, root string, o testOptions) error {
	return testGoPackages(ctx, root, o, "go")
}

// testE2E runs the packages whose tests drive the slipstream, slipstream-e2e
// and fakeworld binaries through fakes/testbin, which CI hands the build
// job's dist/ through TEST_SLIPSTREAM_BINARY, TEST_SLIPSTREAM_E2E_BINARY and
// TEST_FAKEWORLD_BINARY.
func testE2E(ctx context.Context, root string, o testOptions) error {
	return testGoPackages(ctx, root, o, "e2e")
}

// testOnline runs every conformance suite against the real service it
// checks. TEST_ONLINE_TMDB_KEY carries TMDB's key; -update rewrites each
// suite's shape.json.
func testOnline(ctx context.Context, root string, o testOptions) error {
	args := append([]string{"test", "-tags", "online", "-count=1", "./conformance/..."}, o.goArgs...)
	gotest := exec.CommandContext(ctx, "go", args...)
	gotest.Dir = root
	gotest.Stdout, gotest.Stderr = os.Stdout, os.Stderr
	if err := gotest.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}
	return nil
}

// testComponent installs the target's browsers first, because Vitest's
// browser mode launches the same Chromium the browser tier does.
func testComponent(ctx context.Context, root string, _ testOptions) error {
	if err := installWeb(ctx, root); err != nil {
		return err
	}
	web := filepath.Join(root, "web")
	if err := run(ctx, web, nil, "bun", "run", "e2e:install"); err != nil {
		return err
	}
	return run(ctx, web, nil, "bun", "run", "test")
}

// testBrowser needs Go only to find or build a binary the variables leave
// out, so CI's browser shards run it from the build job's dist/ alone.
func testBrowser(ctx context.Context, root string, o testOptions) error {
	slipstream, haveSlipstream := testBinary(root, "TEST_SLIPSTREAM_BINARY")
	fakeworld, haveFakeworld := testBinary(root, "TEST_FAKEWORLD_BINARY")
	if !haveSlipstream || !haveFakeworld {
		_, target, bundle, err := goTargetBundle(ctx)
		if err != nil {
			return err
		}
		dist := filepath.Join(root, "dist")
		if !haveSlipstream {
			slipstream = executable(target, dist, "slipstream")
		}
		if !haveFakeworld {
			fakeworld = executable(target, dist, "fakeworld")
			if err := run(ctx, root, bundleEnv(bundle), "go", "build", "-o", fakeworld, "./cmd/fakeworld"); err != nil {
				return err
			}
		}
	}
	if _, err := os.Stat(slipstream); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s: run release build first: the browser tier needs the web app embedded", slipstream)
	} else if err != nil {
		return fmt.Errorf("release binary: %w", err)
	}
	if err := installWeb(ctx, root); err != nil {
		return err
	}
	web := filepath.Join(root, "web")
	env := append(os.Environ(), "TEST_SLIPSTREAM_BINARY="+slipstream, "TEST_FAKEWORLD_BINARY="+fakeworld)
	if err := run(ctx, web, env, "bun", "run", "e2e:install"); err != nil {
		return err
	}
	return run(ctx, web, env, "bun", append([]string{"run", "e2e"}, playwrightArgs(o)...)...)
}

func playwrightArgs(o testOptions) []string {
	var args []string
	if o.shard != (shard{}) {
		args = append(args, "--shard="+o.shard.String())
	}
	if o.filter != "" {
		args = append(args, "--grep="+o.filter)
	}
	if o.repeat > 1 {
		args = append(args, "--repeat-each="+strconv.Itoa(o.repeat))
	}
	return args
}

// testBinary is the variable's path, read from the repo root as testbin
// reads it, and whether the variable is set.
func testBinary(root, variable string) (string, bool) {
	p, ok := os.LookupEnv(variable)
	if !ok {
		return "", false
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return p, true
}

func executable(target native.Target, dir, name string) string {
	if target.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}
