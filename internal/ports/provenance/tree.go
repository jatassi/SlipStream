package provenance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"
)

// Tree is a clone of Prowlarr's repository. A blobless clone
// (--filter=blob:none) is enough for Check, which reads only trees.
type Tree struct {
	Dir string
}

// Git runs git in the clone with every setting that shapes the output
// of log and diff fixed, so the same history gives the same bytes on
// every machine.
func (t *Tree) Git(ctx context.Context, args ...string) ([]byte, error) {
	return t.run(ctx, nil, args...)
}

func (t *Tree) run(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	fixed := []string{"-C", t.Dir, "-c", "core.quotePath=true", "-c", "color.ui=false", "-c", "diff.algorithm=myers",
		"-c", "diff.indentHeuristic=true", "-c", "diff.interHunkContext=0", "-c", "diff.suppressBlankEmpty=false", "-c", "log.showSignature=false"}
	cmd := exec.CommandContext(ctx, "git", append(fixed, args...)...) //nolint:gosec // G204: git, with arguments this package builds
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// Missing is the files that aren't in the tree at commit, sorted.
func (t *Tree) Missing(commit string, files []string) ([]string, error) {
	ctx := context.Background()
	if _, err := t.Git(ctx, "cat-file", "-e", commit+"^{commit}"); err != nil {
		return nil, fmt.Errorf("commit %s isn't in the Prowlarr clone", commit)
	}
	present, err := t.Files(ctx, commit, files...)
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, f := range files {
		if !slices.Contains(present, f) {
			missing = append(missing, f)
		}
	}
	slices.Sort(missing)
	return missing, nil
}

// Files lists the files at commit under each of paths, sorted. It reads
// trees only.
func (t *Tree) Files(ctx context.Context, commit string, paths ...string) ([]string, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths to list")
	}
	out, err := t.Git(ctx, append([]string{"ls-tree", "-r", "-z", "--name-only", commit, "--"}, paths...)...)
	if err != nil {
		return nil, err
	}
	files := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	if len(files) == 1 && files[0] == "" {
		return nil, nil
	}
	slices.Sort(files)
	return files, nil
}

// Read is the content of each of paths at commit. A blobless clone
// fetches the blobs it lacks.
func (t *Tree) Read(ctx context.Context, commit string, paths []string) (map[string][]byte, error) {
	var req bytes.Buffer
	for _, p := range paths {
		req.WriteString(commit + ":" + p + "\n")
	}
	out, err := t.run(ctx, req.Bytes(), "cat-file", "--batch")
	if err != nil {
		return nil, err
	}
	files := make(map[string][]byte, len(paths))
	for _, p := range paths {
		header, rest, ok := bytes.Cut(out, []byte("\n"))
		fields := strings.Fields(string(header))
		if !ok || len(fields) != 3 {
			return nil, fmt.Errorf("%s isn't in Prowlarr at %s", p, commit)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size+1 > len(rest) {
			return nil, fmt.Errorf("git cat-file gave %q for %s", header, p)
		}
		files[p] = rest[:size]
		out = rest[size+1:]
	}
	return files, nil
}
