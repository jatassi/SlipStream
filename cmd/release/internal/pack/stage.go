package pack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/internal/install"
)

// NoticesPath is the notices file the binary embeds, which every package
// also ships beside it.
const NoticesPath = "internal/notices/NOTICES.txt"

func (in Input) programs() []string {
	names := []string{native.Program(in.Target, install.Executable)}
	if in.Target.GOOS == "windows" || in.Target.GOOS == "darwin" {
		names = append(names, native.Program(in.Target, install.Trampoline))
	}
	for _, tool := range in.Inputs.Tools {
		for _, p := range tool.Programs {
			names = append(names, native.Program(in.Target, p))
		}
	}
	return names
}

func (in Input) stagePrograms(dir string, k install.Kind) error {
	for _, name := range in.programs() {
		if err := copyFile(filepath.Join(in.Dist, name), filepath.Join(dir, name), 0o755); err != nil {
			return err
		}
	}
	return in.writeMarker(filepath.Join(dir, native.Program(in.Target, install.Executable)), k)
}

func (in Input) writeMarker(exe string, k install.Kind) error {
	l := install.Layout{GOOS: in.Target.GOOS, Executable: filepath.ToSlash(exe)}
	marker := filepath.FromSlash(strings.ReplaceAll(install.MarkerPath(l), `\`, "/"))
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		return fmt.Errorf("stage the marker: %w", err)
	}
	return install.Write(marker, k)
}

func (in Input) stageLegal(dir string) error {
	if err := copyFile(filepath.Join(in.Root, "LICENSE"), filepath.Join(dir, "LICENSE"), 0o644); err != nil {
		return err
	}
	return copyFile(filepath.Join(in.Root, filepath.FromSlash(NoticesPath)), filepath.Join(dir, "NOTICES.txt"), 0o644)
}

func copyFile(src, dst string, perm fs.FileMode) error {
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("stage %s: %w", filepath.Base(src), err)
	}
	defer func() { _ = r.Close() }()
	return writeFile(dst, r, perm)
}

func writeFile(dst string, r io.Reader, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("stage %s: %w", filepath.Base(dst), err)
	}
	w, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("stage %s: %w", filepath.Base(dst), err)
	}
	_, err = io.Copy(w, r)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("stage %s: %w", filepath.Base(dst), err)
	}
	// The umask must not decide a packaged file's mode.
	if err := os.Chmod(dst, perm); err != nil {
		return fmt.Errorf("stage %s: %w", filepath.Base(dst), err)
	}
	return nil
}

func writeText(dst, text string, perm fs.FileMode) error {
	return writeFile(dst, strings.NewReader(text), perm)
}

func walkStage(root string, visit func(rel string, info fs.FileInfo) error) error {
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == root {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return fmt.Errorf("walk %s: %w", p, err)
		}
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("walk %s: %w", p, err)
		}
		return visit(filepath.ToSlash(rel), info)
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", root, err)
	}
	return nil
}

// archiveMode is the mode an archive gives each entry, set here because a
// Windows host stages no exec bit: directories and programs are 0755,
// everything else 0644.
func archiveMode(rel string, info fs.FileInfo, programs []string) fs.FileMode {
	if info.IsDir() || slices.Contains(programs, path.Base(rel)) {
		return 0o755
	}
	return 0o644
}

func writeTarGz(archive, root string, programs []string) error {
	f, err := os.Create(archive)
	if err != nil {
		return fmt.Errorf("create %s: %w", filepath.Base(archive), err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	err = walkStage(root, func(rel string, info fs.FileInfo) error {
		hdr := &tar.Header{Name: rel, Mode: int64(archiveMode(rel, info, programs)), ModTime: info.ModTime(), Format: tar.FormatPAX}
		if info.IsDir() {
			hdr.Typeflag, hdr.Name = tar.TypeDir, rel+"/"
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("archive %s: %w", rel, err)
			}
			return nil
		}
		hdr.Typeflag, hdr.Size = tar.TypeReg, info.Size()
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("archive %s: %w", rel, err)
		}
		return copyInto(tw, filepath.Join(root, filepath.FromSlash(rel)))
	})
	for _, c := range []io.Closer{tw, gz, f} {
		if closeErr := c.Close(); err == nil {
			err = closeErr
		}
	}
	if err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(archive), err)
	}
	return nil
}

func writeZip(archive, root string, programs []string) error {
	f, err := os.Create(archive)
	if err != nil {
		return fmt.Errorf("create %s: %w", filepath.Base(archive), err)
	}
	zw := zip.NewWriter(f)
	err = walkStage(root, func(rel string, info fs.FileInfo) error {
		hdr := &zip.FileHeader{Name: rel, Modified: info.ModTime(), Method: zip.Deflate}
		mode := archiveMode(rel, info, programs)
		if info.IsDir() {
			hdr.Name, hdr.Method, mode = rel+"/", zip.Store, mode|fs.ModeDir
		}
		hdr.SetMode(mode)
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return fmt.Errorf("archive %s: %w", rel, err)
		}
		if info.IsDir() {
			return nil
		}
		return copyInto(w, filepath.Join(root, filepath.FromSlash(rel)))
	})
	for _, c := range []io.Closer{zw, f} {
		if closeErr := c.Close(); err == nil {
			err = closeErr
		}
	}
	if err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(archive), err)
	}
	return nil
}

func copyInto(w io.Writer, file string) error {
	r, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("archive %s: %w", filepath.Base(file), err)
	}
	defer func() { _ = r.Close() }()
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("archive %s: %w", filepath.Base(file), err)
	}
	return nil
}

func (in Input) run(ctx context.Context, dir string, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.Stdout, cmd.Stderr = in.Log, in.Log
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
