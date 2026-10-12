package storagetest_test

import (
	"fmt"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/example/clip"
	"github.com/jatassi/slipstream/internal/storage/example/shelf"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type recorder struct {
	testing.TB
	errs []string
}

func (r *recorder) Error(args ...any) { r.errs = append(r.errs, fmt.Sprint(args...)) }

func (r *recorder) Errorf(format string, args ...any) {
	r.errs = append(r.errs, fmt.Sprintf(format, args...))
}

func set(name string, migrations ...string) storage.Schema[storage.Library] {
	fsys := fstest.MapFS{}
	for i, sql := range migrations {
		fsys[fmt.Sprintf("schema/%05d_%s.sql", i+1, name)] = &fstest.MapFile{Data: []byte("-- +goose Up\n" + sql + "\n")}
	}
	return storage.NewSchema[storage.Library](name, fsys)
}

func audit(t *testing.T, schemas ...storage.Schema[storage.Library]) []string {
	t.Helper()
	r := &recorder{TB: t}
	storagetest.AuditLibrary(r, schemas)
	return r.errs
}

func TestExampleOwnersPassTheAudit(t *testing.T) {
	storagetest.AuditLibrary(t, []storage.Schema[storage.Library]{shelf.Schema(), clip.Schema()})
}

func TestAuditFindings(t *testing.T) {
	alpha := set("alpha", "CREATE TABLE alpha (id INTEGER PRIMARY KEY) STRICT;")
	for _, tc := range []struct {
		name    string
		schemas []storage.Schema[storage.Library]
		want    []string
	}{
		{
			name: "references a later set",
			schemas: []storage.Schema[storage.Library]{
				set("early", "CREATE TABLE early (id INTEGER PRIMARY KEY, late_id INTEGER REFERENCES late (id)) STRICT;"),
				set("late", "CREATE TABLE late (id INTEGER PRIMARY KEY) STRICT;"),
			},
			want: []string{"early: foreign key early.late_id targets late, which no earlier set creates"},
		},
		{
			name:    "lacks the prefix",
			schemas: []storage.Schema[storage.Library]{set("alpha", "CREATE TABLE things (id INTEGER PRIMARY KEY) STRICT;")},
			want:    []string{"alpha: table things lacks the prefix alpha_"},
		},
		{
			name:    "not STRICT",
			schemas: []storage.Schema[storage.Library]{set("alpha", "CREATE TABLE alpha (id INTEGER PRIMARY KEY);")},
			want:    []string{"alpha: table alpha is not STRICT"},
		},
		{
			name:    "TEXT _at column",
			schemas: []storage.Schema[storage.Library]{set("alpha", "CREATE TABLE alpha (id INTEGER PRIMARY KEY, created_at TEXT NOT NULL) STRICT;")},
			want:    []string{"alpha: column alpha.created_at is TEXT; _at columns are INTEGER"},
		},
		{
			name:    "alters an earlier set",
			schemas: []storage.Schema[storage.Library]{alpha, set("beta", "ALTER TABLE alpha ADD COLUMN note TEXT;")},
			want:    []string{"beta: changed table alpha of set alpha"},
		},
		{
			name: "references an earlier set by a column other than id",
			schemas: []storage.Schema[storage.Library]{
				set("alpha", "CREATE TABLE alpha (id INTEGER PRIMARY KEY, code TEXT NOT NULL UNIQUE) STRICT;"),
				set("beta", "CREATE TABLE beta (id INTEGER PRIMARY KEY, alpha_code TEXT REFERENCES alpha (code)) STRICT;"),
			},
			want: []string{"beta: foreign key beta.alpha_code targets alpha.code; a foreign key into another set targets id"},
		},
		{
			name: "indexes and triggers on an earlier set",
			schemas: []storage.Schema[storage.Library]{alpha, set("beta",
				"CREATE TABLE beta (id INTEGER PRIMARY KEY) STRICT;",
				"CREATE INDEX beta_alpha ON alpha (id); CREATE TRIGGER beta_sync AFTER INSERT ON beta BEGIN DELETE FROM alpha; END;",
			)},
			want: []string{
				"beta: index beta_alpha is on alpha, a table of set alpha",
				"beta: trigger beta_sync names alpha, a table of set alpha",
			},
		},
		{
			name:    "lists a set twice",
			schemas: []storage.Schema[storage.Library]{alpha, alpha},
			want:    []string{"set alpha is listed twice"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := audit(t, tc.schemas...); !slices.Equal(got, tc.want) {
				t.Errorf("findings = %q, want %q", got, tc.want)
			}
		})
	}
}
