package main

import (
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

func runBuild(args []string) error {
	if err := flag.NewFlagSet("build", flag.ContinueOnError).Parse(args); err != nil {
		return err
	}
	root, err := repoRoot()
	if err != nil {
		return err
	}
	stamp, err := worktree.Stamp(root)
	if err != nil {
		return err
	}
	in, target, bundle, err := goTargetBundle()
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

	if err := buildWeb(root); err != nil {
		return err
	}

	dist := filepath.Join(root, "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dist, ".slipstream-*.tmp")
	if err != nil {
		return err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	tags := strings.Join(append([]string{"release"}, target.GoTags...), ",")
	gobuild := exec.Command("go", "build", "-trimpath", "-tags", tags,
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
	out := filepath.Join(dist, "slipstream")
	if target.GOOS == "windows" {
		out += ".exe"
	}
	if err := os.Rename(tmp.Name(), out); err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func repoRoot() (string, error) {
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(top)), nil
}

func buildWeb(root string) error {
	web := filepath.Join(root, "web")
	if err := run(web, "bun", "install", "--frozen-lockfile"); err != nil {
		return err
	}
	return run(web, "bun", "run", "build")
}

// goTargetBundle fetches the pinned layers of the target the go command
// builds for, which GOOS and GOARCH can change from the host's.
func goTargetBundle() (native.Inputs, native.Target, native.Bundle, error) {
	goenv, err := exec.Command("go", "env", "GOOS", "GOARCH").Output()
	if err != nil {
		return native.Inputs{}, native.Target{}, native.Bundle{}, fmt.Errorf("go env: %w", err)
	}
	goos, goarch, _ := strings.Cut(strings.TrimSpace(string(goenv)), "\n")
	root, in, target, err := loadTarget(goos + "-" + goarch)
	if err != nil {
		return native.Inputs{}, native.Target{}, native.Bundle{}, err
	}
	cacheRoot, err := native.CacheRoot()
	if err != nil {
		return native.Inputs{}, native.Target{}, native.Bundle{}, err
	}
	if _, err := fetchTarget(root, cacheRoot, target); err != nil {
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
