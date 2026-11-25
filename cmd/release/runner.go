package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// prepareRunner readies a CI runner for a tier's tests and returns env
// with what they need. It runs only on CI, where sudo asks for no
// password, so a developer's machine never runs it. Windows runners keep
// TEMP on a drive far slower than the runner's temp dir, and the tests'
// temp dirs live under TEMP. The tiers that run worlds take the build
// job's dist/ from an artifact, which keeps no executable bits, and their
// TEST_SLIPSTREAM_BINARY and TEST_FAKEWORLD_BINARY name binaries there.
func prepareRunner(ctx context.Context, root, tier, goos string, env []string) ([]string, error) {
	if !inCI() {
		return env, nil
	}
	if goos == "windows" {
		temp := os.Getenv("RUNNER_TEMP")
		env = append(slices.Clone(env), "TMP="+temp, "TEMP="+temp)
	}
	if tier == "go" {
		return env, addTestAddresses(ctx, goos)
	}
	if err := restoreExecutableBits(filepath.Join(root, "dist")); err != nil {
		return nil, err
	}
	slipstream, _ := testBinary(root, "TEST_SLIPSTREAM_BINARY")
	fakeworld, _ := testBinary(root, "TEST_FAKEWORLD_BINARY")
	if goos == "darwin" {
		if err := setUpLoopback(ctx, fakeworld); err != nil {
			return nil, err
		}
	}
	// A pull request reaches no fixture cache another branch saved, and
	// worlds that launch together on a cold cache encode past their 2
	// minute launch limits.
	if err := run(ctx, root, env, fakeworld, "fixtures", "--binary", slipstream); err != nil {
		return nil, err
	}
	return env, nil
}

// addTestAddresses adds the addresses the go tier's tests listen on.
// TestTheListenerFollowsTheDefaultRoute moves a listener to 127.0.0.2,
// which macOS answers only once aliased. With one address besides loopback
// a host announces once whatever the engine does, so
// TestEachTorrentAnnouncesStartedOnce needs a second on linux.
func addTestAddresses(ctx context.Context, goos string) error {
	switch goos {
	case "darwin":
		return run(ctx, "", nil, "sudo", "ifconfig", "lo0", "alias", "127.0.0.2", "up")
	case "linux":
		out, err := exec.CommandContext(ctx, "ip", "-4", "route", "show", "default").Output()
		if err != nil {
			return fmt.Errorf("find the default route: %w", err)
		}
		fields := strings.Fields(string(out))
		i := slices.Index(fields, "dev")
		if i < 0 || i+1 == len(fields) {
			return fmt.Errorf("the default route %q names no interface", strings.TrimSpace(string(out)))
		}
		return run(ctx, "", nil, "sudo", "ip", "addr", "replace", "198.51.100.7/32", "dev", fields[i+1])
	}
	return nil
}

func restoreExecutableBits(dist string) error {
	entries, err := os.ReadDir(dist)
	if err != nil {
		return fmt.Errorf("read the build output: %w", err)
	}
	for _, e := range entries {
		if e.Type().IsRegular() {
			if err := os.Chmod(filepath.Join(dist, e.Name()), 0o755); err != nil { //nolint:gosec // the build output is programs
				return fmt.Errorf("restore the executable bits on the build output: %w", err)
			}
		}
	}
	return nil
}

// setUpLoopback aliases the loopback addresses the worlds' Nodes bind,
// which macOS answers only once aliased. A runner is the one place
// setup-loopback runs as root, so a second run there checks that it
// changes nothing.
func setUpLoopback(ctx context.Context, fakeworld string) error {
	if err := run(ctx, "", nil, "sudo", fakeworld, "setup-loopback"); err != nil {
		return err
	}
	out, err := exec.CommandContext(ctx, "sudo", fakeworld, "setup-loopback").Output()
	if err != nil {
		return fmt.Errorf("set up the loopback aliases again: %w", err)
	}
	if got := strings.TrimSpace(string(out)); got != "loopback aliases already set up" {
		return fmt.Errorf("a second setup-loopback changed the aliases: %s", got)
	}
	return nil
}
