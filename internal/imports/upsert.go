package imports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// pend is one folder's pending files as a scan or a Leave files delete
// leaves them.
type pend struct {
	Root        roots.ID
	Folder      string
	ContentType works.TypeID
	Hint        scanning.Hint
	Suggestion  *Suggestion
	// Files are the files found new or changed since the entry listed
	// them. Listed are the paths it lists that stay as they are. Every
	// other path it lists is gone from disk or joined a Work.
	Files  []PendingFile
	Listed []string
	// Ignore ignores the entry whatever its files: a Leave files delete.
	Ignore bool
}

// upsertPending merges p into its folder's entry in tx. The entry keeps
// its suggestion, which may be an Admin's Match, unless the hint changed,
// and stays ignored unless its set of files changed. An entry left with no
// files is deleted.
func upsertPending(ctx context.Context, tx *storage.Tx[file], p pend, now storage.Time) error {
	row, err := db.New(tx).GetEntryAt(ctx, db.GetEntryAtParams{RootID: p.Root, Folder: p.Folder})
	if errors.Is(err, storage.ErrNoRows) {
		return insertEntry(ctx, tx, p, now)
	}
	if err != nil {
		return err
	}
	e, err := entry(ctx, tx, row.ID)
	if err != nil {
		return err
	}
	fresh := map[string]bool{}
	for _, f := range p.Files {
		fresh[f.Path] = true
	}
	kept := slices.DeleteFunc(slices.Clone(e.Files), func(f PendingFile) bool { return fresh[f.Path] || !slices.Contains(p.Listed, f.Path) })
	changed := len(p.Files) > 0 || len(kept) < len(e.Files)
	e.Files = slices.Concat(kept, p.Files)
	slices.SortFunc(e.Files, func(a, b PendingFile) int { return strings.Compare(a.Path, b.Path) })
	same, err := sameHint(e.Hint, p.Hint)
	if err != nil {
		return err
	}
	if !same || e.Suggestion == nil {
		e.Hint, e.Suggestion = p.Hint, p.Suggestion
	}
	switch {
	case p.Ignore && !e.Ignored():
		e.IgnoredAt = now
	case changed && !p.Ignore:
		e.IgnoredAt = storage.Time{}
	}
	return write(ctx, tx, e)
}

func insertEntry(ctx context.Context, tx *storage.Tx[file], p pend, now storage.Time) error {
	if len(p.Files) == 0 {
		return nil
	}
	hint, err := json.Marshal(p.Hint)
	if err != nil {
		return fmt.Errorf("the hint of %s: %w", p.Folder, err)
	}
	e := db.InsertEntryParams{ID: EntryID{UUID: storage.NewUUID()}, RootID: p.Root, Folder: p.Folder, ContentType: p.ContentType, Hint: string(hint), FoundAt: now}
	if p.Ignore {
		e.IgnoredAt = now
	}
	if err := db.New(tx).InsertEntry(ctx, e); err != nil {
		return err
	}
	return write(ctx, tx, Entry{ID: e.ID, Hint: p.Hint, Suggestion: p.Suggestion, IgnoredAt: e.IgnoredAt, Files: p.Files})
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

// listedAt is the paths the entry of root's folder lists, none when it
// has no entry.
func listedAt(ctx context.Context, r storage.Reader[file], root roots.ID, folder string) ([]string, error) {
	q := db.New(r)
	row, err := q.GetEntryAt(ctx, db.GetEntryAtParams{RootID: root, Folder: folder})
	if errors.Is(err, storage.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fs, err := q.ListFilesOf(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Path
	}
	return out, nil
}
