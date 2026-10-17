// Package secrettest gives tests a sealer over a library.key of their own.
package secrettest

import (
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func Sealer(t testing.TB) *secret.Sealer {
	t.Helper()
	keys, err := secret.Open(t.Context(), storagetest.Library(t, secret.Schema()), filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	return keys.Sealer()
}
