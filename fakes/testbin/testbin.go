// Package testbin finds a binary under test: the path a test variable
// names, or one build per test process with the bundled tool programs
// beside it, as a release places them. The variables carry no SLIPSTREAM_
// prefix, so they never reach a Node's strict loader.
package testbin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
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
	e2e        = &binary{variable: "TEST_SLIPSTREAM_E2E_BINARY", pkg: "./cmd/slipstream-e2e"}

	buildDir string
	tools    struct {
		once sync.Once
		err  error
	}
)

// toolsVariable is the variable go run ./cmd/release test and deps env set
// to the path of every tool program the native bundle ships.
const toolsVariable = "SLIPSTREAM_NATIVE_TOOLS"

// Run runs the package's tests and then removes the binaries this process
// built. A package whose tests call Slipstream, E2E or Fakeworld calls it
// from TestMain.
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

// E2E is TEST_SLIPSTREAM_E2E_BINARY when set, otherwise cmd/slipstream-e2e,
// the binary that also registers the test Content Types, built once for the
// test process.
func E2E(t testing.TB) string {
	t.Helper()
	return e2e.find(t)
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
		tools.once.Do(func() { tools.err = placeTools(buildDir) })
		if b.err = tools.err; b.err != nil {
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

// PlaceTools copies the native bundle's tool programs into dir, beside a
// slipstream Build put there, which refuses to run without them.
func PlaceTools(t testing.TB, dir string) {
	t.Helper()
	if err := placeTools(dir); err != nil {
		t.Fatal(err)
	}
}

// placeTools copies the native bundle's tool programs into dir.
func placeTools(dir string) error {
	list, ok := os.LookupEnv(toolsVariable)
	if !ok {
		return fmt.Errorf("%s is unset: run the tests with go run ./cmd/release test, or eval \"$(go run ./cmd/release deps env)\" first", toolsVariable)
	}
	for _, src := range filepath.SplitList(list) {
		if err := copyProgram(src, filepath.Join(dir, filepath.Base(src))); err != nil {
			return err
		}
	}
	return nil
}

func copyProgram(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("%s: %w", toolsVariable, err)
	}
	defer func() { _ = in.Close() }()
	syscall.ForkLock.RLock()
	defer syscall.ForkLock.RUnlock()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755) //nolint:gosec // a tool program must be executable
	if err != nil {
		return fmt.Errorf("place %s: %w", filepath.Base(src), err)
	}
	_, err = io.Copy(out, in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("place %s: %w", filepath.Base(src), err)
	}
	return nil
}

var root struct {
	once sync.Once
	path string
	err  error
}

// ModuleRoot is the directory of the go.mod the test runs under.
func ModuleRoot(t testing.TB) string {
	t.Helper()
	dir, err := moduleRoot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return dir
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
