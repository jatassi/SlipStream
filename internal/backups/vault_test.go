package backups_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

var (
	oldKey = bytes.Repeat([]byte{0x0a}, 32)
	newKey = bytes.Repeat([]byte{0x0b}, 32)
)

func keys(t *testing.T) *secret.Keys {
	t.Helper()
	k, err := secret.Open(t.Context(), storagetest.Library(t, secret.Schema()), filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func open(t *testing.T, path string, k *secret.Keys) *backups.Vault {
	t.Helper()
	v, err := backups.OpenVault(path, k, 0)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func vaultPath(t *testing.T) string { return filepath.Join(t.TempDir(), backups.VaultFile) }

func current(t *testing.T, v *backups.Vault) (g backups.Generation, words string) {
	t.Helper()
	g, p, err := v.Current(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return g, p.Reveal()
}

func replace(t *testing.T, v *backups.Vault) backups.Generation {
	t.Helper()
	g, err := v.Replace(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func bytesOf(t *testing.T, v *backups.Vault) []byte {
	t.Helper()
	b, err := v.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// reopen writes b to its own backup.key and opens it, as a restore's swap
// leaves it.
func reopen(t *testing.T, b []byte, k *secret.Keys) *backups.Vault {
	t.Helper()
	path := vaultPath(t)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return open(t, path, k)
}

func TestAMissingVaultMakesGenerationOne(t *testing.T) {
	k := keys(t)
	path := vaultPath(t)
	v := open(t, path, k)
	if gens := v.Generations(); len(gens) != 0 || v.Fence().Generation != 0 {
		t.Fatalf("a missing backup.key opened with generations %v and fence %d", gens, v.Fence().Generation)
	}
	if v.Proven(oldKey, newKey, backups.Prove(v.Fence(), oldKey, newKey)) {
		t.Fatal("a Vault with no fence secret proved a proof")
	}
	if _, err := v.Bytes(); err == nil {
		t.Fatal("an empty Vault has bytes for an archive")
	}
	g, pass := current(t, v)
	if g != 1 || len(bytes.Fields([]byte(pass))) != 10 || v.Fence().Generation != 1 {
		t.Fatalf("Current = generation %d, %d words, fence %d; want generation 1, 10 words, fence 1", g, len(bytes.Fields([]byte(pass))), v.Fence().Generation)
	}
	again := open(t, path, k)
	if g2, pass2 := current(t, again); g2 != 1 || pass2 != pass {
		t.Fatal("reopening backup.key gave another passphrase")
	}
	if !again.Proven(oldKey, newKey, backups.Prove(v.Fence(), oldKey, newKey)) {
		t.Fatal("reopening backup.key changed the fence secret")
	}
}

func TestReplaceAddsAGenerationAndANewFence(t *testing.T) {
	k := keys(t)
	path := vaultPath(t)
	v := open(t, path, k)
	_, first := current(t, v)
	if g := replace(t, v); g != 2 {
		t.Fatalf("Replace made generation %d, want 2", g)
	}
	g, second := current(t, v)
	if g != 2 || second == first || v.Fence().Generation != 2 {
		t.Fatalf("after Replace, Current = generation %d and fence %d; want a new passphrase at 2 and fence 2", g, v.Fence().Generation)
	}
	reopened := open(t, path, k)
	if old, ok := reopened.Passphrase(1); !ok || old.Reveal() != first {
		t.Fatal("Replace lost generation 1, which older Backups need")
	}
	if gens := reopened.Generations(); !slices.Equal(gens, []backups.Generation{2, 1}) {
		t.Fatalf("Generations = %v, want [2 1]", gens)
	}
	if err := reopened.Drop(func(backups.Generation) bool { return false }); err != nil {
		t.Fatal(err)
	}
	if gens := open(t, path, k).Generations(); !slices.Equal(gens, []backups.Generation{2}) {
		t.Fatalf("after dropping every unused generation, Generations = %v, want the current one", gens)
	}
}

func TestAProofFromAnEarlierFenceIsNotProven(t *testing.T) {
	v := open(t, vaultPath(t), keys(t))
	current(t, v)
	for name, step := range map[string]func() error{
		"Replace":         func() error { _, err := v.Replace(rand.Reader); return err },
		"RegenerateFence": func() error { return v.RegenerateFence(rand.Reader) },
	} {
		t.Run(name, func(t *testing.T) {
			proof := backups.Prove(v.Fence(), oldKey, newKey)
			if !v.Proven(oldKey, newKey, proof) {
				t.Fatal("the Vault didn't prove its own proof")
			}
			if err := step(); err != nil {
				t.Fatal(err)
			}
			if v.Proven(oldKey, newKey, proof) {
				t.Fatalf("a proof from before %s is still proven", name)
			}
		})
	}
}

func TestAForgedProofIsNotProven(t *testing.T) {
	v := open(t, vaultPath(t), keys(t))
	current(t, v)
	other := open(t, vaultPath(t), keys(t))
	current(t, other)
	proof := backups.Prove(v.Fence(), oldKey, newKey)
	flipped := slices.Clone(proof)
	flipped[10] ^= 1
	for name, forged := range map[string][]byte{
		"another Vault's secret": backups.Prove(other.Fence(), oldKey, newKey),
		"a flipped bit":          flipped,
		"cut short":              proof[:35],
		"empty":                  nil,
		"the generation alone":   proof[:4],
	} {
		if v.Proven(oldKey, newKey, forged) {
			t.Errorf("%s is proven", name)
		}
	}
	if v.Proven(oldKey, oldKey, proof) || v.Proven(newKey, newKey, proof) || v.Proven(newKey, oldKey, proof) {
		t.Error("a proof for one pair of keys proved another")
	}
}

func TestMergeKeepsCurrentGenerationsAndFenceAndAddsArchiveGenerations(t *testing.T) {
	k := keys(t)
	live := open(t, vaultPath(t), k)
	_, first := current(t, live)
	replace(t, live)
	_, second := current(t, live)
	archived := bytesOf(t, live)
	archiveFence := live.Fence()
	replace(t, live)
	_, third := current(t, live)
	if err := live.Drop(func(g backups.Generation) bool { return g == 2 }); err != nil {
		t.Fatal(err)
	}

	merged, err := backups.MergeVault(live, archived, k)
	if err != nil {
		t.Fatal(err)
	}
	v := reopen(t, merged, k)
	if gens := v.Generations(); !slices.Equal(gens, []backups.Generation{3, 2, 1}) {
		t.Fatalf("merged generations = %v, want [3 2 1]", gens)
	}
	for g, want := range map[backups.Generation]string{1: first, 2: second, 3: third} {
		if p, ok := v.Passphrase(g); !ok || p.Reveal() != want {
			t.Fatalf("merged generation %d isn't the one it was", g)
		}
	}
	if !v.Proven(oldKey, newKey, backups.Prove(live.Fence(), oldKey, newKey)) || v.Fence().Generation != 3 {
		t.Fatal("the merge didn't keep the current fence secret")
	}
	if v.Proven(oldKey, newKey, backups.Prove(archiveFence, oldKey, newKey)) {
		t.Fatal("the merge took the archive's older fence secret")
	}

	t.Run("no current backup.key", func(t *testing.T) {
		merged, err := backups.MergeVault(open(t, vaultPath(t), k), archived, k)
		if err != nil {
			t.Fatal(err)
		}
		v := reopen(t, merged, k)
		if gens := v.Generations(); !slices.Equal(gens, []backups.Generation{2, 1}) {
			t.Fatalf("merged generations = %v, want the archive's [2 1]", gens)
		}
		if !v.Proven(oldKey, newKey, backups.Prove(archiveFence, oldKey, newKey)) {
			t.Fatal("with no current backup.key the merge didn't keep the archive's fence secret")
		}
	})
}

func TestVaultForNewMachineHasAFreshFenceAtTheNextGeneration(t *testing.T) {
	k := keys(t)
	old := open(t, vaultPath(t), k)
	_, first := current(t, old)
	replace(t, old)
	_, second := current(t, old)
	archived := bytesOf(t, old)

	fresh, err := backups.VaultForNewMachine(archived, k, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	v := reopen(t, fresh, k)
	for g, want := range map[backups.Generation]string{1: first, 2: second} {
		if p, ok := v.Passphrase(g); !ok || p.Reveal() != want {
			t.Fatalf("the new machine lost passphrase generation %d", g)
		}
	}
	if got, want := v.Fence().Generation, old.Fence().Generation+1; got != want {
		t.Fatalf("fence generation = %d, want %d", got, want)
	}
	if v.Proven(oldKey, newKey, backups.Prove(old.Fence(), oldKey, newKey)) {
		t.Fatal("a proof from the archive's fence secret is proven on the new machine")
	}
	if !old.Proven(oldKey, newKey, backups.Prove(old.Fence(), oldKey, newKey)) {
		t.Fatal("the old Controller no longer proves a proof minted from its own Backup")
	}
}

// TestAVaultOpensAfterVacuumInto carries library.db, library.key and
// backup.key the way a Backup does and opens the Vault against the copy.
func TestAVaultOpensAfterVacuumInto(t *testing.T) {
	ctx := t.Context()
	from, to := t.TempDir(), t.TempDir()
	lib, k := openLibrary(t, from)
	v := open(t, filepath.Join(from, backups.VaultFile), k)
	g, pass := current(t, v)

	raw, err := sql.Open("sqlite", filepath.Join(from, storage.LibraryFile))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, "VACUUM INTO ?", filepath.Join(to, storage.LibraryFile)); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	_ = lib.Close()
	for _, name := range []string{secret.KeyFile, backups.VaultFile} {
		b, err := os.ReadFile(filepath.Join(from, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(to, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	_, copied := openLibrary(t, to)
	restored := open(t, filepath.Join(to, backups.VaultFile), copied)
	if p, ok := restored.Passphrase(g); !ok || p.Reveal() != pass {
		t.Fatal("the copied Vault doesn't open its passphrase")
	}
	if !restored.Proven(oldKey, newKey, backups.Prove(v.Fence(), oldKey, newKey)) {
		t.Fatal("the copied Vault lost its fence secret")
	}
	if _, err := backups.OpenVault(filepath.Join(to, backups.VaultFile), keys(t), 0); err == nil {
		t.Fatal("backup.key opened under another library.key")
	}
}

func openLibrary(t *testing.T, dir string) (*storage.DB[storage.Library], *secret.Keys) {
	t.Helper()
	lib, err := storage.OpenLibrary(t.Context(), filepath.Join(dir, storage.LibraryFile), []storage.Schema[storage.Library]{secret.Schema()}, noBackup, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })
	k, err := secret.Open(t.Context(), lib, filepath.Join(dir, secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	return lib, k
}

func noBackup(context.Context, *storage.Unmigrated, storage.Plan) error { return nil }
