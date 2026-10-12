package storagetest

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
)

// AuditLibrary replays schemas one set at a time on a fresh library.db and
// reports every broken convention with t.Errorf. Then it reopens the file
// with the whole list and reports anything that reopening applies.
func AuditLibrary(t testing.TB, schemas []storage.Schema[storage.Library]) {
	t.Helper()
	audit(t, storage.LibraryFile, schemas, openLibrary)
}

func AuditLocal(t testing.TB, schemas []storage.Schema[storage.Local]) {
	t.Helper()
	audit(t, storage.LocalFile, schemas, openLocal)
}

type opener[F storage.File] func(ctx context.Context, path string, schemas []storage.Schema[F], log *slog.Logger) (*storage.DB[F], error)

func audit[F storage.File](t testing.TB, name string, schemas []storage.Schema[F], open opener[F]) {
	t.Helper()
	var sets []string
	for _, s := range schemas {
		if slices.Contains(sets, s.Set()) {
			t.Errorf("set %s is listed twice", s.Set())
			return
		}
		sets = append(sets, s.Set())
	}
	path := filepath.Join(t.TempDir(), name)
	discard := slog.New(slog.DiscardHandler)
	before, err := catalog(t.Context(), path, nil, open, discard)
	if err != nil {
		t.Errorf("open empty %s: %v", name, err)
		return
	}
	for i, s := range schemas {
		after, err := catalog(t.Context(), path, schemas[:i+1], open, discard)
		if err != nil {
			t.Errorf("%s: %v", s.Set(), err)
			return
		}
		for _, f := range findings(s.Set(), sets[:i], before, after) {
			t.Error(f)
		}
		before = after
	}
	var log bytes.Buffer
	again, err := catalog(t.Context(), path, schemas, open, slog.New(slog.NewTextHandler(&log, nil)))
	if err != nil {
		t.Errorf("reopen %s: %v", name, err)
		return
	}
	if strings.Contains(log.String(), "msg=migrated") || !reflect.DeepEqual(again, before) {
		t.Errorf("reopening %s changed it; a migration is not idempotent or a set is missing from the list", name)
	}
}

func catalog[F storage.File](ctx context.Context, path string, schemas []storage.Schema[F], open opener[F], log *slog.Logger) (storage.Catalog, error) {
	db, err := open(ctx, path, schemas, log)
	if err != nil {
		return storage.Catalog{}, err
	}
	c, err := db.Catalog(ctx)
	if cerr := db.Close(); err == nil {
		err = cerr
	}
	return c, err
}

type step struct {
	set    string
	known  []string
	before map[string]storage.Object
	after  map[string]storage.Object
	out    []string
}

func findings(set string, earlier []string, before, after storage.Catalog) []string {
	s := &step{set: set, known: append(slices.Clone(earlier), set), before: byKey(before), after: byKey(after)}
	for _, o := range before.Objects {
		s.checkKept(o)
	}
	for _, o := range after.Objects {
		old, existed := s.before[key(o)]
		if isVersionTable(o) || (existed && (old.SQL == o.SQL || s.owner(o.Name) != set)) {
			continue
		}
		s.checkOwn(o)
	}
	return s.out
}

func (s *step) report(format string, args ...any) {
	s.out = append(s.out, s.set+": "+fmt.Sprintf(format, args...))
}

func (s *step) checkKept(o storage.Object) {
	owner := s.owner(o.Name)
	if owner == "" || owner == s.set || isVersionTable(o) {
		return
	}
	switch now, ok := s.after[key(o)]; {
	case !ok:
		s.report("dropped %s %s of set %s", o.Type, o.Name, owner)
	case now.SQL != o.SQL:
		s.report("changed %s %s of set %s", o.Type, o.Name, owner)
	}
}

func (s *step) checkOwn(o storage.Object) {
	if o.Name != s.set && !strings.HasPrefix(o.Name, s.set+"_") {
		s.report("%s %s lacks the prefix %s_", o.Type, o.Name, s.set)
	}
	if o.Type == "table" && o.Kind == "table" {
		s.checkTable(o)
	}
	if o.Type == "index" || o.Type == "trigger" {
		if other := s.owner(o.Table); other != "" && other != s.set {
			s.report("%s %s is on %s, a table of set %s", o.Type, o.Name, o.Table, other)
		}
	}
	if o.Type == "trigger" || o.Type == "view" {
		for _, word := range identifier.FindAllString(o.SQL, -1) {
			table, ok := s.after["table "+strings.ToLower(word)]
			if other := s.owner(table.Name); ok && other != "" && other != s.set {
				s.report("%s %s names %s, a table of set %s", o.Type, o.Name, table.Name, other)
			}
		}
	}
}

func (s *step) checkTable(o storage.Object) {
	if !o.Strict {
		s.report("table %s is not STRICT", o.Name)
	}
	for _, c := range o.Columns {
		if strings.HasSuffix(c.Name, "_at") && !strings.EqualFold(c.Type, "INTEGER") {
			s.report("column %s.%s is %s; _at columns are INTEGER", o.Name, c.Name, c.Type)
		}
	}
	for _, fk := range o.ForeignKeys {
		_, exists := s.after["table "+strings.ToLower(fk.Table)]
		switch owner := s.owner(fk.Table); {
		case !exists || owner == "":
			s.report("foreign key %s.%s targets %s, which no earlier set creates", o.Name, fk.Column, fk.Table)
		case owner != s.set && fk.To != "id":
			s.report("foreign key %s.%s targets %s.%s; a foreign key into another set targets id", o.Name, fk.Column, fk.Table, fk.To)
		}
	}
}

func (s *step) owner(name string) string {
	prefix, _, _ := strings.Cut(strings.ToLower(name), "_")
	if slices.Contains(s.known, prefix) {
		return prefix
	}
	return ""
}

var identifier = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

func isVersionTable(o storage.Object) bool {
	return o.Type == "table" && strings.HasPrefix(o.Name, "goose_")
}

func key(o storage.Object) string { return o.Type + " " + strings.ToLower(o.Name) }

func byKey(c storage.Catalog) map[string]storage.Object {
	m := make(map[string]storage.Object, len(c.Objects))
	for _, o := range c.Objects {
		m[key(o)] = o
	}
	return m
}
