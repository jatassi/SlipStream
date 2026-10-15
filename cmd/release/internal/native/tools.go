package native

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

// ToolsEnv lists the path of every tool program the fetched bundle ships,
// separated by the OS path list separator, so whatever builds a binary
// outside cmd/release can place the programs beside it.
const ToolsEnv = "SLIPSTREAM_NATIVE_TOOLS"

// ToolDir is where a fetched tool archive lives under the cache root, named
// by the archive's SHA-256 like a layer.
func ToolDir(cacheRoot string, p PinnedTool) string {
	return filepath.Join(cacheRoot, "tool", p.Target, p.SHA256)
}

// Program is the file name of a tool's program on t.
func Program(t Target, name string) string {
	if t.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// FetchTool makes the tool archive present in the cache and returns its
// directory, the way Fetch does a layer. The archive must hold each of
// programs, as named for t, at its root.
func FetchTool(ctx context.Context, cacheRoot string, p PinnedTool, t Target, programs []string) (string, error) {
	dir := ToolDir(cacheRoot, p)
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("check the native cache: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", fmt.Errorf("create the native cache: %w", err)
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".fetching-")
	if err != nil {
		return "", fmt.Errorf("create the native cache: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	archive := filepath.Join(tmp, "archive")
	f, err := os.Create(archive)
	if err != nil {
		return "", fmt.Errorf("create the tool archive: %w", err)
	}
	sum, err := download(ctx, p.URL, f)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	if sum != p.SHA256 {
		return "", fmt.Errorf("%s has SHA-256 %s, %s pins %s", p.URL, sum, LockPath, p.SHA256)
	}
	extracted := filepath.Join(tmp, "tool")
	if err := extractTool(archive, p.URL, extracted); err != nil {
		return "", err
	}
	for _, name := range programs {
		if _, err := os.Stat(filepath.Join(extracted, Program(t, name))); err != nil {
			return "", fmt.Errorf("%s holds no %s at its root", p.URL, Program(t, name))
		}
	}
	if err := os.Rename(extracted, dir); err != nil {
		if _, statErr := os.Stat(dir); statErr == nil {
			return dir, nil
		}
		return "", fmt.Errorf("move the tool into the native cache: %w", err)
	}
	return dir, nil
}

// PinTools hashes each tool archive.
func PinTools(ctx context.Context, archives []ToolArchive) ([]PinnedTool, error) {
	pinned := []PinnedTool{}
	for _, a := range archives {
		sum, err := download(ctx, a.URL, io.Discard)
		if err != nil {
			return nil, err
		}
		pinned = append(pinned, PinnedTool{a, sum})
	}
	return pinned, nil
}

func extractTool(archive, url, dir string) error {
	switch {
	case strings.HasSuffix(url, ".tar.xz"):
		return extractTar(archive, dir, func(r io.Reader) (io.Reader, error) { return xz.NewReader(r) })
	case strings.HasSuffix(url, ".tar.gz"):
		return extractTar(archive, dir, func(r io.Reader) (io.Reader, error) { return gzip.NewReader(r) })
	case strings.HasSuffix(url, ".zip"):
		return extractZip(archive, dir)
	}
	return fmt.Errorf("%s is neither a .tar.xz, a .tar.gz nor a .zip archive", url)
}

func extractTar(archive, dir string, decompress func(io.Reader) (io.Reader, error)) error {
	f, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("open the tool archive: %w", err)
	}
	defer func() { _ = f.Close() }()
	dr, err := decompress(bufio.NewReader(f))
	if err != nil {
		return fmt.Errorf("%s: %w", archive, err)
	}
	tr := tar.NewReader(dr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", archive, err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if err := extractEntry(dir, hdr.Name, tr, hdr.FileInfo().Mode().Perm()); err != nil {
			return err
		}
	}
}

func extractZip(archive, dir string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("%s: %w", archive, err)
	}
	defer func() { _ = zr.Close() }()
	for _, f := range zr.File {
		if !f.Mode().IsRegular() {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return fmt.Errorf("%s: %w", archive, err)
		}
		err = extractEntry(dir, f.Name, r, 0o755)
		if closeErr := r.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func extractEntry(dir, name string, r io.Reader, perm fs.FileMode) error {
	rel := filepath.FromSlash(name)
	if !filepath.IsLocal(rel) {
		return fmt.Errorf("tool archive entry %q leaves the archive", name)
	}
	return writeFile(filepath.Join(dir, rel), r, perm)
}
