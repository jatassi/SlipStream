package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/cmd/release/internal/pack"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/worktree"
)

func init() {
	register("package", "build the host target's packages from dist/ into -o: -tag vX.Y.Z refuses any other stamp, -source adds the ffmpeg source, -sign signpath writes the SignPath request", runPackage)
}

const signPathDir = "signpath"

func runPackage(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("package", flag.ContinueOnError)
	out := flags.String("o", "", "output directory (required)")
	dist := flags.String("from", "", "release build's output (default dist/ in the repository)")
	tag := flags.String("tag", "", "the release tag this build must be stamped with, clean")
	source := flags.Bool("source", false, "also write each tool's pinned source archive")
	sign := flags.String("sign", "", "signing service to write a request for: signpath")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if *out == "" {
		return errors.New("-o is required")
	}
	outDir, err := filepath.Abs(*out)
	if err != nil {
		return fmt.Errorf("find -o: %w", err)
	}
	*out = outDir
	if *sign != "" && *sign != "signpath" {
		return fmt.Errorf("-sign %q is not signpath", *sign)
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	if *dist == "" {
		*dist = filepath.Join(root, "dist")
	}
	in, err := native.LoadInputs(filepath.Join(root, native.InputsPath))
	if err != nil {
		return err
	}
	target, err := in.Target(native.HostTargetName())
	if err != nil {
		return err
	}
	stamp, err := buildinfo.ReadFile(filepath.Join(*dist, native.Program(target, install.Executable)))
	if err != nil {
		return err
	}
	version, err := stampVersion(stamp, target, *tag)
	if err != nil {
		return err
	}
	if err := cleanOutput(*out); err != nil {
		return err
	}
	if err := (pack.Manifest{Version: version.String(), Target: target.Name}).Write(*out); err != nil {
		return err
	}
	cacheRoot, err := native.CacheRoot()
	if err != nil {
		return err
	}
	m, err := pack.Build(ctx, pack.Input{
		Root: root, Dist: *dist, Out: *out, Version: version, Target: target, Inputs: in, CacheRoot: cacheRoot, Log: os.Stderr,
	})
	if err != nil {
		return err
	}
	if err := addExtras(ctx, &m, in, cacheRoot, *out, *source, *sign); err != nil {
		return err
	}
	for _, a := range m.Assets {
		fmt.Println(filepath.Join(*out, a))
	}
	return m.Write(*out)
}

func addExtras(ctx context.Context, m *pack.Manifest, in native.Inputs, cacheRoot, out string, source bool, sign string) error {
	if source {
		sources, err := writeSources(ctx, in, cacheRoot, out)
		if err != nil {
			return err
		}
		m.Assets = append(m.Assets, sources...)
	}
	if sign == "signpath" {
		return writeSignPathRequest(*m, out)
	}
	return nil
}

func stampVersion(stamp buildinfo.Stamp, target native.Target, tag string) (pack.Version, error) {
	if stamp.Target != target.Name {
		return "", fmt.Errorf("dist holds a build for %s, and this runner packages %s", stamp.Target, target.Name)
	}
	if tag == "" {
		return pack.Described(stamp.Version), nil
	}
	if stamp.Version != tag || stamp.Fingerprint != worktree.CleanFingerprint {
		return "", fmt.Errorf("refusing to package %s from a build stamped %s with fingerprint %s; a release takes a clean build of its tag", tag, stamp.Version, stamp.Fingerprint)
	}
	return pack.ParseTag(tag)
}

func cleanOutput(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return makeOutput(dir)
	}
	if err != nil {
		return fmt.Errorf("read the output directory: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}
	if _, err := os.Stat(filepath.Join(dir, pack.ManifestName)); err != nil {
		return fmt.Errorf("output directory %s is not empty and holds no %s from an earlier run", dir, pack.ManifestName)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove the earlier packages: %w", err)
	}
	return makeOutput(dir)
}

func makeOutput(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create the output directory: %w", err)
	}
	return nil
}

func writeSources(ctx context.Context, in native.Inputs, cacheRoot, out string) ([]string, error) {
	var names []string
	for _, tool := range in.Tools {
		if tool.SourceArchive == nil {
			continue
		}
		file, err := native.FetchFile(ctx, cacheRoot, *tool.SourceArchive)
		if err != nil {
			return nil, err
		}
		name := fmt.Sprintf("%s-%s-source.tar.gz", tool.Name, tool.Version)
		if err := copyProgram(file, filepath.Join(out, name)); err != nil {
			return nil, err
		}
		if err := os.Chmod(filepath.Join(out, name), 0o644); err != nil { //nolint:gosec // G302: a release asset
			return nil, fmt.Errorf("write %s: %w", name, err)
		}
		names = append(names, name)
	}
	return names, nil
}

// writeSignPathRequest copies the Windows packages into the directory the
// workflow uploads for SignPath, whose artifact configuration signs the
// two SlipStream programs inside each and the MSI itself.
func writeSignPathRequest(m pack.Manifest, out string) error {
	dir := filepath.Join(out, signPathDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create the SignPath request: %w", err)
	}
	for _, k := range []install.Kind{install.MSI, install.ZIP} {
		if len(m.Packages[k]) == 0 {
			return fmt.Errorf("a SignPath request signs the Windows packages, and this runner built no %s", k)
		}
		for _, name := range m.Packages[k] {
			if err := copyProgram(filepath.Join(out, name), filepath.Join(dir, name)); err != nil {
				return err
			}
		}
	}
	return nil
}
