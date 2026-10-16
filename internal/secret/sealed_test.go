package secret_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

const (
	vectorRow    = "0190f3c2-0000-7000-8000-000000000001"
	vectorSealed = "01297db7cc853bf4176e11597bdf25556f02a5746592d009e27acd4ad40c2ff384dcf46aee78674241eafcfe678542b4787e9bc4"
)

var valueColumn = secret.NewColumn("sealtest_row.value_sealed")

func sealerFor(t *testing.T, key string) *secret.Sealer {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema())
	path := keyPath(t)
	write(t, path, key)
	keys, err := secret.Open(t.Context(), lib, path)
	if err != nil {
		t.Fatal(err)
	}
	return keys.Sealer()
}

func sealedOf(t *testing.T, hexed string) secret.Sealed {
	t.Helper()
	raw, err := hex.DecodeString(hexed)
	if err != nil {
		t.Fatal(err)
	}
	var s secret.Sealed
	if err := s.Scan(raw); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestTheFixedVectorOpensOnlyInItsRowUnderItsKey(t *testing.T) {
	s := sealerFor(t, fixedKey)
	v := sealedOf(t, vectorSealed)
	got, err := s.Open(v, valueColumn.Row(vectorRow))
	if err != nil || string(got) != "torznab-key" {
		t.Fatalf("Open = %q, %v; want torznab-key", got, err)
	}
	refusals := map[string]func() ([]byte, error){
		"another row":    func() ([]byte, error) { return s.Open(v, valueColumn.Row("0190f3c2-0000-7000-8000-000000000002")) },
		"another column": func() ([]byte, error) { return s.Open(v, secret.NewColumn("sealtest_row.other_sealed").Row(vectorRow)) },
		"another key":    func() ([]byte, error) { return sealerFor(t, otherKey).Open(v, valueColumn.Row(vectorRow)) },
		"another version": func() ([]byte, error) {
			return s.Open(sealedOf(t, "02"+vectorSealed[2:]), valueColumn.Row(vectorRow))
		},
		"a truncated value": func() ([]byte, error) { return s.Open(sealedOf(t, vectorSealed[:40]), valueColumn.Row(vectorRow)) },
	}
	for name, open := range refusals {
		if got, err := open(); !errors.Is(err, secret.ErrUnsealable) {
			t.Errorf("%s: Open = %q, %v; want ErrUnsealable", name, got, err)
		}
	}
}

func TestSealingDrawsAFreshNonceEachTime(t *testing.T) {
	s := sealerFor(t, fixedKey)
	ad := valueColumn.Single()
	a, b := s.Seal([]byte("hunter2"), ad), s.Seal([]byte("hunter2"), ad)
	av, _ := a.Value()
	bv, _ := b.Value()
	if bytes.Equal(av.([]byte), bv.([]byte)) {
		t.Fatal("two seals of one value are the same bytes")
	}
	for _, v := range []secret.Sealed{a, b} {
		if got, err := s.Open(v, ad); err != nil || string(got) != "hunter2" {
			t.Fatalf("Open = %q, %v; want hunter2", got, err)
		}
	}
}

func TestASealedValueLogsMasked(t *testing.T) {
	var out bytes.Buffer
	slog.New(slog.NewTextHandler(&out, nil)).Info("saved", "key", sealedOf(t, vectorSealed))
	if want := "key=[sealed]"; !bytes.Contains(out.Bytes(), []byte(want)) {
		t.Fatalf("log = %q, want it to hold %q", out.String(), want)
	}
}

func TestTheZeroSealedIsNULL(t *testing.T) {
	var s secret.Sealed
	if v, err := s.Value(); v != nil || err != nil || s.Set() {
		t.Fatalf("zero Sealed = %v, %v, set %v; want NULL", v, err, s.Set())
	}
	if err := s.Scan(nil); err != nil || s.Set() {
		t.Fatalf("Scan(nil) = %v, set %v", err, s.Set())
	}
}

func TestNewColumnRefusesAColumnNotEndingInSealed(t *testing.T) {
	defer func() {
		if got, want := fmt.Sprint(recover()), `secret: "mail_settings.password" is not <table>.<column>_sealed`; got != want {
			t.Fatalf("panic = %q, want %q", got, want)
		}
	}()
	secret.NewColumn("mail_settings.password")
}

var sealtestSchema = storage.NewSchema[storage.Library]("sealtest", fstest.MapFS{
	"schema/00001_sealtest.sql": {Data: []byte("-- +goose Up\nCREATE TABLE sealtest_row (id TEXT PRIMARY KEY NOT NULL, value_sealed BLOB) STRICT;\n")},
})

// TestASealedValueOpensAfterVacuumInto copies a Library the way a Backup
// does and reads its sealed values back, so they never depend on the
// rowid VACUUM INTO may renumber.
func TestASealedValueOpensAfterVacuumInto(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	keyFile := filepath.Join(dir, secret.KeyFile)
	write(t, keyFile, fixedKey)
	schemas := []storage.Schema[storage.Library]{secret.Schema(), sealtestSchema}
	open := func(path string) (*storage.DB[storage.Library], *secret.Sealer) {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, schemas, noBackup, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = lib.Close() })
		keys, err := secret.Open(ctx, lib, keyFile)
		if err != nil {
			t.Fatal(err)
		}
		return lib, keys.Sealer()
	}
	original := filepath.Join(dir, storage.LibraryFile)
	lib, s := open(original)
	ids := []string{"c", "a", "b"}
	err := lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, id := range ids {
			if _, err := tx.ExecContext(ctx, "INSERT INTO sealtest_row (id, value_sealed) VALUES (?, ?)", id, s.Seal([]byte("key "+id), valueColumn.Row(id))); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM sealtest_row WHERE id = 'c'")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", original)
	if err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(dir, "copy.db")
	if _, err := raw.ExecContext(ctx, "VACUUM INTO ?", copied); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	restored, s := open(copied)
	for _, id := range ids[1:] {
		var v secret.Sealed
		if err := restored.QueryRowContext(ctx, "SELECT value_sealed FROM sealtest_row WHERE id = ?", id).Scan(&v); err != nil {
			t.Fatal(err)
		}
		if got, err := s.Open(v, valueColumn.Row(id)); err != nil || string(got) != "key "+id {
			t.Fatalf("row %s opens to %q, %v; want key %s", id, got, err, id)
		}
	}
}

func noBackup(context.Context, *storage.Unmigrated, storage.Plan) error { return nil }
