package rules

import (
	"cmp"
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
// finding. It applies every set's migrations to one empty in-memory
// SQLite, as a library file holds them, so a data step may read another
// set's tables.
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
	dirs := slices.Sorted(maps.Keys(sets))
	sources := map[string][]string{}
	fsyss := map[string]fstest.MapFS{}
	for _, dir := range dirs {
		slices.Sort(sets[dir])
		fsyss[dir] = fstest.MapFS{}
		for _, f := range sets[dir] {
			src, err := repo.Read(f)
			if err != nil {
				return nil, err
			}
			sources[dir] = append(sources[dir], string(src))
			fsyss[dir][path.Base(f)] = &fstest.MapFile{Data: src}
		}
	}
	columns, err := finalColumns(dirs, fsyss)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, dir := range dirs {
		for _, c := range columns[dir] {
			if !plainCredential(c.column) {
				continue
			}
			at, line := introduced(c.column, sets[dir], sources[dir])
			findings = append(findings, lint.Finding{Path: at, Line: line, Message: fmt.Sprintf("%s.%s holds a credential in plain text; seal it as %[2]s_sealed or keep only its digest as %[2]s_hash", c.table, c.column)})
		}
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

// finalColumns applies each dir's migrations and returns the columns of
// the tables each made. A set whose data step reads a table another set
// has yet to make waits a round; a round that applies nothing fails.
func finalColumns(dirs []string, fsyss map[string]fstest.MapFS) (map[string][]tableColumn, error) {
	ctx := context.Background()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open an in-memory database: %w", err)
	}
	defer func() { _ = conn.Close() }()
	conn.SetMaxOpenConns(1)
	provs := map[string]*goose.Provider{}
	for i, dir := range dirs {
		prov, err := goose.NewProvider(goose.DialectSQLite3, conn, fsyss[dir], goose.WithDisableGlobalRegistry(true), goose.WithLogger(goose.NopLogger()),
			goose.WithTableName(fmt.Sprintf("goose_set%d", i)))
		if err != nil {
			return nil, fmt.Errorf("%s: read the migrations: %w", dir, err)
		}
		provs[dir] = prov
	}
	owner := map[string]string{}
	for pending := dirs; len(pending) > 0; {
		var waiting []string
		var first error
		for _, dir := range pending {
			_, err := provs[dir].Up(ctx)
			if err := claimTables(ctx, conn, owner, dir); err != nil {
				return nil, err
			}
			if err != nil {
				waiting = append(waiting, dir)
				first = cmp.Or(first, fmt.Errorf("%s: apply the migrations: %w", dir, err))
			}
		}
		if len(waiting) == len(pending) {
			return nil, first
		}
		pending = waiting
	}
	rows, err := conn.QueryContext(ctx, `SELECT m.name, p.name FROM sqlite_schema AS m, pragma_table_info(m.name) AS p
		WHERE m.type = 'table' AND m.name NOT LIKE 'goose%' AND m.name NOT LIKE 'sqlite%' ORDER BY m.name, p.cid`)
	if err != nil {
		return nil, fmt.Errorf("read the schema: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string][]tableColumn{}
	for rows.Next() {
		var c tableColumn
		if err := rows.Scan(&c.table, &c.column); err != nil {
			return nil, fmt.Errorf("read the schema: %w", err)
		}
		out[owner[c.table]] = append(out[owner[c.table]], c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the schema: %w", err)
	}
	return out, nil
}

// claimTables records dir as the owner of every table no set owns yet.
func claimTables(ctx context.Context, conn *sql.DB, owner map[string]string, dir string) error {
	rows, err := conn.QueryContext(ctx, `SELECT name FROM sqlite_schema WHERE type = 'table'`)
	if err != nil {
		return fmt.Errorf("read the schema: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("read the schema: %w", err)
		}
		if _, ok := owner[name]; !ok {
			owner[name] = dir
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read the schema: %w", err)
	}
	return nil
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
