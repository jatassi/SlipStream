package files

import (
	"context"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// PathChangeKind is what happened to a library file's bytes, as a Media
// Server that lists its root sees it.
type PathChangeKind uint8

const (
	// PathAdded is bytes at Path that no server listed there.
	PathAdded PathChangeKind = iota + 1
	// PathReplaced is bytes at Path changed in place: a scan saw a new size.
	PathReplaced
	// PathRemoved is bytes gone from Path for good.
	PathRemoved
	// PathMoved is bytes gone from From to Path: a finished rename, or a
	// scan's moved file.
	PathMoved
	// PathUnmoved is a rename put back: the bytes are at Path, where they
	// were before Prepare saw the rename.
	PathUnmoved
)

// PathChange is one library file change, told in the Write that records
// it. No path of a change lies in a root's holding folder.
type PathChange struct {
	Kind PathChangeKind
	File ID
	Root roots.ID
	// Path is where the bytes are now, or for PathRemoved where they were.
	Path storing.Rel
	// From is where a PathMoved's bytes were.
	From storing.Rel
	// ReplacedBy is, on a PathRemoved, the file an upgrade put in this
	// file's cells. Its PathAdded pairs with this change.
	ReplacedBy ID
	// Upgrade is, on a PathAdded, that a PathRemoved naming File as its
	// replacer will follow.
	Upgrade bool
}

// Rename is a pending change of the path a server lists a file at.
type Rename struct {
	File     ID
	Root     roots.ID
	From, To storing.Rel
	// ReplacedBy is set when File is a doomed file whose replacer belongs
	// at To, another name: an upgrade. Its PathRemoved carries it.
	ReplacedBy ID
	// Leaving is whether the bytes still sit at From.
	Leaving bool
}

// MediaServers is what files tells about the library's file changes.
type MediaServers interface {
	// Changed records cs in tx, the Write that records the change of the
	// bytes. An error fails the Write.
	Changed(ctx context.Context, tx *storage.Tx[storage.Library], cs []PathChange) error
	// Prepare runs once per pass of the disk queue, with no Write open,
	// before any byte of a leaving rename moves or is removed. It lists
	// every pending rename. It returns within its own bound and fails
	// nothing: the bytes move with whatever it managed.
	Prepare(ctx context.Context, renames []Rename)
}

// NoMediaServers is the MediaServers of a library no server lists.
type NoMediaServers struct{}

func (NoMediaServers) Changed(context.Context, *storage.Tx[storage.Library], []PathChange) error {
	return nil
}

func (NoMediaServers) Prepare(context.Context, []Rename) {}

// fact is what a Write did to one file's bytes: from the path of its root
// they sat at before to the one they sit at after, "" for neither.
// resized is bytes changed in place.
type fact struct {
	file       ID
	root       roots.ID
	from, to   string
	resized    bool
	replacedBy ID
}

// shown is p, "" when p is in holding, where no server lists a file.
func shown(p string) string {
	if isHeld(p) {
		return ""
	}
	return p
}

// change is x as a server sees it, false when it sees nothing.
func (x fact) change() (PathChange, bool, error) {
	from, to := shown(x.from), shown(x.to)
	c := PathChange{File: x.file, Root: x.root, ReplacedBy: x.replacedBy}
	at := to
	switch {
	case from == "" && to == "":
		return c, false, nil
	case from == "":
		c.Kind = PathAdded
	case to == "":
		c.Kind, at = PathRemoved, from
	case from == to && x.resized:
		c.Kind = PathReplaced
	case from == to:
		c.Kind = PathUnmoved
	default:
		c.Kind = PathMoved
		var err error
		if c.From, err = storing.ParseRel(from); err != nil {
			return c, false, err
		}
	}
	var err error
	c.Path, err = storing.ParseRel(at)
	return c, err == nil, err
}

// tell has the Media Servers record facts in tx, the Write that records
// them.
func (f *Files) tell(ctx context.Context, tx *storage.Tx[file], facts []fact) error {
	q := db.New(tx)
	var cs []PathChange
	for _, x := range facts {
		c, ok, err := x.change()
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if c.Kind == PathAdded {
			if c.Upgrade, err = q.AwaitsRemoval(ctx, db.AwaitsRemovalParams{FileID: c.File, Holding: storing.HoldingDir + "/"}); err != nil {
				return err
			}
		}
		cs = append(cs, c)
	}
	if len(cs) == 0 {
		return nil
	}
	return f.cfg.MediaServers.Changed(ctx, tx, cs)
}

// named is the path a doomed file's removal names: the origin of the
// move it was doomed in, else its path.
func named(d db.FilesDoomed) string {
	if d.Origin != nil {
		return *d.Origin
	}
	return d.Path
}

// leaves reports whether m's next step takes bytes off a path a server
// lists: they still sit at its origin, which isn't its path.
func (m move) leaves() bool { return m.sits == m.origin && m.renames() }

// renames reports whether m changes the path a server lists its file at.
func (m move) renames() bool { return !isHeld(m.origin) && !isHeld(m.path) && m.origin != m.path }

// renames is every pending rename of q: each move that renames, and each
// doomed file whose replacer belongs under another name.
func (q queue) renames() ([]Rename, error) {
	var out []Rename
	add := func(r Rename, from, to string) error {
		var err error
		if r.From, err = storing.ParseRel(from); err != nil {
			return err
		}
		if r.To, err = storing.ParseRel(to); err != nil {
			return err
		}
		out = append(out, r)
		return nil
	}
	for _, m := range q.moves {
		if !m.renames() {
			continue
		}
		if err := add(Rename{File: m.file, Root: m.root, Leaving: m.sits == m.origin}, m.origin, m.path); err != nil {
			return nil, err
		}
	}
	for _, d := range q.doomed {
		from, to := named(d), q.replacers[d.FileID]
		if to == "" || isHeld(from) || isHeld(to) || from == to {
			continue
		}
		if err := add(Rename{File: d.FileID, Root: d.RootID, ReplacedBy: d.ReplacedBy, Leaving: true}, from, to); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// IDsAt reads, in r, the file of root whose bytes sit at each of rels, gone
// files included: at its move's place while one is pending, else at its
// path. A rel no file's bytes sit at is left out.
func IDsAt(ctx context.Context, r storage.Reader[file], root roots.ID, rels []storing.Rel) (map[storing.Rel]ID, error) {
	paths := make([]string, len(rels))
	for i, rel := range rels {
		paths[i] = rel.String()
	}
	rows, err := db.New(r).IDsAt(ctx, db.IDsAtParams{RootID: root, Sits: paths, Paths: paths})
	if err != nil {
		return nil, err
	}
	out := make(map[storing.Rel]ID, len(rows))
	for _, row := range rows {
		rel, err := storing.ParseRel(row.Place)
		if err != nil {
			return nil, err
		}
		out[rel] = row.ID
	}
	return out, nil
}
