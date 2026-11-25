package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/platform"
)

// platformTag tags the browser specs of the platform suite, which darwin
// runs on WebKit, the one engine no other target runs.
const platformTag = "@platform"

// testPlatform runs the platform suite: the Go tests that platform.Tests
// lists, under the race detector, and on darwin the browser specs tagged
// platformTag. On CI it first readies the runner for the tests' worlds.
func testPlatform(ctx context.Context, root string, _ testOptions) ([]failure, error) {
	_, target, bundle, err := goTargetBundle(ctx)
	if err != nil {
		return nil, err
	}
	env := bundleEnv(bundle)
	if inCI() {
		if env, err = preparePlatformRunner(ctx, root, target.GOOS, env); err != nil {
			return nil, err
		}
	}
	base := []string{"-race", "-shuffle=on", "-timeout", "25m"}
	fmt.Printf("platform tier on %s:\n", target.Name)
	all := goRun{}
	var runs []string
	for _, t := range platform.Tests {
		fmt.Printf("  %s -run %s\n", t.Package, t.Run)
		all.packages = append(all.packages, "./"+t.Package)
		runs = append(runs, "(?:"+t.Run+")")
	}
	// One go test builds each package once, where a go test per package
	// would build their shared dependencies in each. So each pattern
	// reaches every listed package, and a test that shares a name with one
	// another package lists runs too.
	all.run = strings.Join(runs, "|")
	_, first, goErr := runShard(ctx, root, env, base, []goRun{all})
	failed := rerunFailed(first, func(f []failure) results {
		_, again, _ := runShard(ctx, root, env, append(slices.Clone(base), "-count=1"), rerunRuns(f))
		return again
	})
	if target.GOOS != "darwin" {
		return failed, goErr
	}
	browserFailed, browserErr := testBrowser(ctx, root, testOptions{filter: platformTag, repeat: 1})
	return append(failed, browserFailed...), errors.Join(goErr, browserErr)
}

// preparePlatformRunner does for the suite's worlds what the e2e job's
// steps do before its tests: Windows keeps TEMP on a drive far slower than
// the runner's temp dir, the Nodes of a world bind loopback addresses that
// macOS answers only once aliased, and worlds that launch together on a
// cold fixture cache encode past their launch limits. The job sets
// TEST_SLIPSTREAM_BINARY and TEST_FAKEWORLD_BINARY to its build output.
func preparePlatformRunner(ctx context.Context, root, goos string, env []string) ([]string, error) {
	slipstream, _ := testBinary(root, "TEST_SLIPSTREAM_BINARY")
	fakeworld, _ := testBinary(root, "TEST_FAKEWORLD_BINARY")
	switch goos {
	case "windows":
		temp := os.Getenv("RUNNER_TEMP")
		env = append(env, "TMP="+temp, "TEMP="+temp)
	case "darwin":
		if err := run(ctx, root, nil, "sudo", fakeworld, "setup-loopback"); err != nil {
			return nil, err
		}
	}
	if err := run(ctx, root, env, fakeworld, "fixtures", "--binary", slipstream); err != nil {
		return nil, err
	}
	return env, nil
}
