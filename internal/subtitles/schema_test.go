package subtitles_test

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestTheSchemaRefusesWhatNoRecordMeans(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, f := adopted(t)
		for _, c := range []struct{ name, stmt string }{
			{"a found Sidecar naming a provider", `UPDATE subtitles_sidecar SET provider = 'subdl'`},
			{"a tail with a folder", `UPDATE subtitles_sidecar SET tail = 'Subs/en.srt'`},
			{"a timed Sidecar with no match", `UPDATE subtitles_sidecar SET timed = 1`},
			{"a hash that isn't SHA-256", `UPDATE subtitles_sidecar SET sha256 = x'00'`},
			{"an audio list without its commas", `UPDATE subtitles_file SET audio = 'en'`},
		} {
			err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				_, err := tx.ExecContext(ctx, c.stmt)
				return err
			})
			if err == nil {
				t.Errorf("the schema took %s", c.name)
			}
		}
		err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return doom(ctx, tx, w, f, "Heat/Heat.mkv", nil, files.ID{})
		})
		if err != nil {
			t.Fatal(err)
		}
		err = w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `DELETE FROM files_file WHERE id = ?`, f)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"subtitles_file", "subtitles_want", "subtitles_sidecar"} {
			if got := query(t, w, `SELECT count(*) FROM `+table); got[0] != "0" {
				t.Errorf("%s keeps %s rows of a deleted file", table, got[0])
			}
		}
		if got := query(t, w, `SELECT count(*) FROM subtitles_removal`); got[0] != "2" {
			t.Errorf("the file's delete took its removal rows, %s left, want them kept for the Keeper", got[0])
		}
		if got := query(t, w, `SELECT "table" FROM pragma_foreign_key_check`); !slices.Equal(got, nil) {
			t.Errorf("foreign keys break in %v", got)
		}
	})
}
