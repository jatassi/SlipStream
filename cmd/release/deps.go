package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

var depsCommands = map[string]command{
	"build":   {"build one of a target's layers into a publishable archive", depsBuild},
	"env":     {"print the build environment of the fetched bundle", depsEnv},
	"fetch":   {"download, verify and extract a target's pinned layers", depsFetch},
	"missing": {"print the CI matrix of wanted layers that are not published", depsMissing},
	"pin":     {"write native/lock.json from the published layers", depsPin},
	"verify":  {"check native/lock.json against the inputs and recipe", depsVerify},
}

func init() {
	register("deps", "fetch, pin, verify and build the native bundle layers", runDeps)
}

func runDeps(ctx context.Context, args []string) error {
	names := make([]string, 0, len(depsCommands))
	for name := range depsCommands {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(args) == 0 {
		var b strings.Builder
		b.WriteString("usage: deps <command> [args]")
		for _, name := range names {
			fmt.Fprintf(&b, "\n  %-8s %s", name, depsCommands[name].summary)
		}
		return errors.New(b.String())
	}
	cmd, ok := depsCommands[args[0]]
	if !ok {
		return fmt.Errorf("unknown deps command %q; one of %s", args[0], strings.Join(names, ", "))
	}
	return cmd.run(ctx, args[1:])
}

func targetFlag(fs *flag.FlagSet) *string {
	return fs.String("target", native.HostTargetName(), "target from "+native.InputsPath)
}

func loadTarget(ctx context.Context, name string) (string, native.Inputs, native.Target, error) {
	root, err := repoRoot(ctx)
	if err != nil {
		return "", native.Inputs{}, native.Target{}, err
	}
	in, err := native.LoadInputs(filepath.Join(root, native.InputsPath))
	if err != nil {
		return "", native.Inputs{}, native.Target{}, err
	}
	t, err := in.Target(name)
	return root, in, t, err
}

func wantRepo(ctx context.Context, name string, args []string) (string, native.Inputs, []native.LayerID, error) {
	if err := flag.NewFlagSet("deps "+name, flag.ContinueOnError).Parse(args); err != nil {
		return "", native.Inputs{}, nil, fmt.Errorf("parse flags: %w", err)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return "", native.Inputs{}, nil, err
	}
	in, want, err := native.WantRepo(root)
	return root, in, want, err
}

func depsBuild(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("deps build", flag.ContinueOnError)
	target := targetFlag(fs)
	kind := fs.String("kind", "", "layer kind to build: toolchain or libs (required)")
	archive := fs.String("archive", "", "write the layer to this .tar.gz (required)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	root, in, t, err := loadTarget(ctx, *target)
	if err != nil {
		return err
	}
	if *archive == "" {
		return errors.New("-archive is required")
	}
	if t.GOOS != runtime.GOOS || t.GOARCH != runtime.GOARCH {
		return fmt.Errorf("target %s builds on its own OS and architecture, not on %s", t.Name, native.HostTargetName())
	}
	cacheRoot, err := native.CacheRoot()
	if err != nil {
		return err
	}
	recipe := filepath.Join(root, native.RecipeDir)
	switch native.Kind(*kind) {
	case native.KindToolchain:
		return native.BuildToolchain(ctx, in, t, recipe, cacheRoot, *archive, os.Stderr)
	case native.KindLibs:
		return native.BuildLibs(ctx, in, t, recipe, cacheRoot, *archive, os.Stderr)
	}
	return fmt.Errorf("-kind is %s or %s, not %q", native.KindToolchain, native.KindLibs, *kind)
}

func depsEnv(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("deps env", flag.ContinueOnError)
	target := targetFlag(fs)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	root, in, t, err := loadTarget(ctx, *target)
	if err != nil {
		return err
	}
	cacheRoot, err := native.CacheRoot()
	if err != nil {
		return err
	}
	b, err := native.FetchedBundle(root, cacheRoot, in, t)
	if err != nil {
		return err
	}
	fmt.Print(native.ExportLines(b.Env))
	return nil
}

func depsFetch(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("deps fetch", flag.ContinueOnError)
	target := targetFlag(fs)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	root, in, t, err := loadTarget(ctx, *target)
	if err != nil {
		return err
	}
	cacheRoot, err := native.CacheRoot()
	if err != nil {
		return err
	}
	dirs, err := fetchTarget(ctx, root, cacheRoot, in, t)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		fmt.Println(dir)
	}
	return nil
}

// fetchTarget makes every layer and tool archive the lock pins for t
// present in the cache and returns their directories.
func fetchTarget(ctx context.Context, root, cacheRoot string, in native.Inputs, t native.Target) ([]string, error) {
	lock, err := native.LoadLock(filepath.Join(root, native.LockPath))
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, l := range lock.Layers {
		if l.Target != t.Name {
			continue
		}
		dir, err := native.Fetch(ctx, cacheRoot, l)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, dir)
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("%s pins no layer for %s", native.LockPath, t.Name)
	}
	for _, tool := range in.Tools {
		p, err := lock.PinnedTool(native.ToolArchive{Tool: tool.Name, Target: t.Name, URL: tool.Archives[t.Name]})
		if err != nil {
			return nil, err
		}
		dir, err := native.FetchTool(ctx, cacheRoot, p, t, tool.Programs)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

func depsVerify(ctx context.Context, args []string) error {
	if err := flag.NewFlagSet("deps verify", flag.ContinueOnError).Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	return runLint(ctx, []string{"-rule", "native-lock"})
}

func depsPin(ctx context.Context, args []string) error {
	root, in, want, err := wantRepo(ctx, "pin", args)
	if err != nil {
		return err
	}
	lock, err := native.Pin(ctx, native.ReleasesURL, want)
	if err != nil {
		return err
	}
	if lock.Tools, err = native.PinTools(ctx, in.ToolArchives()); err != nil {
		return err
	}
	return lock.Write(filepath.Join(root, native.LockPath))
}

type matrixEntry struct {
	Target string `json:"target"`
	Runner string `json:"runner"`
	Tag    string `json:"tag"`
	Asset  string `json:"asset"`
	// Xcode is the Xcode version the runner selects to build the layer.
	Xcode string `json:"xcode,omitempty"`
}

type matrix struct {
	Include []matrixEntry `json:"include"`
}

// depsMissing prints one CI matrix per layer kind. The bundle workflow
// builds and publishes the toolchain layers before the libs layers that
// download them.
func depsMissing(ctx context.Context, args []string) error {
	_, in, want, err := wantRepo(ctx, "missing", args)
	if err != nil {
		return err
	}
	missing := map[native.Kind]*matrix{
		native.KindToolchain: {[]matrixEntry{}},
		native.KindLibs:      {[]matrixEntry{}},
	}
	for _, id := range want {
		published, err := native.Published(ctx, native.ReleasesURL, id)
		if err != nil {
			return err
		}
		if published {
			continue
		}
		t, err := in.Target(id.Target)
		if err != nil {
			return err
		}
		tc, err := in.Toolchain(t)
		if err != nil {
			return err
		}
		missing[id.Kind].Include = append(missing[id.Kind].Include, matrixEntry{t.Name, t.Runner, id.Tag(), id.AssetName(), tc.Xcode})
	}
	if err := json.NewEncoder(os.Stdout).Encode(missing); err != nil {
		return fmt.Errorf("print the missing layers: %w", err)
	}
	return nil
}
