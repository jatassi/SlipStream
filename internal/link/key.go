package link

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/internal/atomicfile"
)

// KeyFile is the Node key's file in the data dir, outside both databases
// and every Backup.
const KeyFile = "node.key"

// Key is a Node's raw Ed25519 public key, the value peers pin.
type Key [ed25519.PublicKeySize]byte

var keyText = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding)

// KeyFromBytes reads a key as the wire and the database hold it.
func KeyFromBytes(b []byte) (Key, error) {
	if len(b) != len(Key{}) {
		return Key{}, fmt.Errorf("a Node key is %d bytes, not %d", len(Key{}), len(b))
	}
	return Key(b), nil
}

// ParseKey reads the text form String writes.
func ParseKey(s string) (Key, error) {
	b, err := keyText.DecodeString(s)
	if err != nil || len(b) != len(Key{}) || keyText.EncodeToString(b) != s {
		return Key{}, fmt.Errorf("%q is not a Node key", s)
	}
	return Key(b), nil
}

// String is unpadded lower-case Crockford base32.
func (k Key) String() string { return keyText.EncodeToString(k[:]) }

// Fingerprint names a key in logs and the Join flow: its text form's first
// 16 characters, upper-cased in fours.
func (k Key) Fingerprint() string {
	s := strings.ToUpper(k.String()[:16])
	return s[:4] + "-" + s[4:8] + "-" + s[8:12] + "-" + s[12:]
}

func (k Key) MarshalText() ([]byte, error) { return []byte(k.String()), nil }

func (k *Key) UnmarshalText(text []byte) error {
	parsed, err := ParseKey(string(text))
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}

type PrivateKey struct {
	key ed25519.PrivateKey
}

// NewPrivateKey is the key whose seed node.key holds.
func NewPrivateKey(seed [ed25519.SeedSize]byte) PrivateKey {
	return PrivateKey{ed25519.NewKeyFromSeed(seed[:])}
}

// Public is the key's second half, after the seed.
func (p PrivateKey) Public() Key { return Key(p.key[ed25519.SeedSize:]) }

// MalformedKeyError is a node.key whose content is not a key.
type MalformedKeyError struct {
	Path string
}

func (e *MalformedKeyError) Error() string {
	return fmt.Sprintf("node key %s is not 64 hex characters", e.Path)
}

// UnreadableKeyError is a node.key that exists but cannot be read.
type UnreadableKeyError struct{ Err error }

func (e *UnreadableKeyError) Error() string { return "read node key: " + e.Err.Error() }

func (e *UnreadableKeyError) Unwrap() error { return e.Err }

// OpenKey reads dir's node.key: exactly 64 lower-case hex characters and
// one newline, holding the 32-byte seed. A Node's first start creates it
// from crypto/rand.
func OpenKey(dir string) (PrivateKey, error) {
	path := filepath.Join(dir, KeyFile)
	b, err := os.ReadFile(path) //nolint:gosec // path is the data dir's key file, chosen by the Node
	if errors.Is(err, fs.ErrNotExist) {
		return createKey(path)
	}
	if err != nil {
		return PrivateKey{}, &UnreadableKeyError{Err: err}
	}
	seed, err := hex.DecodeString(strings.TrimSuffix(string(b), "\n"))
	if err != nil || len(seed) != ed25519.SeedSize || hex.EncodeToString(seed)+"\n" != string(b) {
		return PrivateKey{}, &MalformedKeyError{Path: path}
	}
	return NewPrivateKey([ed25519.SeedSize]byte(seed)), nil
}

func createKey(path string) (PrivateKey, error) {
	var seed [ed25519.SeedSize]byte
	_, _ = rand.Read(seed[:])
	if err := atomicfile.Write(path, []byte(hex.EncodeToString(seed[:])+"\n")); err != nil {
		return PrivateKey{}, fmt.Errorf("create node key: %w", err)
	}
	return NewPrivateKey(seed), nil
}
