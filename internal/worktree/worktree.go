// Package worktree derives the build stamp's source fields from a git worktree.
package worktree

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/buildinfo"
)

// Root is the top directory of the git worktree that holds dir.
func Root(ctx context.Context, dir string) (string, error) {
	top, err := git(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return filepath.FromSlash(strings.TrimSpace(string(top))), nil
}

// CleanFingerprint is the fingerprint of a worktree with nothing changed or untracked.
const CleanFingerprint = "clean"

// Stamp returns the version, commit and fingerprint of the worktree at dir.
func Stamp(ctx context.Context, dir string) (buildinfo.Stamp, error) {
	version, err := git(ctx, dir, "describe", "--tags", "--always", "--match", "v[0-9]*")
	if err != nil {
		return buildinfo.Stamp{}, err
	}
	commit, err := git(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return buildinfo.Stamp{}, err
	}
	fingerprint, err := Fingerprint(ctx, dir)
	if err != nil {
		return buildinfo.Stamp{}, err
	}
	return buildinfo.Stamp{
		Version:     strings.TrimSpace(string(version)),
		Commit:      strings.TrimSpace(string(commit)),
		Fingerprint: fingerprint,
	}, nil
}

// Fingerprint hashes the path and content of every file git reports as changed
// or untracked, with deleted files hashed as deleted.
func Fingerprint(ctx context.Context, dir string) (string, error) {
	status, err := git(ctx, dir, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return "", err
	}
	paths := changedPaths(status)
	if len(paths) == 0 {
		return CleanFingerprint, nil
	}
	h := sha256.New()
	for _, p := range paths {
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			h.Write(fmt.Appendf(nil, "%s\x00deleted\n", p))
		case err != nil:
			return "", fmt.Errorf("fingerprint the worktree: %w", err)
		default:
			h.Write(fmt.Appendf(nil, "%s\x00%x\n", p, sha256.Sum256(content)))
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func changedPaths(status []byte) []string {
	var paths []string
	for len(status) > 0 {
		entry, rest, _ := bytes.Cut(status, []byte{0})
		paths = append(paths, string(entry[3:]))
		if entry[0] == 'R' || entry[0] == 'C' {
			var orig []byte
			orig, rest, _ = bytes.Cut(rest, []byte{0})
			paths = append(paths, string(orig))
		}
		status = rest
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}

func git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, bytes.TrimSpace(stderr.Bytes()))
	}
	return out, nil
}
