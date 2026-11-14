package imports

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Suggestion is a root Import's best candidate at its type's source.
type Suggestion struct {
	Key       works.Key   `json:"key"`
	Label     string      `json:"label"`
	Year      int         `json:"year,omitempty"`
	Poster    artwork.Ref `json:"poster"`
	Confident bool        `json:"confident,omitempty"`
	// WorkID is the library Work with Key, read when the Import is, zero
	// while none has it.
	WorkID works.WorkID `json:"-"`
}

var (
	ErrNoImport     = errors.New("no Import has that id")
	ErrNoSuggestion = errors.New("the Import has no suggested Work")
	ErrElsewhere    = errors.New("the Work lives in another library root")
	ErrNotOfWork    = errors.New("a file names an Item the Work doesn't hold")
	ErrChanged      = errors.New("the Import no longer lists a chosen file")
	ErrListMoved    = errors.New("the Import a page ended at is no longer listed")
)

// Filter picks unfinished Imports: of one source or every source, one
// root's or every root's, one Work's or every Work's, the ones that need
// the Admin or the ignored ones, and with Waiting only those with a file
// waiting. A page starts past After and holds at most Limit, every one
// when Limit is 0.
type Filter struct {
	Source  *Source
	Root    *roots.ID
	Work    *works.WorkID
	Ignored bool
	Waiting bool
	After   *ImportID
	Limit   int
}

func (f Filter) picks(im Import) bool {
	return im.IsIgnored() == f.Ignored && (f.Source == nil || im.Source == *f.Source) && (f.Root == nil || im.Root == *f.Root) &&
		(f.Work == nil || im.Work == *f.Work) && (!f.Waiting || slices.ContainsFunc(im.Files, func(f File) bool { return f.Outcome == Waiting }))
}

// page cuts ims to f's page, and names the Import the next page starts
// after, nil on the last page. ErrListMoved when After is no longer
// listed.
func (f Filter) page(ims []Import) ([]Import, *ImportID, error) {
	if f.After != nil {
		i := slices.IndexFunc(ims, func(im Import) bool { return im.ID == *f.After })
		if i < 0 {
			return nil, nil, ErrListMoved
		}
		ims = ims[i+1:]
	}
	if f.Limit == 0 || len(ims) <= f.Limit {
		return ims, nil, nil
	}
	ims = ims[:f.Limit]
	return ims, &ims[len(ims)-1].ID, nil
}

// List is Pending imports: what each root's last comparison added and
// how many root Imports need the Admin, then the Imports.
type List struct {
	Leads   []Lead
	Imports []Import
	// Next is the Import the next page starts after, nil on the last page.
	Next *ImportID
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

// List lists the unfinished Imports f picks: the root Imports in folder
// order within each root, then the rest by folder.
func (im *Imports) List(ctx context.Context, _ identity.Admin, f Filter) (List, error) {
	words, err := im.cfg.Roots.Words(ctx)
	if err != nil {
		return List{}, err
	}
	var out List
	err = im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := db.New(r).ListUnfinishedImports(ctx)
		if err != nil {
			return err
		}
		all, err := allImports(ctx, r, rows)
		if err != nil {
			return err
		}
		inLibrary, err := keysIn(ctx, r)
		if err != nil {
			return err
		}
		for _, e := range all {
			if f.picks(e) {
				out.Imports = append(out.Imports, withWork(e, inLibrary))
			}
		}
		if out.Imports, out.Next, err = f.page(out.Imports); err != nil {
			return err
		}
		if f.Source == nil || *f.Source == FromRoot {
			out.Leads, err = leads(ctx, r, f.Root)
		}
		return err
	})
	if err != nil {
		return List{}, err
	}
	out.Roots = named(words, out)
	return out, nil
}

// Get is one Import with its files, ErrNoImport when none has id, and
// the words of every root it names.
func (im *Imports) Get(ctx context.Context, _ identity.Admin, id ImportID) (Import, map[roots.ID]string, error) {
	words, err := im.cfg.Roots.Words(ctx)
	if err != nil {
		return Import{}, nil, err
	}
	var out Import
	err = im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if out, err = importOf(ctx, r, id); err != nil {
			return err
		}
		inLibrary, err := keysIn(ctx, r)
		out = withWork(out, inLibrary)
		return err
	})
	if err != nil {
		return Import{}, nil, err
	}
	return out, named(words, List{Imports: []Import{out}}), nil
}

func withWork(im Import, inLibrary map[typedKey]works.WorkID) Import {
	if im.Suggestion != nil {
		s := *im.Suggestion
		s.WorkID = inLibrary[typedKey{im.ContentType, s.Key}]
		im.Suggestion = &s
	}
	return im
}

// named words each root l names: its leads', its Imports' and the roots
// its files' reasons name.
func named(words map[roots.ID]string, l List) map[roots.ID]string {
	out := map[roots.ID]string{}
	for _, lead := range l.Leads {
		out[lead.Root] = words[lead.Root]
	}
	for _, e := range l.Imports {
		if !e.Root.IsZero() {
			out[e.Root] = words[e.Root]
		}
		for _, f := range e.Files {
			if id, err := storage.ParseUUID(f.Scan().Root); err == nil {
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

// Count is how many Imports of any source have a file waiting for the
// Admin, ignored ones aside, for the shell's count.
func (im *Imports) Count(ctx context.Context) (int, error) {
	n, err := db.New(im.cfg.Library).CountWaiting(ctx)
	return int(n), err
}

// unfinishedRootImports is every unfinished root Import with its files,
// in root and folder order.
func unfinishedRootImports(ctx context.Context, r storage.Reader[file]) ([]Import, error) {
	rows, err := db.New(r).ListUnfinishedRootImports(ctx)
	if err != nil {
		return nil, err
	}
	return allImports(ctx, r, rows)
}
