// Package worktree derives the build stamp's source fields from a git worktree.
package worktree

import (
	"bytes"
	"crypto/sha256"
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

// CleanFingerprint is the fingerprint of a worktree with nothing changed or untracked.
const CleanFingerprint = "clean"

// Stamp returns the version, commit and fingerprint of the worktree at dir.
func Stamp(dir string) (buildinfo.Stamp, error) {
	version, err := git(dir, "describe", "--tags", "--always", "--match", "v[0-9]*")
	if err != nil {
		return buildinfo.Stamp{}, err
	}
	commit, err := git(dir, "rev-parse", "HEAD")
	if err != nil {
		return buildinfo.Stamp{}, err
	}
	fingerprint, err := Fingerprint(dir)
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
func Fingerprint(dir string) (string, error) {
	status, err := git(dir, "status", "--porcelain=v1", "-z", "--untracked-files=all")
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
			fmt.Fprintf(h, "%s\x00deleted\n", p)
		case err != nil:
			return "", err
		default:
			fmt.Fprintf(h, "%s\x00%x\n", p, sha256.Sum256(content))
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
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

func git(dir string, args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, bytes.TrimSpace(stderr.Bytes()))
	}
	return out, nil
}
