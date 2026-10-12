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
	goenv, err := exec.Command("go", "env", "GOOS", "GOARCH").Output()
	if err != nil {
		return fmt.Errorf("go env: %w", err)
	}
	goos, goarch, _ := strings.Cut(strings.TrimSpace(string(goenv)), "\n")
	in, err := native.LoadInputs(filepath.Join(root, native.InputsPath))
	if err != nil {
		return err
	}
	target, err := in.Target(goos + "-" + goarch)
	if err != nil {
		return err
	}
	cacheRoot, err := native.CacheRoot()
	if err != nil {
		return err
	}
	bundle, err := native.FetchedLibs(root, cacheRoot, in, target)
	if err != nil {
		return err
	}
	libtorrent, err := in.Component("libtorrent-rasterbar")
	if err != nil {
		return err
	}
	stamp.Target = target.Name
	stamp.NativeBundle = bundle.Layer.Digest
	stamp.Libtorrent = libtorrent.Version
	ldflag, err := buildinfo.Encode(stamp)
	if err != nil {
		return err
	}

	web := filepath.Join(root, "web")
	if err := run(web, "bun", "install", "--frozen-lockfile"); err != nil {
		return err
	}
	if err := run(web, "bun", "run", "build"); err != nil {
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
	gobuild := exec.Command("go", "build", "-trimpath", "-tags", "release",
		"-ldflags", "-X=github.com/jatassi/slipstream/internal/buildinfo.stamp="+ldflag,
		"-o", tmp.Name(), "./cmd/slipstream")
	gobuild.Dir = root
	gobuild.Stdout, gobuild.Stderr = os.Stderr, os.Stderr
	gobuild.Env = append(withoutCgoEnv(os.Environ()), "CGO_ENABLED=1")
	for _, v := range bundle.Env {
		gobuild.Env = append(gobuild.Env, v.Name+"="+v.Value)
	}
	if err := gobuild.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	if err := linkage.Check(tmp.Name()); err != nil {
		return err
	}
	out := filepath.Join(dist, "slipstream")
	if goos == "windows" {
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

// withoutCgoEnv drops the C toolchain and cgo flags from env, so the build's
// cgo environment comes only from the native bundle.
func withoutCgoEnv(env []string) []string {
	return slices.DeleteFunc(env, func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return name == "CC" || name == "CXX" || strings.HasPrefix(name, "CGO_")
	})
}
