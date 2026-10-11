package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/linkage"
	"github.com/jatassi/slipstream/cmd/release/internal/worktree"
	"github.com/jatassi/slipstream/internal/buildinfo"
)

func init() {
	register("build", "build cmd/slipstream for the host target into dist/", runBuild)
}

func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	bundle := fs.String("native-bundle", "", "digest of the native bundle the cgo environment links")
	libtorrent := fs.String("libtorrent", "", "libtorrent-rasterbar version in the native bundle")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *bundle == "" || *libtorrent == "" {
		fs.Usage()
		return errors.New("-native-bundle and -libtorrent are required")
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
	stamp.Target = goos + "-" + goarch
	stamp.NativeBundle = *bundle
	stamp.Libtorrent = *libtorrent
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
	err = run(root, "go", "build", "-trimpath", "-tags", "release",
		"-ldflags", "-X=github.com/jatassi/slipstream/internal/buildinfo.stamp="+ldflag,
		"-o", tmp.Name(), "./cmd/slipstream")
	if err != nil {
		return err
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
