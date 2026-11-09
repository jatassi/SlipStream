package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type checkout struct {
	upstream
	key, dir, date string
}

// fetch brings each upstream's sparse paths at its pinned commit into
// dir/<key>, reusing what an earlier run fetched. It uses git because
// codeload tarballs get a 403 response through the agent proxy.
func fetch(ctx context.Context, dir string, p pins) (map[string]checkout, error) {
	out := map[string]checkout{}
	for key, u := range p.Upstreams {
		c := checkout{upstream: u, key: key, dir: filepath.Join(dir, key)}
		if err := c.sync(ctx); err != nil {
			return nil, fmt.Errorf("%s: %w", u.Name, err)
		}
		date, err := c.git(ctx, "log", "-1", "--format=%cs")
		if err != nil {
			return nil, err
		}
		c.date = date
		out[key] = c
	}
	return out, nil
}

func (c checkout) sync(ctx context.Context) error {
	if _, err := os.Stat(filepath.Join(c.dir, ".git")); errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(c.dir, 0o750); err != nil {
			return fmt.Errorf("make the checkout directory: %w", err)
		}
		if _, err := c.git(ctx, "init", "-q"); err != nil {
			return err
		}
		if _, err := c.git(ctx, "remote", "add", "origin", c.Repo); err != nil {
			return err
		}
	}
	if _, err := c.git(ctx, "remote", "set-url", "origin", c.Repo); err != nil {
		return err
	}
	if _, err := c.git(ctx, append([]string{"sparse-checkout", "set", "--no-cone"}, c.Paths...)...); err != nil {
		return err
	}
	if head, _ := c.git(ctx, "rev-parse", "HEAD"); head == c.Commit {
		return nil
	}
	if _, err := c.git(ctx, "fetch", "-q", "--depth", "1", "--filter=blob:none", "origin", c.Commit); err != nil {
		return err
	}
	_, err := c.git(ctx, "checkout", "-q", "--detach", "FETCH_HEAD")
	return err
}

func (c checkout) git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // G204: git, with arguments this command builds
	cmd.Dir = c.dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

func (c checkout) read(path string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(c.dir, filepath.FromSlash(path)))
	if err != nil {
		return nil, fmt.Errorf("%s at %s: %w", c.Name, c.Commit[:8], err)
	}
	return b, nil
}

func (c checkout) source(path string) (*source, error) {
	b, err := c.read(path)
	if err != nil {
		return nil, err
	}
	return lex(c.Name+" "+path, string(b))
}
