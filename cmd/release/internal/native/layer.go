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

// CacheRootEnv names a directory that replaces the default cache root. CI
// points it at the runner's temp drive, which on Windows extracts the
// cached layers about four times faster than the user profile's drive.
const CacheRootEnv = "SLIPSTREAM_NATIVE_CACHE"

// CacheRoot is the per-machine native cache that every worktree shares. Its
// name differs from SlipStream v1's data folder, %LOCALAPPDATA%\SlipStream,
// in more than case, since Windows paths ignore case.
func CacheRoot() (string, error) {
	if dir := os.Getenv(CacheRootEnv); dir != "" {
		return dir, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "slipstream-native"), nil
}

// LayerDir is where a fetched layer lives under the cache root. The path
// holds the asset's SHA-256, so a bump moves every cgo flag that points
// into the layer and Go's build cache rebuilds the cgo packages.
func LayerDir(cacheRoot string, l Layer) string {
	return filepath.Join(cacheRoot, string(l.Kind), l.Target, l.SHA256)
}

// BuildToolchain builds the toolchain layer for t with its recipe and writes
// it to archive as a gzipped tar whose paths are relative to the layer root.
func BuildToolchain(ctx context.Context, in Inputs, t Target, recipeDir, cacheRoot, archive string, log io.Writer) error {
	tc, err := in.Toolchain(t)
	if err != nil {
		return err
	}
	return buildLayer(archive, func(prefix, work string) error {
		args := []string{
			"-DLAYER_PREFIX=" + filepath.ToSlash(prefix),
			"-DWORK_DIR=" + filepath.ToSlash(work),
			"-DDOWNLOAD_DIR=" + filepath.ToSlash(downloadDir(cacheRoot)),
			"-DTRIPLE=" + t.Triple,
		}
		for _, s := range tc.Sources {
			args = append(args, sourceDefines(s)...)
		}
		script := filepath.Join(recipeDir, string(KindToolchain), tc.Name, "toolchain.cmake")
		return cmake(ctx, log, nil, append(args, "-P", script)...)
	})
}

// BuildLibs builds the libs layer for t with its recipe and t's toolchain,
// and writes it to archive like BuildToolchain.
func BuildLibs(ctx context.Context, in Inputs, t Target, recipeDir, cacheRoot, archive string, log io.Writer) error {
	toolchainDir, err := os.MkdirTemp("", "slipstream-toolchain-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(toolchainDir)
	cc, err := libsCompilers(ctx, in, t, recipeDir, toolchainDir)
	if err != nil {
		return err
	}
	var env []string
	for name, value := range t.Env {
		env = append(env, name+"="+value)
	}
	return buildLayer(archive, func(prefix, work string) error {
		configure := []string{
			"-G", "Ninja",
			"-S", filepath.Join(recipeDir, string(KindLibs)), "-B", work,
			"-DCMAKE_C_COMPILER=" + filepath.ToSlash(cc.CC),
			"-DCMAKE_CXX_COMPILER=" + filepath.ToSlash(cc.CXX),
			"-DLAYER_PREFIX=" + filepath.ToSlash(prefix),
			"-DDOWNLOAD_DIR=" + filepath.ToSlash(downloadDir(cacheRoot)),
			"-DOPENSSL_TARGET=" + t.OpenSSLTarget,
			"-DSYSTEM_LIBS=" + strings.Join(t.SystemLibs, ";"),
		}
		for _, c := range in.Components {
			configure = append(configure, sourceDefines(c.Source)...)
		}
		if err := cmake(ctx, log, env, configure...); err != nil {
			return err
		}
		return cmake(ctx, log, env, "--build", work)
	})
}

// libsCompilers returns the compilers that build t's libs layer: those of
// its published toolchain layer, extracted into dir, or those of the
// selected Xcode once its version is the pinned one.
func libsCompilers(ctx context.Context, in Inputs, t Target, recipeDir, dir string) (Compilers, error) {
	tc, err := in.Toolchain(t)
	if err != nil {
		return Compilers{}, err
	}
	if !tc.Layered() {
		return xcodeCompilers, checkXcode(ctx, tc.Xcode)
	}
	digest, err := ToolchainDigest(in, t, os.DirFS(recipeDir))
	if err != nil {
		return Compilers{}, err
	}
	id := LayerID{KindToolchain, t.Name, digest}
	archive := filepath.Join(dir, id.AssetName())
	f, err := os.Create(archive)
	if err != nil {
		return Compilers{}, err
	}
	_, err = download(ctx, id.URL(ReleasesURL), f)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return Compilers{}, err
	}
	layer := filepath.Join(dir, "layer")
	if err := extract(archive, layer); err != nil {
		return Compilers{}, err
	}
	return ReadCompilers(layer)
}

// checkXcode fails unless the selected Xcode is the pinned version.
func checkXcode(ctx context.Context, version string) error {
	out, err := exec.CommandContext(ctx, "xcodebuild", "-version").Output()
	if err != nil {
		return fmt.Errorf("xcodebuild -version: %w", err)
	}
	selected, _, _ := strings.Cut(string(out), "\n")
	if want := "Xcode " + version; selected != want {
		return fmt.Errorf("the libs layer is built with %s, but %s is selected; select it with DEVELOPER_DIR or xcode-select", want, selected)
	}
	return nil
}

// buildLayer runs build with a fresh layer prefix and work directory, then
// packs the prefix into archive.
func buildLayer(archive string, build func(prefix, work string) error) error {
	tmp, err := os.MkdirTemp("", "slipstream-layer-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	prefix := filepath.Join(tmp, "layer")
	if err := build(prefix, filepath.Join(tmp, "work")); err != nil {
		return err
	}
	return pack(prefix, archive)
}

// sourceDefines passes a source to a recipe as <NAME>_URL and
// <NAME>_SHA256, with the name upper-cased and dashes made underscores.
func sourceDefines(s Source) []string {
	id := strings.ToUpper(strings.ReplaceAll(s.Name, "-", "_"))
	return []string{"-D" + id + "_URL=" + s.URL, "-D" + id + "_SHA256=" + s.SHA256}
}

func downloadDir(cacheRoot string) string { return filepath.Join(cacheRoot, "downloads") }

func cmake(ctx context.Context, log io.Writer, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, "cmake", args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmake %s: %w", strings.Join(args, " "), err)
	}
	return nil
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
