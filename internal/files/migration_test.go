package files_test

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/storage"
)

var frames = [...]media.Video{{Width: 3840, Height: 2160}, {Width: 3840, Height: 1600}, {Width: 1920, Height: 1080}, {Width: 1280, Height: 720}, {Width: 720, Height: 480}}

// opener opens a library file of t's, migrating it to schemas.
func opener(t *testing.T) func(schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	return func(schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(t.Context(), path, schemas, func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil },
			storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
}

type seeds struct {
	node, indexer, work, root storage.UUID
}

// seedLibrary adds a Node, an Indexer on it, a Work and a root, for seedFile.
func seedLibrary(ctx context.Context, tx *storage.Tx[storage.Library]) (seeds, error) {
	at := seeds{storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()}
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, []any{at.node}},
		{`INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, []any{at.indexer, at.node}},
		{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, []any{at.work}},
		{`INSERT INTO roots_root VALUES (?, 'fixture', ?, '/media')`, []any{at.root, at.node}},
	} {
		if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
			return seeds{}, fmt.Errorf("%s: %w", stmt.sql, err)
		}
	}
	return at, nil
}

// seedFile adds the nth Item of the Work and a file of it probed at probe,
// brought by a Grab of title unless title is empty, and returns the
// file's path.
func seedFile(ctx context.Context, tx *storage.Tx[storage.Library], n int, title string, probe media.Video, at seeds) (string, error) {
	item, file := storage.NewUUID(), storage.NewUUID()
	if _, err := tx.ExecContext(ctx, `INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, ?, 'Item', ?)`, item, at.work, strconv.Itoa(n), n); err != nil {
		return "", err
	}
	var grab any
	if title != "" {
		id := storage.NewUUID()
		_, err := tx.ExecContext(ctx, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size,
			release_link, stage, created_at, updated_at) VALUES (?, ?, ?, 'standard', 'search', ?, ?, ?, 1, 'https://site.test/dl', 'fetching', 0, 0)`,
			id, at.work, item, at.indexer, at.node, title)
		if err != nil {
			return "", err
		}
		grab = id
	}
	probeJSON := fmt.Sprintf(`{"width":%d,"height":%d}`, probe.Width, probe.Height)
	if _, err := tx.ExecContext(ctx, `INSERT INTO files_file VALUES (?, ?, ?, 'standard', ?, ?, 1, ?, ?)`, file, at.work, item, at.root, file.String(), probeJSON, grab); err != nil {
		return "", err
	}
	return file.String(), nil
}

// distinct checks the one value query reads in lib is want.
func distinct(t *testing.T, lib *storage.DB[storage.Library], query, want string) {
	t.Helper()
	var got []string
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := r.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				return fmt.Errorf("read a value: %w", err)
			}
			got = append(got, v)
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("%s = %v after the migration, want only %s", query, got, want)
	}
}
