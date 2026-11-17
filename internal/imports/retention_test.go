package imports

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

type keepDefault struct{}

func (keepDefault) Fields(context.Context) ([]setting.Field, error) { return nil, nil }
func (keepDefault) Apply(context.Context, setting.Patch) error      { return nil }
func (keepDefault) Keep(context.Context) (time.Duration, error)     { return history.KeepFor.Default, nil }

// seedImport is a folder Import of one file, made and finished ago, or
// unfinished when finished is false, of grab unless it is zero.
func (w *world) seedImport(t *testing.T, ago time.Duration, finished bool, grab storage.UUID) ImportID {
	t.Helper()
	id := ImportID{UUID: storage.NewUUID()}
	at := storage.FromTime(time.Now().Add(-ago))
	var done, of any
	if finished {
		done = at
	}
	source, folder := "folder", any("/downloads/Shelf")
	if !grab.IsZero() {
		source, folder, of = "grab", nil, grab
	}
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO imports_import (id, source, grab_id, node_id, info_hash, folder, content_type, hint, created_at, finished_at)
			VALUES (?, ?, ?, ?, iif(? = 'grab', randomblob(20), NULL), ?, 'fixture', '{}', ?, ?)`, id, source, of, w.node, source, folder, at, done); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO imports_file (import_id, position, path, size, cut, release_name, source_name, outcome)
			VALUES (?, 0, 'Shelf.mkv', 1, 'standard', 'Shelf', 'Shelf.mkv', 'pending')`, id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (w *world) exists(t *testing.T, id ImportID) bool {
	t.Helper()
	_, _, err := w.im.Get(t.Context(), identity.Admin{}, id)
	if err != nil && !errors.Is(err, ErrNoImport) {
		t.Fatal(err)
	}
	return err == nil
}

func TestAnOldFinishedImportGoesWithItsFilesUnlessItsGrabStays(t *testing.T) {
	w := newWorld(t)
	tree := w.add(t, "w1", w.root.ID)
	grab := storage.NewUUID()
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)
			ON CONFLICT DO NOTHING`, w.node)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet,
			stage, failure_code, failed_stage, failure_message, created_at, updated_at)
			VALUES (?, ?, ?, 'any', 'search', ?, 'Shelf.WEB-DL', 1, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567',
			'failed', 'import_failed', 'importing', 'The import failed.', 0, 0)`, grab, tree.ID, tree.Items[0].ID, w.node)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	old := history.KeepFor.Default + 24*time.Hour
	gone := w.seedImport(t, old, true, storage.UUID{})
	ofGrab := w.seedImport(t, old, true, grab)
	open := w.seedImport(t, old, false, storage.UUID{})
	young := w.seedImport(t, time.Hour, true, storage.UUID{})

	task := housekeeping.Task(housekeeping.Config{Keep: keepDefault{}, Batch: 1, Steps: []housekeeping.Step{w.im.Retention()}})
	if err := task.Run(t.Context(), tasks.Unit{Key: "finished-imports"}, &tasks.Progress{}); err != nil {
		t.Fatal(err)
	}
	if w.exists(t, gone) {
		t.Fatal("an old finished Import with no Grab stays")
	}
	for what, id := range map[string]ImportID{"whose Grab is kept": ofGrab, "unfinished": open, "finished an hour ago": young} {
		if !w.exists(t, id) {
			t.Errorf("an Import %s went", what)
		}
	}
	var files int
	if err := w.lib.QueryRowContext(t.Context(), `SELECT count(*) FROM imports_file WHERE import_id = ?`, gone).Scan(&files); err != nil || files != 0 {
		t.Fatalf("the deleted Import left %d files (%v), want none", files, err)
	}
}
