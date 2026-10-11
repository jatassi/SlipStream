package native

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CacheRoot is the per-machine native cache that every worktree shares.
func CacheRoot() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "slipstream", "native"), nil
}

// LibsDir is where the libs layer for t lives under the cache root. key is
// the layer digest for a locally built layer.
func LibsDir(cacheRoot string, t Target, key string) string {
	return filepath.Join(cacheRoot, "libs", t.Name, key)
}

// BuildLibs builds the libs layer for t with the recipe and returns its
// directory. It builds into a temporary directory beside the final one and
// renames it into place, so the cache never holds a half-built layer. An
// existing layer is returned as is.
func BuildLibs(ctx context.Context, in Inputs, t Target, recipeDir, cacheRoot string, log io.Writer) (string, error) {
	tc, err := t.Compilers()
	if err != nil {
		return "", err
	}
	digest, err := LibsDigest(in, t, os.DirFS(recipeDir))
	if err != nil {
		return "", err
	}
	dir := LibsDir(cacheRoot, t, digest)
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".building-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	prefix, work := filepath.Join(tmp, "layer"), filepath.Join(tmp, "work")
	configure := []string{
		"-S", recipeDir, "-B", work,
		"-DCMAKE_C_COMPILER=" + tc.CC,
		"-DCMAKE_CXX_COMPILER=" + tc.CXX,
		"-DLAYER_PREFIX=" + prefix,
		"-DDOWNLOAD_DIR=" + filepath.Join(cacheRoot, "downloads"),
		"-DOPENSSL_TARGET=" + t.OpenSSLTarget,
		"-DSYSTEM_LIBS=" + strings.Join(t.SystemLibs, ";"),
	}
	for _, c := range in.Components {
		id := strings.ToUpper(strings.ReplaceAll(c.Name, "-", "_"))
		configure = append(configure, "-D"+id+"_URL="+c.URL, "-D"+id+"_SHA256="+c.SHA256)
	}
	for _, args := range [][]string{configure, {"--build", work}} {
		cmd := exec.CommandContext(ctx, "cmake", args...)
		cmd.Stdout, cmd.Stderr = log, log
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("cmake %s: %w", args[0], err)
		}
	}
	if err := os.Rename(prefix, dir); err != nil {
		// A concurrent build of the same digest renamed its layer first.
		if _, statErr := os.Stat(dir); statErr == nil {
			return dir, nil
		}
		return "", err
	}
	return dir, nil
}

// FindLibs returns the cached libs layer for t, built from the current
// inputs and recipe.
func FindLibs(in Inputs, t Target, recipeDir, cacheRoot string) (string, error) {
	digest, err := LibsDigest(in, t, os.DirFS(recipeDir))
	if err != nil {
		return "", err
	}
	dir := LibsDir(cacheRoot, t, digest)
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("no libs layer for %s in %s: run scripts/native-deps build -target %s", t.Name, cacheRoot, t.Name)
	} else if err != nil {
		return "", err
	}
	return dir, nil
}
