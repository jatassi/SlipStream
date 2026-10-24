package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/linkage"
	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/worktree"
)

func init() {
	register("build", "build cmd/slipstream, cmd/slipstream-e2e for test jobs, and cmd/slipstream-app on Windows and macOS, for the host target into dist/", runBuild)
}

var mains = []struct {
	pkg, name string
	role      linkage.Role
	goos      []string
}{
	{pkg: "./cmd/slipstream", name: install.Executable, role: linkage.Node},
	{pkg: "./cmd/slipstream-e2e", name: "slipstream-e2e", role: linkage.Node},
	{pkg: "./cmd/slipstream-app", name: install.Trampoline, role: linkage.Trampoline, goos: []string{"windows", "darwin"}},
}

func runBuild(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	dist := flags.String("o", "", "output directory (default dist/ in the repository)")
	withKeys := flags.Bool("bundle-keys", false, "carry SlipStream's metadata keys from "+strings.Join(bundledEnv(), " and ")+" in cmd/slipstream; only the release workflow's tag build passes it")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	keys, err := bundledKeys(*withKeys, os.Getenv)
	if err != nil {
		return err
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	if *dist == "" {
		*dist = filepath.Join(root, "dist")
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

	if err := os.MkdirAll(*dist, 0o755); err != nil {
		return fmt.Errorf("create dist: %w", err)
	}
	tags := strings.Join(append([]string{"release"}, target.GoTags...), ",")
	stamped := "-X=github.com/jatassi/slipstream/internal/buildinfo.stamp=" + ldflag
	for _, m := range mains {
		if m.goos != nil && !slices.Contains(m.goos, target.GOOS) {
			continue
		}
		out := executable(target, *dist, m.name)
		flags, env := nodeBuild(m.pkg, tags, stamped, keys), bundleEnv(bundle)
		if m.role == linkage.Trampoline {
			flags, env = trampolineBuild(target.GOOS)
		}
		if err := buildMain(ctx, root, out, m.pkg, m.role, flags, env); err != nil {
			return err
		}
		fmt.Println(out)
	}
	return placeTools(bundle, *dist)
}

// bundledKeyVars are the providers SlipStream bundles a key for, each with
// the variable the release workflow puts its repository secret in.
var bundledKeyVars = []struct{ provider, env string }{
	{"tmdb", "SLIPSTREAM_TMDB_KEY"},
	{"omdb", "SLIPSTREAM_OMDB_KEY"},
}

func bundledEnv() []string {
	names := make([]string, len(bundledKeyVars))
	for i, v := range bundledKeyVars {
		names[i] = v.env
	}
	return names
}

// bundledKeys is the -X flag that bundles each provider's key from getenv,
// and none without withKeys: then it never reads them.
func bundledKeys(withKeys bool, getenv func(string) string) (string, error) {
	if !withKeys {
		return "", nil
	}
	pairs := make([]string, len(bundledKeyVars))
	for i, v := range bundledKeyVars {
		key := getenv(v.env)
		if key == "" {
			return "", fmt.Errorf("-bundle-keys: %s is empty", v.env)
		}
		if strings.ContainsAny(key, ",: \t\r\n'\"") {
			return "", fmt.Errorf("-bundle-keys: %s holds a character no API key has", v.env)
		}
		pairs[i] = v.provider + ":" + key
	}
	return "-X=github.com/jatassi/slipstream/internal/metadata.bundled=" + strings.Join(pairs, ","), nil
}

// nodeBuild is the flags of a Node main: only cmd/slipstream carries the
// bundled keys.
func nodeBuild(pkg, tags, stamped, keys string) []string {
	ldflags := stamped
	if pkg == "./cmd/slipstream" && keys != "" {
		ldflags += " " + keys
	}
	return []string{"-trimpath", "-tags", tags, "-ldflags", ldflags}
}

func trampolineBuild(goos string) (flags, env []string) {
	ldflags := "-s -w"
	if goos == "windows" {
		ldflags += " -H windowsgui"
	}
	return []string{"-trimpath", "-buildvcs=false", "-ldflags", ldflags}, append(bundleEnv(native.Bundle{}), "CGO_ENABLED=0")
}

// placeTools copies the bundle's tool programs into dir, beside the
// binaries built there, as the release manifest ships them (ADR 0015).
// They run as separate processes, so linkage.Check, which governs what a
// binary links, does not apply to them.
func placeTools(bundle native.Bundle, dir string) error {
	for _, src := range bundle.Tools {
		if err := copyProgram(src, filepath.Join(dir, filepath.Base(src))); err != nil {
			return err
		}
	}
	return nil
}

func copyProgram(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open the tool: %w", err)
	}
	defer func() { _ = in.Close() }()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".tool-*.tmp")
	if err != nil {
		return fmt.Errorf("create the tool copy: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = io.Copy(tmp, in)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("copy %s: %w", src, err)
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil { //nolint:gosec // the tools run as programs
		return fmt.Errorf("copy %s: %w", src, err)
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return fmt.Errorf("place %s: %w", dst, err)
	}
	return nil
}

func buildMain(ctx context.Context, root, out, pkg string, role linkage.Role, flags, env []string) error {
	tmp, err := os.CreateTemp(filepath.Dir(out), ".slipstream-*.tmp")
	if err != nil {
		return fmt.Errorf("create build output: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("create build output: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	gobuild := exec.CommandContext(ctx, "go", slices.Concat([]string{"build"}, flags, []string{"-o", tmp.Name(), pkg})...)
	gobuild.Dir = root
	gobuild.Stdout, gobuild.Stderr = os.Stderr, os.Stderr
	gobuild.Env = env
	if err := gobuild.Run(); err != nil {
		return fmt.Errorf("go build %s: %w", pkg, err)
	}
	if err := linkage.Check(tmp.Name(), role); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), out); err != nil {
		return fmt.Errorf("move build output into dist: %w", err)
	}
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
	return worktree.Root(ctx, ".")
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
	if _, err := fetchTarget(ctx, root, cacheRoot, in, target); err != nil {
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
