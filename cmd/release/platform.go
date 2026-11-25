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
// The job sets TEST_SLIPSTREAM_BINARY and TEST_FAKEWORLD_BINARY to its
// build output.
func testPlatform(ctx context.Context, root string, o testOptions) (results, error) {
	_, target, bundle, err := goTargetBundle(ctx)
	if err != nil {
		return results{}, err
	}
	env, err := prepareRunner(ctx, root, "platform", target.GOOS, bundleEnv(bundle))
	if err != nil {
		return results{}, err
	}
	base := []string{"-race", "-shuffle=on", "-timeout", "25m"}
	fmt.Printf("platform tier on %s:\n", target.Name)
	quarantined := goQuarantine(o.quarantined)
	runs := onlyQuarantinedRuns(quarantined)
	if !o.onlyQuarantined {
		all := goRun{}
		var patterns []string
		for _, t := range platform.Tests {
			fmt.Printf("  %s -run %s\n", t.Package, t.Run)
			all.packages = append(all.packages, modulePath+"/"+t.Package)
			patterns = append(patterns, "(?:"+t.Run+")")
		}
		// One go test builds each package once, where a go test per package
		// would build their shared dependencies in each. So each pattern
		// reaches every listed package, and a test that shares a name with one
		// another package lists runs too.
		all.run = strings.Join(patterns, "|")
		runs = skipQuarantined([]goRun{all}, quarantined)
	}
	var r results
	var goErr error
	if len(runs) > 0 {
		var first results
		_, first, goErr = runShard(ctx, root, env, base, runs)
		failed := rerunFailed(first, func(f []failure) results {
			_, again, _ := runShard(ctx, root, env, append(slices.Clone(base), "-count=1"), rerunRuns(f))
			return again
		})
		r = results{failed: failed, passed: first.passed}
	}
	browserIDs := slices.DeleteFunc(slices.Clone(o.quarantined), func(id string) bool { return runnerOf(id) != playwrightRunner })
	if target.GOOS != "darwin" || o.onlyQuarantined && len(browserIDs) == 0 {
		return r, goErr
	}
	browser, browserErr := runBrowserTier(ctx, root, os.Environ(), testOptions{filter: platformTag, repeat: 1, quarantined: browserIDs, onlyQuarantined: o.onlyQuarantined, systemDeps: o.systemDeps})
	r.add(browser)
	return r, errors.Join(goErr, browserErr)
}
