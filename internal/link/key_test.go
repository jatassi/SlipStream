package link_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/link"
)

func sequenceSeed() [32]byte {
	var seed [32]byte
	for i := range seed {
		seed[i] = byte(i)
	}
	return seed
}

func TestKeyText(t *testing.T) {
	k := link.NewPrivateKey(sequenceSeed()).Public()
	if got := [2]string{k.String(), k.Fingerprint()}; got != [2]string{"0eggffzksr8bw7bgvmceejy0k5ky9nhgkejgtqrxvj3684jn66w0", "0EGG-FFZK-SR8B-W7BG"} {
		t.Fatalf("text and fingerprint %q", got)
	}
	parsed, err := link.ParseKey("0eggffzksr8bw7bgvmceejy0k5ky9nhgkejgtqrxvj3684jn66w0")
	if err != nil || parsed != k {
		t.Fatalf("ParseKey = %v, %v", parsed, err)
	}
	if _, err := link.ParseKey("0EGGFFZKSR8BW7BGVMCEEJY0K5KY9NHGKEJGTQRXVJ3684JN66W0"); err == nil {
		t.Fatal("ParseKey read an upper-case key")
	}
}

func TestKeyFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	created, err := link.OpenKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "node.key"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 65 || b[64] != '\n' {
		t.Fatalf("node.key is %q, want 64 hex characters and a newline", b)
	}
	read, err := link.OpenKey(dir)
	if err != nil || read.Public() != created.Public() {
		t.Fatalf("reopened key %v, %v; created %v", read.Public(), err, created.Public())
	}

	known := "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f\n"
	if err := os.WriteFile(filepath.Join(dir, "node.key"), []byte(known), 0o600); err != nil {
		t.Fatal(err)
	}
	read, err = link.OpenKey(dir)
	if err != nil || read.Public() != link.NewPrivateKey(sequenceSeed()).Public() {
		t.Fatalf("OpenKey of a known seed = %v, %v", read.Public().Fingerprint(), err)
	}
}

func TestAMalformedKeyFileIsRefused(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node.key")
	if err := os.WriteFile(path, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := link.OpenKey(dir)
	if want := "node key " + path + " is not 64 hex characters"; err == nil || err.Error() != want {
		t.Fatalf("OpenKey = %v, want %s", err, want)
	}
}
