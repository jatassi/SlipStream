package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/linkage"
	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/cmd/release/internal/worktree"
	"github.com/jatassi/slipstream/internal/buildinfo"
)

func init() {
	register("build", "build cmd/slipstream for the host target into dist/", runBuild)
}

func runBuild(ctx context.Context, args []string) error {
	if err := flag.NewFlagSet("build", flag.ContinueOnError).Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	stamp, err := worktree.Stamp(ctx, root)
	if err != nil {
		return err
	}
	in, target, bundle, err := goTargetBundle(ctx)
	if err != nil {
		return err
	}
	libtorrent, err := in.Component("libtorrent-rasterbar")
	if err != nil {
		return err
	}
	stamp.Target = target.Name
	stamp.NativeBundle = bundle.Libs.Digest
	stamp.Libtorrent = libtorrent.Version
	ldflag, err := buildinfo.Encode(stamp)
	if err != nil {
		return err
	}

	if err := buildWeb(ctx, root); err != nil {
		return err
	}

	dist := filepath.Join(root, "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		return fmt.Errorf("create dist: %w", err)
	}
	tmp, err := os.CreateTemp(dist, ".slipstream-*.tmp")
	if err != nil {
		return fmt.Errorf("create build output: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("create build output: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	tags := strings.Join(append([]string{"release"}, target.GoTags...), ",")
	gobuild := exec.CommandContext(ctx, "go", "build", "-trimpath", "-tags", tags,
		"-ldflags", "-X=github.com/jatassi/slipstream/internal/buildinfo.stamp="+ldflag,
		"-o", tmp.Name(), "./cmd/slipstream")
	gobuild.Dir = root
	gobuild.Stdout, gobuild.Stderr = os.Stderr, os.Stderr
	gobuild.Env = bundleEnv(bundle)
	if err := gobuild.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	if err := linkage.Check(tmp.Name()); err != nil {
		return err
	}
	out := executable(target, dist, "slipstream")
	if err := os.Rename(tmp.Name(), out); err != nil {
		return fmt.Errorf("move build output into dist: %w", err)
	}
	fmt.Println(out)
	return nil
}

// run sends the output of name to stderr. A nil env inherits the process
// environment.
func run(ctx context.Context, dir string, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func repoRoot(ctx context.Context) (string, error) {
	top, err := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(top)), nil
}

func installWeb(ctx context.Context, root string) error {
	return run(ctx, filepath.Join(root, "web"), nil, "bun", "install", "--frozen-lockfile")
}

func buildWeb(ctx context.Context, root string) error {
	if err := installWeb(ctx, root); err != nil {
		return err
	}
	return run(ctx, filepath.Join(root, "web"), nil, "bun", "run", "build")
}

// goTargetBundle fetches the pinned layers of the target the go command
// builds for, which GOOS and GOARCH can change from the host's.
func goTargetBundle(ctx context.Context) (native.Inputs, native.Target, native.Bundle, error) {
	goenv, err := exec.CommandContext(ctx, "go", "env", "GOOS", "GOARCH").Output()
	if err != nil {
		return native.Inputs{}, native.Target{}, native.Bundle{}, fmt.Errorf("go env: %w", err)
	}
	goos, goarch, _ := strings.Cut(strings.TrimSpace(string(goenv)), "\n")
	root, in, target, err := loadTarget(ctx, goos+"-"+goarch)
	if err != nil {
		return native.Inputs{}, native.Target{}, native.Bundle{}, err
	}
	cacheRoot, err := native.CacheRoot()
	if err != nil {
		return native.Inputs{}, native.Target{}, native.Bundle{}, err
	}
	if _, err := fetchTarget(ctx, root, cacheRoot, target); err != nil {
		return native.Inputs{}, native.Target{}, native.Bundle{}, err
	}
	bundle, err := native.FetchedBundle(root, cacheRoot, in, target)
	return in, target, bundle, err
}

// bundleEnv is the process environment with the C toolchain, cgo flags and
// GOFLAGS replaced by the native bundle's.
func bundleEnv(b native.Bundle) []string {
	env := slices.DeleteFunc(os.Environ(), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return name == "CC" || name == "CXX" || name == "GOFLAGS" || strings.HasPrefix(name, "CGO_")
	})
	for _, v := range b.Env {
		env = append(env, v.Name+"="+v.Value)
	}
	return env
}
