package native

import (
	"archive/tar"
	"compress/gzip"
	"context"
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

// LayerDir is where a fetched layer lives under the cache root. The path
// holds the asset's SHA-256, so a bump moves every cgo flag that points
// into the layer and Go's build cache rebuilds the cgo packages.
func LayerDir(cacheRoot string, l Layer) string {
	return filepath.Join(cacheRoot, string(l.Kind), l.Target, l.SHA256)
}

// BuildLibs builds the libs layer for t with the recipe and writes it to
// archive as a gzipped tar whose paths are relative to the layer root.
func BuildLibs(ctx context.Context, in Inputs, t Target, recipeDir, cacheRoot, archive string, log io.Writer) error {
	tc, err := t.Compilers()
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "slipstream-libs-")
	if err != nil {
		return err
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
			return fmt.Errorf("cmake %s: %w", args[0], err)
		}
	}
	return pack(prefix, archive)
}

func pack(dir, archive string) error {
	f, err := os.Create(archive)
	if err != nil {
		return err
	}
	zw := gzip.NewWriter(f)
	tw := tar.NewWriter(zw)
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == dir {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() && !info.IsDir() {
			return fmt.Errorf("%s: layers hold only files and directories", path)
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			hdr.Name += "/"
		}
		hdr.Uid, hdr.Gid, hdr.Uname, hdr.Gname = 0, 0, "", ""
		if err := tw.WriteHeader(hdr); err != nil || info.IsDir() {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(tw, src)
		return err
	})
	for _, c := range []io.Closer{tw, zw, f} {
		if closeErr := c.Close(); err == nil {
			err = closeErr
		}
	}
	if err != nil {
		os.Remove(archive)
	}
	return err
}

func extract(archive, dir string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		rel := filepath.FromSlash(strings.TrimSuffix(hdr.Name, "/"))
		if !filepath.IsLocal(rel) {
			return fmt.Errorf("%s: entry %q leaves the layer", archive, hdr.Name)
		}
		path := filepath.Join(dir, rel)
		switch hdr.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(path, 0o755)
		case tar.TypeReg:
			err = writeFile(path, tr, hdr.FileInfo().Mode().Perm())
		default:
			err = fmt.Errorf("%s: entry %q is neither a file nor a directory", archive, hdr.Name)
		}
		if err != nil {
			return err
		}
	}
}

func writeFile(path string, r io.Reader, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}
