package works_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

func TestTheAddDefaultKeepsItsSearchThroughOtherRememberedChoices(t *testing.T) {
	lib := storagetest.Library(t, works.Schema())
	write := func(f func(ctx context.Context, tx *storage.Tx[storage.Library]) error) error {
		return lib.Write(t.Context(), f)
	}
	err := write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO works_add_default (content_type, search) VALUES ('fixture', 'first_level')`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	err = write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return works.RememberIn(ctx, tx, "fixture", works.Remembering{Preset: new(works.PresetFirst), LevelFolders: new(false)})
	})
	if err != nil {
		t.Fatal(err)
	}
	var got string
	if err := lib.QueryRowContext(t.Context(), `SELECT monitor || ' ' || level_folders || ' ' || search FROM works_add_default`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := "first 0 first_level"; got != want {
		t.Errorf("the add default reads %q, want %q", got, want)
	}

	for _, c := range []struct {
		search any
		ok     bool
	}{{nil, true}, {"monitored", true}, {"latest_level", true}, {"first_item", true}, {"none", true}, {"everything", false}} {
		err := write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `UPDATE works_add_default SET search = ?`, c.search)
			return err
		})
		if c.ok != (err == nil) || !c.ok && !strings.Contains(err.Error(), "CHECK constraint failed") {
			t.Errorf("an add search of %v: written with %v, want ok=%v", c.search, err, c.ok)
		}
	}
}
