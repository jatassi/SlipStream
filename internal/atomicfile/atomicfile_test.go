package atomicfile_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/atomicfile"
)

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCreateNeverReplaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "record.json")
	if err := atomicfile.Create(path, []byte("first"), 0); err != nil {
		t.Fatal(err)
	}
	if err := atomicfile.Create(path, []byte("second"), 0); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("second create = %v, want fs.ErrExist", err)
	}
	if got := read(t, path); got != "first" {
		t.Fatalf("file = %q after a refused create, want first", got)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("dir holds %v, %v; want only record.json", entries, err)
	}
}

func TestReplaceReplaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "record.json")
	for _, data := range []string{"first", "second"} {
		if err := atomicfile.Replace(path, []byte(data), 0); err != nil {
			t.Fatal(err)
		}
	}
	if got := read(t, path); got != "second" {
		t.Fatalf("file = %q, want second", got)
	}
}

func TestRemoveOfAMissingFileIsNoError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone")
	if err := atomicfile.Remove(path, 0); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := atomicfile.Remove(path, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("stat after remove = %v, want not exist", err)
	}
}
