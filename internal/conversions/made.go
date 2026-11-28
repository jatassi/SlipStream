package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
)

// Made is what a conversion Import reads of its Conversion: the made
// file's provenance, nil once its source file or Target is gone, and the
// source file's name.
type Made struct {
	Derived    *files.Derived
	SourceName string
}

// MadeFrom is Conversion id's Made, ErrNoConversion when none has id.
func MadeFrom(ctx context.Context, r storage.Reader[storage.Library], id ID) (Made, error) {
	row, err := readRow(ctx, db.New(r), id)
	if err != nil {
		return Made{}, err
	}
	m := Made{SourceName: row.Snapshot.SourceName}
	if row.SourceFile != nil && row.Target != nil {
		m.Derived = &files.Derived{From: *row.SourceFile, Target: *row.Target, Version: row.Version}
	}
	return m, nil
}
