//go:build scale

package files_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
)

func TestScaleDeletingAThousandGrabsAmongAHundredThousandChildren(t *testing.T) {
	lib, s := openRetention(t)
	ids := make([]any, 0, 1000)
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i := range 2000 {
			grab, file := storage.NewUUID(), storage.NewUUID()
			records := 97
			if i%2 == 0 {
				ids, records = append(ids, grab), 1
			}
			err := exec(ctx, tx,
				st(seedGrab, grab, s.work, s.item, s.node),
				st(`INSERT INTO files_file (id, work_id, cut, root_id, path, size, probe, grab_id) VALUES (?, ?, 'standard', ?, ?, 1, '{}', ?)`,
					file, s.work, s.root, fmt.Sprintf("Orchard/%d.mkv", i), grab),
				st(`INSERT INTO imports_import (id, source, grab_id, node_id, info_hash, content_type, hint, created_at, finished_at)
					VALUES (?, 'grab', ?, ?, randomblob(20), 'fixture', '{}', 0, 1)`, storage.NewUUID(), grab, s.node),
				st(`WITH RECURSIVE n(k) AS (SELECT 1 UNION ALL SELECT k + 1 FROM n WHERE k < ?)
					INSERT INTO files_imported (id, file_id, grab_id, work_id, placed_by, items, cells, replaced, at)
					SELECT randomblob(16), ?, ?, ?, 'copy', '[]', '[]', '[]', k FROM n`, records, file, grab, s.work),
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var took time.Duration
	err = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		start := time.Now()
		_, err := tx.ExecContext(ctx, `DELETE FROM grabs_grab WHERE id IN (`+strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")+`)`, ids...)
		took = time.Since(start)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("deleted 1,000 Grabs among 100,000 children in %v", took)
	if took > 100*time.Millisecond {
		t.Fatalf("deleting 1,000 Grabs took %v, want under 100 ms", took)
	}
	distinct(t, lib, `SELECT CAST(count(*) AS TEXT) FROM imports_import WHERE grab_id IS NULL`, "1000")
}
