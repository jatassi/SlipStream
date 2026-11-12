package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// pend is one folder's waiting files as a scan or a Leave files delete
// leaves them.
type pend struct {
	Root        roots.ID
	Folder      string
	ContentType works.TypeID
	Hint        scanning.Hint
	Suggestion  *Suggestion
	// Files are the files found new or changed since the Import listed
	// them, each with its reason. Listed are the paths it lists that stay
	// as they are. Every other path it lists is gone from disk or joined a
	// Work.
	Files  []File
	Listed []string
	// Ignore ignores the Import whatever its files: a Leave files delete.
	Ignore bool
}

// upsertPending merges p into its folder's unfinished root Import in tx.
// The Import keeps its suggestion, which may be an Admin's Match, unless
// the hint changed, and stays ignored unless its set of waiting files
// changed. One left with no files is deleted.
func upsertPending(ctx context.Context, tx *storage.Tx[file], p pend, now storage.Time) error {
	e, err := rootImportAt(ctx, tx, db.GetRootImportAtParams{RootID: p.Root, Folder: &p.Folder})
	if errors.Is(err, storage.ErrNoRows) {
		return insertPending(ctx, tx, p, now)
	}
	if err != nil {
		return err
	}
	fresh := map[string]bool{}
	for _, f := range p.Files {
		fresh[f.Path] = true
	}
	before := len(e.Files)
	e.Files = slices.DeleteFunc(e.Files, func(f File) bool {
		return f.Outcome == Waiting && (fresh[f.Path] || !slices.Contains(p.Listed, f.Path))
	})
	changed := len(p.Files) > 0 || len(e.Files) < before
	appendWaiting(&e, p.Files...)
	same, err := sameHint(e.Hint, p.Hint)
	if err != nil {
		return err
	}
	if !same || e.Suggestion == nil {
		e.Hint, e.Suggestion = p.Hint, p.Suggestion
	}
	switch {
	case p.Ignore && !e.IsIgnored():
		e.Ignored = now
	case changed && !p.Ignore:
		e.Ignored = storage.Time{}
	}
	return save(ctx, tx, &e)
}

func insertPending(ctx context.Context, tx *storage.Tx[file], p pend, now storage.Time) error {
	if len(p.Files) == 0 {
		return nil
	}
	e := Import{
		ID: ImportID{UUID: storage.NewUUID()}, Source: FromRoot, Root: p.Root, Folder: p.Folder, ContentType: p.ContentType,
		Hint: p.Hint, Suggestion: p.Suggestion, Created: now,
	}
	if p.Ignore {
		e.Ignored = now
	}
	appendWaiting(&e, p.Files...)
	return insertImport(ctx, tx, e)
}

// sameHint compares two hints as they are stored.
func sameHint(a, b scanning.Hint) (bool, error) {
	ja, err := json.Marshal(a)
	if err != nil {
		return false, fmt.Errorf("a hint: %w", err)
	}
	jb, err := json.Marshal(b)
	if err != nil {
		return false, fmt.Errorf("a hint: %w", err)
	}
	return bytes.Equal(ja, jb), nil
}

// listedAt is the paths of the waiting files of root's folder's
// unfinished Import, none when it has none.
func listedAt(ctx context.Context, r storage.Reader[file], root roots.ID, folder string) ([]string, error) {
	e, err := rootImportAt(ctx, r, db.GetRootImportAtParams{RootID: root, Folder: &folder})
	if errors.Is(err, storage.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range e.waiting() {
		out = append(out, f.Path)
	}
	return out, nil
}
