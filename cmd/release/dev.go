package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	register("dev", "start a Fake world and the Vite dev server pointed at its Node: [-scenario single-machine|first-run]", runDev)
}

// devManifest is the part of fakeworld's manifest that dev reads.
type devManifest struct {
	Nodes []struct {
		URL string `json:"url"`
		Log string `json:"log"`
	} `json:"nodes"`
	Admin *struct {
		Email string `json:"email"`
		PIN   string `json:"pin"`
	} `json:"admin"`
}

func runDev(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	scenario := fs.String("scenario", "single-machine", "the catalog Scenario to start, such as single-machine or first-run")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	_, target, bundle, err := goTargetBundle(ctx)
	if err != nil {
		return err
	}
	bin := filepath.Join(root, "dist", "dev")
	slipstream := executable(target, bin, "slipstream")
	fakeworld := executable(target, bin, "fakeworld")
	for pkg, out := range map[string]string{"./cmd/slipstream": slipstream, "./cmd/fakeworld": fakeworld} {
		if err := run(ctx, root, bundleEnv(bundle), "go", "build", "-o", out, pkg); err != nil {
			return err
		}
	}
	if err := placeTools(bundle, bin); err != nil {
		return err
	}
	if err := installWeb(ctx, root); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, ".dev"), 0o755); err != nil {
		return fmt.Errorf("create .dev: %w", err)
	}
	runDir := filepath.Join(root, ".dev", "run-"+time.Now().Format("20060102-150405"))
	start := exec.CommandContext(ctx, fakeworld, "start", "--binary", slipstream, "--scenario", *scenario, "--run-dir", runDir)
	start.Stderr = os.Stderr
	out, err := start.Output()
	if err != nil {
		return fmt.Errorf("fakeworld start: %w", err)
	}
	defer stopWorld(context.WithoutCancel(ctx), fakeworld, runDir)

	origin, err := announceWorld(os.Stderr, strings.TrimSpace(string(out)))
	if err != nil {
		return err
	}
	vite := exec.CommandContext(ctx, filepath.Join(root, "web", "node_modules", ".bin", "vite"))
	vite.Dir = filepath.Join(root, "web")
	vite.Env = append(os.Environ(), "DEV_API_ORIGIN="+origin)
	vite.Stdin, vite.Stdout, vite.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := vite.Run(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("vite: %w", err)
	}
	return nil
}

// announceWorld prints how to get in to the world's Node and returns the
// Node's URL.
func announceWorld(w io.Writer, manifestPath string) (string, error) {
	// os.Root, as world.ReadManifest says, lets the supervisor replace the
	// manifest while it is read.
	root, err := os.OpenRoot(filepath.Dir(manifestPath))
	if err != nil {
		return "", fmt.Errorf("read manifest: %w", err)
	}
	defer func() { _ = root.Close() }()
	data, err := root.ReadFile(filepath.Base(manifestPath))
	if err != nil {
		return "", fmt.Errorf("read manifest %s: %w", manifestPath, err)
	}
	var m devManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("read manifest %s: %w", manifestPath, err)
	}
	if len(m.Nodes) == 0 {
		return "", fmt.Errorf("manifest %s lists no Node", manifestPath)
	}
	node := m.Nodes[0]
	_, _ = fmt.Fprintf(w, "Node %s, manifest %s\n", node.URL, manifestPath)
	if m.Admin != nil {
		_, _ = fmt.Fprintf(w, "sign in as %s with PIN %s\n", m.Admin.Email, m.Admin.PIN)
		return node.URL, nil
	}
	code, err := setupCode(node.Log)
	if err != nil {
		return "", err
	}
	_, _ = fmt.Fprintf(w, "Setup code %s\n", code)
	return node.URL, nil
}

// setupCode reads the code from the Node's log, where a person finds it.
func setupCode(log string) (string, error) {
	data, err := os.ReadFile(log)
	if err != nil {
		return "", fmt.Errorf("read the Node's log: %w", err)
	}
	lines := bufio.NewScanner(bytes.NewReader(data))
	for lines.Scan() {
		var rec struct{ Msg, Code string }
		if json.Unmarshal(lines.Bytes(), &rec) == nil && rec.Msg == "setup code" {
			return rec.Code, nil
		}
	}
	return "", errors.New(log + " has no setup code record")
}

func stopWorld(ctx context.Context, fakeworld, runDir string) {
	stop := exec.CommandContext(ctx, fakeworld, "stop", runDir)
	stop.Stdout, stop.Stderr = os.Stderr, os.Stderr
	if err := stop.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fakeworld stop %s: %v\n", runDir, err)
	}
}
