// Package secret owns library.key, the key file beside library.db, and the
// keys derived from it for each purpose.
package secret

import (
	"context"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jatassi/slipstream/internal/secret/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("secret", schemaFiles) }

// KeyFile is the key file's name in the data dir, beside library.db.
const KeyFile = "library.key"

const keySize = 32

// Keys holds library.key. Each purpose derives its own key from it, so two
// features never share a key by accident.
type Keys struct {
	key [keySize]byte
}

var (
	errMissing = errors.New("library.key is missing, but library.db was set up with one; restore library.key from the Backup that holds this library.db")
	errForeign = errors.New("library.key does not belong to this library.db; restore the library.key that does")
)

// Open converges the key file at path with library.db's check row. A key
// file that is missing or foreign to lib is a *storage.RefusedError.
func Open(ctx context.Context, lib *storage.DB[file], path string) (*Keys, error) {
	stored, err := db.New(lib).GetCheck(ctx)
	hasRow := err == nil
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return nil, err
	}
	k, err := readKey(path)
	switch {
	case errors.Is(err, fs.ErrNotExist) && hasRow:
		return nil, &storage.RefusedError{Err: errMissing}
	case errors.Is(err, fs.ErrNotExist):
		if k, err = createKey(path); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	}
	if hasRow {
		if subtle.ConstantTimeCompare(stored, k.check()) != 1 {
			return nil, &storage.RefusedError{Err: errForeign}
		}
		return k, nil
	}
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).InsertCheck(ctx, k.check())
	})
	if err != nil {
		return nil, err
	}
	return k, nil
}

// PINPepper keys the HMAC that every PIN passes through before argon2id. It
// never rotates: a new pepper would need every PIN again.
func (k *Keys) PINPepper() []byte { return k.derive("slipstream pin pepper") }

func (k *Keys) CredentialKey() []byte { return k.derive("slipstream credential key") }

func (k *Keys) check() []byte { return k.derive("slipstream key check") }

func (k *Keys) derive(info string) []byte {
	out, err := hkdf.Key(sha256.New, k.key[:], nil, info, keySize)
	if err != nil {
		panic(fmt.Sprintf("secret: derive %s: %v", info, err))
	}
	return out
}

func readKey(path string) (*Keys, error) {
	b, err := os.ReadFile(path) //nolint:gosec // path is the data dir's key file, chosen by the Node
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", KeyFile, err)
	}
	raw, err := hex.DecodeString(strings.TrimSuffix(string(b), "\n"))
	if err != nil || len(raw) != keySize {
		return nil, fmt.Errorf("%s is not %d hex characters", KeyFile, 2*keySize)
	}
	var k Keys
	copy(k.key[:], raw)
	return &k, nil
}

func createKey(path string) (*Keys, error) {
	var k Keys
	_, _ = rand.Read(k.key[:])
	tmp, err := os.CreateTemp(filepath.Dir(path), KeyFile+".*")
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", KeyFile, err)
	}
	_, err = tmp.WriteString(hex.EncodeToString(k.key[:]) + "\n")
	if err == nil {
		err = tmp.Sync()
	}
	err = errors.Join(err, tmp.Close())
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("create %s: %w", KeyFile, err), os.Remove(tmp.Name()))
	}
	return &k, nil
}
