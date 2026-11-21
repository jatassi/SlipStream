package atomicfile_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/atomicfile"
)

// holdExclusively opens path with no sharing, as an antivirus scan may, so
// a rename of it fails with a sharing violation until the handle closes.
func holdExclusively(t *testing.T, path string) syscall.Handle {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestRenameRetriesASharingViolation(t *testing.T) {
	dir := t.TempDir()
	from, to := filepath.Join(dir, "library.db.restore"), filepath.Join(dir, "library.db")
	if err := os.WriteFile(from, []byte("restored"), 0o600); err != nil {
		t.Fatal(err)
	}

	h := holdExclusively(t, from)
	if err := atomicfile.Rename(from, to, 0); err == nil {
		t.Fatal("rename with no retry succeeded while the file was held, so the test can't see a sharing violation")
	}

	done := make(chan error, 1)
	go func() { done <- atomicfile.Rename(from, to, 30*time.Second) }()
	select {
	case err := <-done:
		t.Fatalf("rename returned %v while the file was still held", err)
	case <-time.After(300 * time.Millisecond):
	}
	if err := syscall.CloseHandle(h); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("rename after the handle closed = %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("rename didn't finish within its retry after the handle closed")
	}
	if got := read(t, to); got != "restored" {
		t.Fatalf("%s = %q, want restored", to, got)
	}
}

func TestRemoveRetriesASharingViolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db.pre-restore")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	h := holdExclusively(t, path)
	done := make(chan error, 1)
	go func() { done <- atomicfile.Remove(path, 30*time.Second) }()
	time.Sleep(300 * time.Millisecond)
	if err := syscall.CloseHandle(h); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("remove after the handle closed = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stat after remove = %v, want not exist", err)
	}
}
