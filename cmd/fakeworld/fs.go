package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/world"
)

var fsCommands = map[string]func(runDir string, args []string) int{
	"stat":  fsStat,
	"match": fsMatch,
	"rm":    fsChange("rm", 1, remove),
	"mv":    fsChange("mv", 2, move),
	"cp":    fsChange("cp", 2, copyPath),
}

func fsCommand(args []string) int {
	if len(args) < 2 {
		return usage(errors.New("fs: takes stat, match, rm, mv or cp and a run dir"))
	}
	command, ok := fsCommands[args[0]]
	if !ok {
		return usage(fmt.Errorf("fs: unknown command %q", args[0]))
	}
	return command(args[1], args[2:])
}

// inRun is path made absolute, refused unless it lies in the run dir or
// the run's scratch dir.
func inRun(m world.Manifest, path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("path: %w", err)
	}
	for _, dir := range []string{m.RunDir, m.ScratchDir} {
		if rel, err := filepath.Rel(dir, abs); err == nil && filepath.IsLocal(rel) {
			return abs, nil
		}
	}
	return "", fmt.Errorf("%s is outside the run dir %s and the scratch dir %s", abs, m.RunDir, m.ScratchDir)
}

func fsStat(runDir string, args []string) int {
	if len(args) != 1 {
		return usage(errors.New("fs stat: takes a run dir and a path"))
	}
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, "fs stat", args, func(context.Context) int {
		return printed("fs stat", func() (any, error) {
			m, err := world.ReadLastManifest(runDir)
			if err != nil {
				return nil, err
			}
			path, err := inRun(m, args[0])
			if err != nil {
				return nil, err
			}
			return world.StatFile(path)
		})
	})
}

func fsMatch(runDir string, args []string) int {
	fs := newFlagSet("fs match")
	var release string
	fs.StringVar(&release, "release", "", "")
	positional, err := parse(fs, args)
	if err != nil {
		return usage(err)
	}
	if len(positional) != 1 {
		return usage(errors.New("fs match: takes a run dir and a path"))
	}
	if err := required(fs, "release"); err != nil {
		return usage(err)
	}
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, "fs match", args, func(context.Context) int {
		return printed("fs match", func() (any, error) {
			m, err := world.ReadLastManifest(runDir)
			if err != nil {
				return nil, err
			}
			path, err := inRun(m, positional[0])
			if err != nil {
				return nil, err
			}
			return world.MatchRelease(m, path, release)
		})
	})
}

// changed is what rm, mv and cp print: the path they removed, or the
// one that now holds the file or folder.
type changed struct {
	Path string `json:"path"`
}

// fsChange is a journaled verb that changes files in the live run's
// media dir, as a person changes a library on disk: change runs on the
// paths, each relative to the media dir or absolute inside it.
func fsChange(verb string, paths int, change func(paths []string) error) func(runDir string, args []string) int {
	verb = "fs " + verb
	return func(runDir string, args []string) int {
		if len(args) != paths {
			return usage(fmt.Errorf("%s: takes a run dir and %d paths", verb, paths))
		}
		ctx, stop := interrupted()
		defer stop()
		return journal(ctx, runDir, verb, args, func(context.Context, world.Controls, world.Step) (int, map[string]string) {
			return printed(verb, func() (any, error) {
				m, err := world.ReadLastManifest(runDir)
				if err != nil {
					return nil, err
				}
				abs := make([]string, len(args))
				for i, p := range args {
					if abs[i], err = inMedia(m, p); err != nil {
						return nil, err
					}
				}
				if err := change(abs); err != nil {
					return nil, err
				}
				return changed{Path: abs[len(abs)-1]}, nil
			}), nil
		})
	}
}

// inMedia is path inside the run's media dir, which a relative path is
// relative to. The media dir itself and anything outside it are refused.
func inMedia(m world.Manifest, path string) (string, error) {
	if m.MediaDir == "" {
		return "", errors.New("the run has no media dir because its Scenario has no library")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(m.MediaDir, filepath.FromSlash(path))
	}
	if rel, err := filepath.Rel(m.MediaDir, path); err != nil || rel == "." || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("%s is not inside the media dir %s", path, m.MediaDir)
	}
	return filepath.Clean(path), nil
}

func remove(paths []string) error {
	if _, err := os.Lstat(paths[0]); err != nil {
		return err //nolint:wrapcheck // the error names the path
	}
	return os.RemoveAll(paths[0]) //nolint:wrapcheck // the error names the path
}

// move renames a file or folder to a path that doesn't exist yet, making
// its parent folders.
func move(paths []string) error {
	if err := newPath(paths[1]); err != nil {
		return err
	}
	return os.Rename(paths[0], paths[1]) //nolint:wrapcheck // the error names the path
}

// copyPath copies a file, or a folder and all it holds, to a path that
// doesn't exist yet, making its parent folders.
func copyPath(paths []string) error {
	from, to := paths[0], paths[1]
	info, err := os.Stat(from)
	if err != nil {
		return err //nolint:wrapcheck // the error names the path
	}
	if err := newPath(to); err != nil {
		return err
	}
	if info.IsDir() {
		return os.CopyFS(to, os.DirFS(from)) //nolint:wrapcheck // the error names the path
	}
	in, err := os.Open(from)
	if err != nil {
		return err //nolint:wrapcheck // the error names the path
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err //nolint:wrapcheck // the error names the path
	}
	_, err = io.Copy(out, in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err
}

func newPath(p string) error {
	if _, err := os.Lstat(p); err == nil {
		return fmt.Errorf("%s already exists", p)
	}
	return os.MkdirAll(filepath.Dir(p), 0o755) //nolint:wrapcheck // the error names the path
}
