package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Catalog struct {
	Versions Versions
	Objects  []Object // ordered by Type, then Name; sqlite_* objects excluded
}

type Object struct {
	Type  string // table, index, trigger or view
	Name  string
	Table string // the table an index or trigger is on
	SQL   string

	// Tables only, from PRAGMA table_list, table_info and foreign_key_list.
	Kind        string // table, virtual or shadow
	Strict      bool
	Columns     []Column
	ForeignKeys []ForeignKey
}

type Column struct {
	Name string
	Type string // declared type, as written
	PK   bool
}

type ForeignKey struct {
	Column string
	Table  string
	To     string // parent column; the parent's primary key when the SQL names none
}

func (d *DB[F]) Catalog(ctx context.Context) (Catalog, error) {
	mustNotHold[F](ctx)
	name := policyOf[F]().name
	versions, err := readVersions(ctx, d.read)
	if err != nil {
		return Catalog{}, fmt.Errorf("%s: %w", name, err)
	}
	objects, err := readObjects(ctx, d.read)
	if err != nil {
		return Catalog{}, fmt.Errorf("%s: catalog: %w", name, err)
	}
	for i := range objects {
		if objects[i].Type == "table" {
			if err := readTable(ctx, d.read, &objects[i]); err != nil {
				return Catalog{}, fmt.Errorf("%s: catalog table %s: %w", name, objects[i].Name, err)
			}
		}
	}
	resolveImplicitKeys(objects)
	return Catalog{Versions: versions, Objects: objects}, nil
}

func readObjects(ctx context.Context, conn *sql.DB) ([]Object, error) {
	rows, err := conn.QueryContext(ctx, "SELECT type, name, tbl_name, coalesce(sql, '') FROM sqlite_schema WHERE name NOT LIKE 'sqlite\\_%' ESCAPE '\\' ORDER BY type, name")
	if err != nil {
		return nil, fmt.Errorf("read sqlite_schema: %w", err)
	}
	var objects []Object
	for rows.Next() {
		var o Object
		if err := rows.Scan(&o.Type, &o.Name, &o.Table, &o.SQL); err != nil {
			return nil, errors.Join(fmt.Errorf("read sqlite_schema: %w", err), rows.Close())
		}
		objects = append(objects, o)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, fmt.Errorf("read sqlite_schema: %w", err)
	}
	return objects, nil
}

func readTable(ctx context.Context, conn *sql.DB, o *Object) error {
	if err := conn.QueryRowContext(ctx, "SELECT type, strict FROM pragma_table_list WHERE schema = 'main' AND name = ?", o.Name).Scan(&o.Kind, &o.Strict); err != nil {
		return fmt.Errorf("table_list: %w", err)
	}
	if o.Kind != "table" {
		return nil
	}
	cols, err := conn.QueryContext(ctx, "SELECT name, type, pk > 0 FROM pragma_table_info(?) ORDER BY cid", o.Name)
	if err != nil {
		return fmt.Errorf("table_info: %w", err)
	}
	for cols.Next() {
		var c Column
		if err := cols.Scan(&c.Name, &c.Type, &c.PK); err != nil {
			return errors.Join(fmt.Errorf("table_info: %w", err), cols.Close())
		}
		o.Columns = append(o.Columns, c)
	}
	if err := errors.Join(cols.Err(), cols.Close()); err != nil {
		return fmt.Errorf("table_info: %w", err)
	}
	fks, err := conn.QueryContext(ctx, `SELECT "from", "table", coalesce("to", '') FROM pragma_foreign_key_list(?) ORDER BY id, seq`, o.Name)
	if err != nil {
		return fmt.Errorf("foreign_key_list: %w", err)
	}
	for fks.Next() {
		var fk ForeignKey
		if err := fks.Scan(&fk.Column, &fk.Table, &fk.To); err != nil {
			return errors.Join(fmt.Errorf("foreign_key_list: %w", err), fks.Close())
		}
		o.ForeignKeys = append(o.ForeignKeys, fk)
	}
	if err := errors.Join(fks.Err(), fks.Close()); err != nil {
		return fmt.Errorf("foreign_key_list: %w", err)
	}
	return nil
}

func resolveImplicitKeys(objects []Object) {
	pk := map[string]string{}
	for _, o := range objects {
		for _, c := range o.Columns {
			if c.PK {
				pk[o.Name] = c.Name
			}
		}
	}
	for i := range objects {
		for j := range objects[i].ForeignKeys {
			if fk := &objects[i].ForeignKeys[j]; fk.To == "" {
				fk.To = pk[fk.Table]
			}
		}
	}
}
