package storing

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOSDiskWritesAtOffsetsAndReadsThemBack(t *testing.T) {
	ctx, d := t.Context(), newOSDisk(t)
	part := filepath.Join(t.TempDir(), "0.part")
	w, err := d.OpenWrite(ctx, part)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteAt([]byte("world"), 6); err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteAt([]byte("hello "), 0); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(w.Sync(), w.Close()); err != nil {
		t.Fatal(err)
	}
	if got := readAt(t, d, part, 6, 5); got != "world" {
		t.Errorf("ReadAt(6) = %q, want world", got)
	}
	w, err = d.OpenWrite(ctx, part)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := d.Stat(ctx, part); err != nil || info.Size() != 11 {
		t.Fatalf("a reopened file reads %v, %v; want the 11 bytes it held", info, err)
	}
	if _, err := w.WriteAt([]byte("W"), 6); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(w.Truncate(7), w.Close()); err != nil {
		t.Fatal(err)
	}
	if got := readAt(t, d, part, 0, 7); got != "hello W" {
		t.Errorf("the file reads %q, want %q", got, "hello W")
	}
	if info, err := d.Stat(ctx, part); err != nil || info.Size() != 7 {
		t.Errorf("a file truncated to 7 bytes reads %v, %v", info, err)
	}
}

func readAt(t *testing.T, d Disk, path string, off int64, n int) string {
	t.Helper()
	r, err := d.OpenAt(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, n)
	_, err = r.ReadAt(b, off)
	if err := errors.Join(err, r.Close()); err != nil {
		t.Fatalf("ReadAt(%s, %d): %v", path, off, err)
	}
	return string(b)
}

func TestOSDiskModTimeMovesOnWrite(t *testing.T) {
	ctx, d := t.Context(), newOSDisk(t)
	path := write(t, filepath.Join(t.TempDir(), "0.part"), "")
	past := time.Date(2001, 9, 9, 1, 46, 40, 0, time.UTC)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}
	if info, err := d.Stat(ctx, path); err != nil || !info.ModTime().Equal(past) {
		t.Fatalf("Stat = %v, %v; want modified at %v", info, err, past)
	}
	w, err := d.OpenWrite(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteAt([]byte("x"), 0); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := d.Stat(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().After(past) {
		t.Errorf("ModTime after a write = %v, want later than %v", info.ModTime(), past)
	}
}

func TestOSDiskFileIDFollowsTheFile(t *testing.T) {
	ctx, d, dir := t.Context(), newOSDisk(t), t.TempDir()
	a, b := write(t, filepath.Join(dir, "a.mkv"), "film"), write(t, filepath.Join(dir, "b.mkv"), "film")
	first, err := d.Stat(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if first.FileID() == nil {
		t.Fatal("a file on a local volume has no file id")
	}
	moved := filepath.Join(dir, "c.mkv")
	if err := d.Rename(ctx, a, moved); err != nil {
		t.Fatal(err)
	}
	again, err := d.Stat(ctx, moved)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again.FileID(), first.FileID()) {
		t.Errorf("the file id went from %x to %x across a rename", first.FileID(), again.FileID())
	}
	other, err := d.Stat(ctx, b)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(other.FileID(), first.FileID()) {
		t.Errorf("two files share the id %x", first.FileID())
	}
}

func TestOSDiskRemoveAllRemovesAFolderAndWhatItHolds(t *testing.T) {
	ctx, d, dir := t.Context(), newOSDisk(t), t.TempDir()
	staging := filepath.Join(dir, "transfer")
	write(t, filepath.Join(staging, "0.part"), "film")
	write(t, filepath.Join(staging, "nested", "1.part"), "film")
	if err := d.RemoveAll(ctx, staging); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Stat(ctx, staging); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat of a removed folder = %v, want not exist", err)
	}
	if err := d.RemoveAll(ctx, staging); err != nil {
		t.Errorf("RemoveAll of a missing folder = %v, want nil", err)
	}
}
