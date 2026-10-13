package secret_test

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

const (
	fixedKey    = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f\n"
	fixedPepper = "335253ddf24bc6eec473a8bd5bf54a47d1e379b304e113a0b20b1c315c579601"
	otherKey    = "1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100\n"
)

func keyPath(t *testing.T) string { return filepath.Join(t.TempDir(), secret.KeyFile) }

func TestNoRowNoFileCreatesTheKey(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema())
	path := keyPath(t)
	first, err := secret.Open(t.Context(), lib, path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}\n$`).Match(b) {
		t.Fatalf("library.key = %q, want 64 lower-case hex characters and a newline", b)
	}
	if info, err := os.Stat(path); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("library.key mode = %v, %v; want 0600", info.Mode().Perm(), err)
	}
	again, err := secret.Open(t.Context(), lib, path)
	if err != nil {
		t.Fatalf("reopen with the key it created: %v", err)
	}
	if hex.EncodeToString(again.PINPepper()) != hex.EncodeToString(first.PINPepper()) {
		t.Fatal("reopening derived a different pepper")
	}
}

func TestNoRowAndAFileUsesTheFile(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema())
	path := keyPath(t)
	write(t, path, fixedKey)
	keys, err := secret.Open(t.Context(), lib, path)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(keys.PINPepper()); got != fixedPepper {
		t.Fatalf("PINPepper = %s, want %s", got, fixedPepper)
	}
	if hex.EncodeToString(keys.CredentialKey()) == fixedPepper {
		t.Fatal("CredentialKey equals PINPepper")
	}
	if _, err := secret.Open(t.Context(), lib, path); err != nil {
		t.Fatalf("the file did not write the check row: %v", err)
	}
}

func TestARowAndItsFileUsesIt(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema())
	path := keyPath(t)
	write(t, path, fixedKey)
	if _, err := secret.Open(t.Context(), lib, path); err != nil {
		t.Fatal(err)
	}
	keys, err := secret.Open(t.Context(), lib, path)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(keys.PINPepper()); got != fixedPepper {
		t.Fatalf("PINPepper = %s, want %s", got, fixedPepper)
	}
}

func TestARowWithoutItsFileIsRefused(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema())
	path := keyPath(t)
	if _, err := secret.Open(t.Context(), lib, path); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	_, err := secret.Open(t.Context(), lib, path)
	refused(t, err, "library.key is missing, but library.db was set up with one; restore library.key from the Backup that holds this library.db")
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused open wrote library.key: %v", err)
	}
}

func TestARowWithAForeignFileIsRefused(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema())
	path := keyPath(t)
	write(t, path, fixedKey)
	if _, err := secret.Open(t.Context(), lib, path); err != nil {
		t.Fatal(err)
	}
	write(t, path, otherKey)
	_, err := secret.Open(t.Context(), lib, path)
	refused(t, err, "library.key does not belong to this library.db; restore the library.key that does")
}

func TestOnlyTheExactKeyFormIsRead(t *testing.T) {
	text := strings.TrimSuffix(fixedKey, "\n")
	for _, tc := range []struct {
		name, content string
		malformed     bool
	}{
		{"exact", fixedKey, false},
		{"upper-case", strings.ToUpper(text) + "\n", true},
		{"no newline", text, true},
		{"trailing space", text + " \n", true},
		{"CRLF", text + "\r\n", true},
		{"leading space", " " + fixedKey, true},
		{"63 characters", text[:63] + "\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lib := storagetest.Library(t, secret.Schema())
			path := keyPath(t)
			write(t, path, tc.content)
			keys, err := secret.Open(t.Context(), lib, path)
			if tc.malformed {
				if err == nil || err.Error() != "library.key is not 64 hex characters" {
					t.Fatalf("err = %v, want library.key is not 64 hex characters", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := hex.EncodeToString(keys.PINPepper()); got != fixedPepper {
				t.Fatalf("PINPepper = %s, want %s", got, fixedPepper)
			}
		})
	}
}

func TestARowWithATruncatedFileIsRefused(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema())
	path := keyPath(t)
	write(t, path, fixedKey)
	if _, err := secret.Open(t.Context(), lib, path); err != nil {
		t.Fatal(err)
	}
	write(t, path, fixedKey[:40])
	_, err := secret.Open(t.Context(), lib, path)
	refused(t, err, "library.key does not belong to this library.db; restore the library.key that does")
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func refused(t *testing.T, err error, want string) {
	t.Helper()
	r, ok := errors.AsType[*storage.RefusedError](err)
	if !ok || r.Error() != want {
		t.Fatalf("err = %v, want *storage.RefusedError %q", err, want)
	}
}
