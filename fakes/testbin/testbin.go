// Package testbin finds a binary under test: the path a test variable
// names, or one build per test process. The variables carry no SLIPSTREAM_
// prefix, so they never reach a Node's strict loader.
package testbin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

type binary struct {
	variable, pkg string

	once sync.Once
	path string
	err  error
}

var (
	slipstream = &binary{variable: "TEST_SLIPSTREAM_BINARY", pkg: "./cmd/slipstream"}
	fakeworld  = &binary{variable: "TEST_FAKEWORLD_BINARY", pkg: "./cmd/fakeworld"}

	buildDir string
)

// Run runs the package's tests and then removes the binaries this process
// built. A package whose tests call Slipstream or Fakeworld calls it from
// TestMain.
func Run(m *testing.M) int {
	dir, err := os.MkdirTemp("", "testbin")
	if err != nil {
		fmt.Fprintln(os.Stderr, "testbin:", err)
		return 1
	}
	buildDir = dir
	defer func() { _ = os.RemoveAll(dir) }()
	return m.Run()
}

// Slipstream is TEST_SLIPSTREAM_BINARY when set, otherwise cmd/slipstream
// built once for the test process.
func Slipstream(t testing.TB) string {
	t.Helper()
	return slipstream.find(t)
}

// Fakeworld is TEST_FAKEWORLD_BINARY when set, otherwise cmd/fakeworld
// built once for the test process.
func Fakeworld(t testing.TB) string {
	t.Helper()
	return fakeworld.find(t)
}

func (b *binary) find(t testing.TB) string {
	t.Helper()
	b.once.Do(func() {
		if p, ok := os.LookupEnv(b.variable); ok {
			b.path, b.err = fromVariable(t.Context(), b.variable, p)
			return
		}
		if buildDir == "" {
			b.err = fmt.Errorf("%s is unset and nothing can build %s: call testbin.Run from TestMain", b.variable, b.pkg)
			return
		}
		b.path, b.err = build(t.Context(), buildDir, b.pkg)
	})
	if b.err != nil {
		t.Fatal(b.err)
	}
	return b.path
}

func fromVariable(ctx context.Context, name, p string) (string, error) {
	if !filepath.IsAbs(p) {
		root, err := moduleRoot(ctx)
		if err != nil {
			return "", err
		}
		p = filepath.Join(root, p)
	}
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return p, nil
}

// Build runs go build with flags on pkg, a path from the module root such
// as ./cmd/slipstream, into dir, and returns the binary's path.
func Build(t testing.TB, dir, pkg string, flags ...string) string {
	t.Helper()
	p, err := build(t.Context(), dir, pkg, flags...)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func build(ctx context.Context, dir, pkg string, flags ...string) (string, error) {
	root, err := moduleRoot(ctx)
	if err != nil {
		return "", err
	}
	out := filepath.Join(dir, path.Base(pkg))
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	args := append(append([]string{"build"}, flags...), "-o", out, pkg)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = root
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go build %s: %w\n%s", pkg, err, b)
	}
	return out, nil
}

var root struct {
	once sync.Once
	path string
	err  error
}

func moduleRoot(ctx context.Context) (string, error) {
	root.once.Do(func() {
		out, err := exec.CommandContext(ctx, "go", "env", "GOMOD").Output()
		gomod := strings.TrimSpace(string(out))
		switch {
		case err != nil:
			root.err = fmt.Errorf("go env GOMOD: %w", err)
		case gomod == "" || gomod == os.DevNull:
			root.err = errors.New("go env GOMOD: the test is not inside a module")
		default:
			root.path = filepath.Dir(gomod)
		}
	})
	return root.path, root.err
}
