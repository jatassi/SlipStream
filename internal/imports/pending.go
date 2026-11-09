package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// Entry is one pending import: a Work folder of a root, or a file loose at
// its top, and its files that wait.
type Entry struct {
	ID          EntryID
	Root        roots.ID
	Folder      string
	ContentType works.TypeID
	Hint        scanning.Hint
	// Suggestion is the best candidate, nil for none.
	Suggestion *Suggestion
	FoundAt    storage.Time
	// IgnoredAt is zero while the entry needs the Admin.
	IgnoredAt storage.Time
	Files     []PendingFile
}

func (e Entry) Ignored() bool { return !e.IgnoredAt.IsZero() }

// Suggestion is an entry's best candidate at its type's source.
type Suggestion struct {
	Key       works.Key   `json:"key"`
	Label     string      `json:"label"`
	Year      int         `json:"year,omitempty"`
	Poster    artwork.Ref `json:"poster"`
	Confident bool        `json:"confident,omitempty"`
	// WorkID is the library Work with Key, read when the entry is, zero
	// while none has it.
	WorkID works.WorkID `json:"-"`
}

// PendingFile is one file of an entry, root-relative.
type PendingFile struct {
	Path string
	Size int64
	// Seen is nil for a file left from a row no scan or import read.
	Seen *storing.Seen
	// Media is nil for a file ffprobe can't read; Quality is then zero.
	Media   *media.Record
	Quality release.Quality
	Cut     release.CutKey
	// Items is what its path names in its Work, empty until a Work places
	// it.
	Items  []works.ItemID
	Reason scanning.Reason
}

var (
	ErrNoEntry      = errors.New("no pending import has that id")
	ErrNoSuggestion = errors.New("the pending import has no suggested Work")
	ErrElsewhere    = errors.New("the Work lives in another library root")
	ErrNotOfWork    = errors.New("a file names an Item the Work doesn't hold")
	ErrChanged      = errors.New("the pending import no longer lists a chosen file")
)

// Filter picks Pending imports: one root's or every root's, and the
// entries that need the Admin or the ignored ones.
type Filter struct {
	Root    *roots.ID
	Ignored bool
}

// List is Pending imports: what each root's last comparison added and
// how many entries need the Admin, then the entries.
type List struct {
	Leads   []Lead
	Entries []Entry
	// Roots is the words of every root the listing names, its leads' and
	// its reasons' alike.
	Roots map[roots.ID]string
}

// Lead is one compared root's line: "The last scan of the Movies library
// root on Media server added 398 movies. 14 folders need you."
type Lead struct {
	Root        roots.ID
	ContentType works.TypeID
	Added       int
	NeedsYou    int
}

// Pending lists the entries f picks, folder order within each root.
func (im *Imports) Pending(ctx context.Context, _ identity.Admin, f Filter) (List, error) {
	words, err := im.cfg.Roots.Words(ctx)
	if err != nil {
		return List{}, err
	}
	var out List
	err = im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		all, err := entries(ctx, r)
		if err != nil {
			return err
		}
		inLibrary, err := keysIn(ctx, r)
		if err != nil {
			return err
		}
		for _, e := range all {
			if e.Ignored() != f.Ignored || f.Root != nil && e.Root != *f.Root {
				continue
			}
			if e.Suggestion != nil {
				e.Suggestion.WorkID = inLibrary[typedKey{e.ContentType, e.Suggestion.Key}]
			}
			out.Entries = append(out.Entries, e)
		}
		out.Leads, err = leads(ctx, r, f.Root)
		return err
	})
	if err != nil {
		return List{}, err
	}
	out.Roots = named(words, out)
	return out, nil
}

// named words each root l names: its leads', its entries' and the roots
// its files' reasons name.
func named(words map[roots.ID]string, l List) map[roots.ID]string {
	out := map[roots.ID]string{}
	for _, lead := range l.Leads {
		out[lead.Root] = words[lead.Root]
	}
	for _, e := range l.Entries {
		out[e.Root] = words[e.Root]
		for _, f := range e.Files {
			if id, err := storage.ParseUUID(f.Reason.Root); err == nil {
				out[roots.ID{UUID: id}] = words[roots.ID{UUID: id}]
			}
		}
	}
	return out
}

type typedKey struct {
	t   works.TypeID
	key works.Key
}

func keysIn(ctx context.Context, r storage.Reader[file]) (map[typedKey]works.WorkID, error) {
	all, err := works.Summaries(ctx, r)
	if err != nil {
		return nil, err
	}
	out := make(map[typedKey]works.WorkID, len(all))
	for _, w := range all {
		out[typedKey{w.ContentType, w.Key}] = w.ID
	}
	return out, nil
}

// leads is a Lead per root a comparison finished at, of root alone when
// it is set.
func leads(ctx context.Context, r storage.Reader[file], root *roots.ID) ([]Lead, error) {
	q := db.New(r)
	compared, err := q.ListCompared(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := q.CountNeedsYouByRoot(ctx)
	if err != nil {
		return nil, err
	}
	all, err := roots.All(ctx, r)
	if err != nil {
		return nil, err
	}
	var out []Lead
	for _, c := range compared {
		if c.ComparedAt.IsZero() || root != nil && c.RootID != *root {
			continue
		}
		l := Lead{Root: c.RootID, Added: int(c.Added)}
		if i := slices.IndexFunc(all, func(r roots.Root) bool { return r.ID == c.RootID }); i >= 0 {
			l.ContentType = all[i].ContentType
		}
		if i := slices.IndexFunc(counts, func(n db.CountNeedsYouByRootRow) bool { return n.RootID == c.RootID }); i >= 0 {
			l.NeedsYou = int(counts[i].Entries)
		}
		out = append(out, l)
	}
	return out, nil
}

// Count is how many entries need the Admin, for the shell's count.
func (im *Imports) Count(ctx context.Context) (int, error) {
	n, err := db.New(im.cfg.Library).CountNeedsYou(ctx)
	return int(n), err
}

// entries is every entry with its files, in root and folder order.
func entries(ctx context.Context, r storage.Reader[file]) ([]Entry, error) {
	q := db.New(r)
	rows, err := q.ListEntries(ctx)
	if err != nil {
		return nil, err
	}
	fileRows, err := q.ListFiles(ctx)
	if err != nil {
		return nil, err
	}
	of := map[EntryID][]db.ImportsPendingFile{}
	for _, f := range fileRows {
		of[f.PendingID] = append(of[f.PendingID], f)
	}
	out := make([]Entry, len(rows))
	for i, row := range rows {
		if out[i], err = entryOf(row, of[row.ID]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// entry reads one entry with its files, ErrNoEntry when none has id.
func entry(ctx context.Context, r storage.Reader[file], id EntryID) (Entry, error) {
	q := db.New(r)
	row, err := q.GetEntry(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Entry{}, ErrNoEntry
	}
	if err != nil {
		return Entry{}, err
	}
	fs, err := q.ListFilesOf(ctx, id)
	if err != nil {
		return Entry{}, err
	}
	return entryOf(row, fs)
}

func entryOf(row db.ImportsPending, fs []db.ImportsPendingFile) (Entry, error) {
	e := Entry{ID: row.ID, Root: row.RootID, Folder: row.Folder, ContentType: row.ContentType, FoundAt: row.FoundAt, IgnoredAt: row.IgnoredAt}
	if err := json.Unmarshal([]byte(row.Hint), &e.Hint); err != nil {
		return Entry{}, fmt.Errorf("pending import %s: its hint: %w", row.ID, err)
	}
	if row.Suggestion != nil {
		e.Suggestion = &Suggestion{}
		if err := json.Unmarshal([]byte(*row.Suggestion), e.Suggestion); err != nil {
			return Entry{}, fmt.Errorf("pending import %s: its suggestion: %w", row.ID, err)
		}
	}
	e.Files = make([]PendingFile, len(fs))
	for i, f := range fs {
		pf, err := pendingFileOf(f)
		if err != nil {
			return Entry{}, fmt.Errorf("pending import %s: %w", row.ID, err)
		}
		e.Files[i] = pf
	}
	return e, nil
}

func pendingFileOf(row db.ImportsPendingFile) (PendingFile, error) {
	f := PendingFile{Path: row.Path, Size: row.Size, Media: row.Probe, Cut: row.Cut}
	if row.Quality != nil {
		f.Quality = *row.Quality
	}
	if !row.ModifiedAt.IsZero() {
		f.Seen = &storing.Seen{Modified: row.ModifiedAt.Time()}
		if row.NodeFileID != nil {
			f.Seen.ID = storing.FileID{Volume: *row.NodeVolumeID, File: *row.NodeFileID}
		}
	}
	var items []string
	if err := json.Unmarshal([]byte(row.Items), &items); err != nil {
		return PendingFile{}, fmt.Errorf("the items of %s: %w", row.Path, err)
	}
	for _, s := range items {
		id, err := storage.ParseUUID(s)
		if err != nil {
			return PendingFile{}, fmt.Errorf("an item of %s: %w", row.Path, err)
		}
		f.Items = append(f.Items, works.ItemID{UUID: id})
	}
	if err := json.Unmarshal([]byte(row.Reason), &f.Reason); err != nil {
		return PendingFile{}, fmt.Errorf("the reason of %s: %w", row.Path, err)
	}
	return f, nil
}

// write stores e and its files over what its id held, or deletes it when
// it has no files left.
func write(ctx context.Context, tx *storage.Tx[file], e Entry) error {
	q := db.New(tx)
	if err := q.DeleteEntryFiles(ctx, e.ID); err != nil {
		return err
	}
	if len(e.Files) == 0 {
		return q.DeleteEntry(ctx, e.ID)
	}
	hint, err := json.Marshal(e.Hint)
	if err != nil {
		return fmt.Errorf("pending import %s: its hint: %w", e.ID, err)
	}
	var suggestion *string
	if e.Suggestion != nil {
		b, err := json.Marshal(e.Suggestion)
		if err != nil {
			return fmt.Errorf("pending import %s: its suggestion: %w", e.ID, err)
		}
		suggestion = new(string(b))
	}
	if err := q.UpdateEntry(ctx, db.UpdateEntryParams{ID: e.ID, Hint: string(hint), Suggestion: suggestion, IgnoredAt: e.IgnoredAt}); err != nil {
		return err
	}
	for _, f := range e.Files {
		if err := insertFile(ctx, q, e.ID, f); err != nil {
			return err
		}
	}
	return nil
}

func insertFile(ctx context.Context, q *db.Queries, id EntryID, f PendingFile) error {
	items := make([]string, len(f.Items))
	for i, it := range f.Items {
		items[i] = it.String()
	}
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("the items of %s: %w", f.Path, err)
	}
	reason, err := json.Marshal(f.Reason)
	if err != nil {
		return fmt.Errorf("the reason of %s: %w", f.Path, err)
	}
	p := db.InsertEntryFileParams{PendingID: id, Path: f.Path, Size: f.Size, Probe: f.Media, Cut: f.Cut, Items: string(itemsJSON), Reason: string(reason)}
	if f.Media != nil {
		p.Quality = &f.Quality
	}
	if f.Seen != nil {
		p.ModifiedAt = storage.FromTime(f.Seen.Modified)
		if !f.Seen.ID.IsZero() {
			p.NodeFileID, p.NodeVolumeID = &f.Seen.ID.File, &f.Seen.ID.Volume
		}
	}
	return q.InsertEntryFile(ctx, p)
}
