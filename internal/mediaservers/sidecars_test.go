package mediaservers_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

type row struct {
	kind, path, from string
	file             storage.UUID
}

func TestASidecarChangeIsItsOwnRowBesideItsVideosOpenCarry(t *testing.T) {
	for _, c := range []struct {
		kind mediaservers.SidecarKind
		want string
	}{
		{mediaservers.SidecarAdded, "added"},
		{mediaservers.SidecarReplaced, "replaced"},
		{mediaservers.SidecarRemoved, "removed"},
	} {
		t.Run(c.want, func(t *testing.T) {
			f := seed(t)
			must(t, f.lib, `DELETE FROM mediaservers_change`)
			f.change(t, "moved", "New/Heat.mkv", "Old/Heat.mkv", "due", "none")
			ms := mediaservers.New(mediaservers.Config{Library: f.lib, Log: slog.New(slog.DiscardHandler)})
			sidecar := storage.NewUUID()
			err := f.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				return ms.SidecarsChanged(ctx, tx, []mediaservers.SidecarChange{
					{Kind: c.kind, Root: roots.ID{UUID: f.root}, Path: mustRel(t, "New/Heat.en.srt"), Sidecar: sidecar},
				})
			})
			if err != nil {
				t.Fatal(err)
			}
			got := f.rows(t)
			want := []row{{"moved", "New/Heat.mkv", "Old/Heat.mkv", f.file}, {c.want, "New/Heat.en.srt", "", sidecar}}
			if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("after a Sidecar %s beside a renamed video the queue holds %+v, want %+v", c.want, got, want)
			}
		})
	}
}

// rows is every queued change, its kind, paths and file, by path.
func (f fixture) rows(t *testing.T) []row {
	t.Helper()
	rs, err := f.lib.QueryContext(t.Context(), `SELECT kind, path, coalesce(from_path, ''), file_id FROM mediaservers_change ORDER BY path DESC`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rs.Close() }()
	var out []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.kind, &r.path, &r.from, &r.file); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rs.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
