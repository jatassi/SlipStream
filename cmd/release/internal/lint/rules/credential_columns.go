package rules

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // the driver the migrations are applied with

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "credential-columns",
		Doc:   "no schema ends with a plain-text credential column: one named like a password, API key, token, secret, passkey or cookie ends in _sealed or _hash (spec #137, Secrets at rest)",
		Check: checkCredentialColumns,
	})
}

var (
	credentialSuffixes = []string{"password", "api_key", "token", "secret", "passkey", "cookie"}
	protectedSuffixes  = []string{"_sealed", "_hash"}
)

// checkCredentialColumns judges the schema each set's migrations end at,
// not each file, so a column a later migration dropped is history and no
// finding. It applies the set's migrations to an empty in-memory SQLite.
func checkCredentialColumns(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.Files()
	if err != nil {
		return nil, err
	}
	sets := map[string][]string{}
	for _, f := range files {
		if lint.IsOwnerSQL(f, "schema") && migrationName.MatchString(path.Base(f)) {
			sets[path.Dir(f)] = append(sets[path.Dir(f)], f)
		}
	}
	var findings []lint.Finding
	for _, dir := range slices.Sorted(maps.Keys(sets)) {
		found, err := setCredentialColumns(repo, sets[dir])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", dir, err)
		}
		findings = append(findings, found...)
	}
	return findings, nil
}

func setCredentialColumns(repo *lint.Repo, migrations []string) ([]lint.Finding, error) {
	slices.Sort(migrations)
	fsys := fstest.MapFS{}
	sources := make([]string, len(migrations))
	for i, f := range migrations {
		src, err := repo.Read(f)
		if err != nil {
			return nil, err
		}
		sources[i] = string(src)
		fsys[path.Base(f)] = &fstest.MapFile{Data: src}
	}
	columns, err := finalColumns(fsys)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, c := range columns {
		if !plainCredential(c.column) {
			continue
		}
		at, line := introduced(c.column, migrations, sources)
		findings = append(findings, lint.Finding{Path: at, Line: line, Message: fmt.Sprintf("%s.%s holds a credential in plain text; seal it as %[2]s_sealed or keep only its digest as %[2]s_hash", c.table, c.column)})
	}
	return findings, nil
}

func plainCredential(column string) bool {
	for _, p := range protectedSuffixes {
		if strings.HasSuffix(column, p) {
			return false
		}
	}
	for _, s := range credentialSuffixes {
		if strings.HasSuffix(column, s) {
			return true
		}
	}
	return false
}

type tableColumn struct{ table, column string }

func finalColumns(fsys fstest.MapFS) ([]tableColumn, error) {
	ctx := context.Background()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open an in-memory database: %w", err)
	}
	defer func() { _ = conn.Close() }()
	conn.SetMaxOpenConns(1)
	prov, err := goose.NewProvider(goose.DialectSQLite3, conn, fsys, goose.WithDisableGlobalRegistry(true), goose.WithLogger(goose.NopLogger()))
	if err != nil {
		return nil, fmt.Errorf("read the migrations: %w", err)
	}
	if _, err := prov.Up(ctx); err != nil {
		return nil, fmt.Errorf("apply the migrations: %w", err)
	}
	rows, err := conn.QueryContext(ctx, `SELECT m.name, p.name FROM sqlite_schema AS m, pragma_table_info(m.name) AS p
		WHERE m.type = 'table' AND m.name NOT LIKE 'goose%' AND m.name NOT LIKE 'sqlite%' ORDER BY m.name, p.cid`)
	if err != nil {
		return nil, fmt.Errorf("read the schema: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []tableColumn
	for rows.Next() {
		var c tableColumn
		if err := rows.Scan(&c.table, &c.column); err != nil {
			return nil, fmt.Errorf("read the schema: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the schema: %w", err)
	}
	return out, nil
}

func introduced(column string, migrations, sources []string) (file string, line int) {
	word := regexp.MustCompile(`\b` + regexp.QuoteMeta(column) + `\b`)
	for i, src := range sources {
		for n, line := range strings.Split(src, "\n") {
			if word.MatchString(line) {
				return migrations[i], n + 1
			}
		}
	}
	return migrations[len(migrations)-1], 1
}
