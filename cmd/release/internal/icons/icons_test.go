package icons

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheck(t *testing.T) {
	root := t.TempDir()
	for _, f := range files {
		write(t, root, f, f)
	}
	if err := Check(root); err == nil {
		t.Fatal("Check passed with no stamp")
	}
	if err := Record(root); err != nil {
		t.Fatal(err)
	}
	if err := Check(root); err != nil {
		t.Fatalf("after Record: %v", err)
	}
	stamp, err := os.ReadFile(filepath.Join(root, stampPath))
	if err != nil {
		t.Fatal(err)
	}
	const master = "bd9db6d6ce1abf117e33d7799c287cd21c84ce90d456999f82540039197432cc  brand/app-icon.svg\n"
	if got := string(stamp[:len(master)]); got != master {
		t.Errorf("stamp starts %q, want %q", got, master)
	}

	for _, f := range []string{"brand/app-icon.svg", "web/public/icon-512.png"} {
		t.Run(f, func(t *testing.T) {
			write(t, root, f, "changed")
			t.Cleanup(func() { write(t, root, f, f) })
			if err := Check(root); err == nil || err.Error() != "brand/icons.sha256 is stale; run go run ./cmd/release gen icons" {
				t.Fatalf("changed %s: %v", f, err)
			}
		})
	}
}
