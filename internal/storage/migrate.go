package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/pressly/goose/v3"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// PreMigrationHook runs before the first pending migration on a library.db
// that already holds at least one applied migration. An error fails the
// open with nothing migrated (ADR 0015).
type PreMigrationHook func(ctx context.Context, db *Unmigrated, plan Plan) error

type Unmigrated struct {
	conn *sql.DB
}

func (u *Unmigrated) Versions(ctx context.Context) (Versions, error) {
	return readVersions(ctx, u.conn)
}

func migrate[F File](ctx context.Context, path string, schemas []Schema[F], hook PreMigrationHook, t Timings, log *slog.Logger) (err error) {
	p := policyOf[F]()
	conn, err := sql.Open("sqlite", dsn(path, p, t, roleMigrate))
	if err != nil {
		return fmt.Errorf("%s: open: %w", p.name, err)
	}
	conn.SetMaxOpenConns(1)
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("%s: close migration connection: %w", p.name, cerr))
		}
	}()
	held, plan, err := check(ctx, conn, p, schemas)
	if err != nil || len(plan) == 0 {
		return err
	}
	if hook != nil && len(held) > 0 {
		if err := hook(ctx, &Unmigrated{conn: conn}, plan); err != nil {
			return fmt.Errorf("before migrating %s: %w", p.name, err)
		}
	}
	for _, s := range schemas {
		if !slices.ContainsFunc(plan, func(step Step) bool { return step.Set == s.set }) {
			continue
		}
		if err := applySet(ctx, conn, p, s); err != nil {
			return err
		}
		if err := foreignKeyCheck(ctx, conn, p, s.set); err != nil {
			return err
		}
	}
	log.Info("migrated", "file", p.name, "sets", plan)
	return nil
}

func check[F File](ctx context.Context, conn *sql.DB, p policy, schemas []Schema[F]) (Versions, Plan, error) {
	if p.quickCheck {
		if err := quickCheck(ctx, conn, p); err != nil {
			return nil, nil, err
		}
	}
	held, err := readVersions(ctx, conn)
	if err != nil {
		return nil, nil, corruptOr(p, err)
	}
	plan, err := Compare(held, schemas)
	return held, plan, err
}

func quickCheck(ctx context.Context, conn *sql.DB, p policy) error {
	var result string
	if err := conn.QueryRowContext(ctx, "PRAGMA quick_check(1)").Scan(&result); err != nil {
		return corruptOr(p, err)
	}
	if result != "ok" {
		return &corruptError{file: p.name, advice: p.corruptAdvice, cause: errors.New(result)}
	}
	return nil
}

func readVersions(ctx context.Context, conn *sql.DB) (Versions, error) {
	rows, err := conn.QueryContext(ctx, "SELECT name FROM sqlite_schema WHERE type = 'table' AND name GLOB 'goose_*' ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("read schema versions: %w", err)
	}
	var sets []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("read schema versions: %w", err)
		}
		if set, ok := versionTableSet(name); ok {
			sets = append(sets, set)
		}
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, fmt.Errorf("read schema versions: %w", err)
	}
	held := Versions{}
	for _, set := range sets {
		var v int64
		q := "SELECT coalesce(max(version_id), 0) FROM " + versionTable(set) + " WHERE is_applied"
		if err := conn.QueryRowContext(ctx, q).Scan(&v); err != nil {
			return nil, fmt.Errorf("read %s: %w", versionTable(set), err)
		}
		if v > 0 {
			held[set] = v
		}
	}
	return held, nil
}

const versionTablePrefix = "goose_"

func versionTable(set string) string { return versionTablePrefix + set }

func versionTableSet(table string) (string, bool) {
	set, ok := strings.CutPrefix(table, versionTablePrefix)
	return set, ok && setName.MatchString(set)
}

func applySet[F File](ctx context.Context, conn *sql.DB, p policy, s Schema[F]) error {
	prov, err := goose.NewProvider(goose.DialectSQLite3, conn, s.files,
		goose.WithTableName(versionTable(s.set)),
		goose.WithDisableGlobalRegistry(true),
		goose.WithExcludeVersions(s.above()),
		goose.WithLogger(goose.NopLogger()),
	)
	if err != nil {
		return fmt.Errorf("%s: schema %s: %w", p.name, s.set, err)
	}
	if _, err := prov.Up(ctx); err != nil {
		var partial *goose.PartialError
		if errors.As(err, &partial) && partial.Failed != nil && partial.Failed.Source != nil {
			return &migrationError{file: p.name, set: s.set, version: partial.Failed.Source.Version, cause: partial.Err, advice: p.migrationAdvice}
		}
		return fmt.Errorf("%s: schema %s: %w", p.name, s.set, err)
	}
	return nil
}

func foreignKeyCheck(ctx context.Context, conn *sql.DB, p policy, set string) error {
	var table, parent string
	var rowid sql.NullInt64
	var fkid int64
	err := conn.QueryRowContext(ctx, "PRAGMA foreign_key_check").Scan(&table, &rowid, &parent, &fkid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: foreign key check after set %s: %w", p.name, set, err)
	}
	return fmt.Errorf("%s: set %s left rows in %s whose foreign key into %s matches nothing", p.name, set, table, parent)
}

func corruptOr(p policy, err error) error {
	var serr *sqlite.Error
	if errors.As(err, &serr) {
		switch serr.Code() & 0xff {
		case sqlite3.SQLITE_NOTADB, sqlite3.SQLITE_CORRUPT:
			return &corruptError{file: p.name, advice: p.corruptAdvice, cause: err}
		}
	}
	return fmt.Errorf("%s: %w", p.name, err)
}

type corruptError struct {
	file   string
	advice string
	cause  error
}

func (e *corruptError) Error() string {
	return e.file + " isn't a readable SQLite database" + e.advice
}

func (e *corruptError) Unwrap() error { return e.cause }

type migrationError struct {
	file    string
	set     string
	version int64
	cause   error
	advice  string
}

func (e *migrationError) Error() string {
	return fmt.Sprintf("%s migration %s %d failed: %v%s", e.file, e.set, e.version, e.cause, e.advice)
}

func (e *migrationError) Unwrap() error { return e.cause }
