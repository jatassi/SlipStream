package targets

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// Origin is where a cut comes from.
type Origin uint8

const (
	BuiltIn Origin = iota + 1
	FromType
	Custom
)

// ListedCut is a cut an Edition list can name, where it comes from, and
// the Content Type that adds it, for FromType. A custom cut's Aliases
// start with its name, which titles name it by too.
type ListedCut struct {
	release.Cut
	Origin Origin
	Type   works.TypeID
}

// Cuts is every cut SlipStream knows: the built-in ones, each Content
// Type's by type, then the Admin's custom ones by name.
type Cuts struct{ all []ListedCut }

var (
	ErrNoCut           = errors.New("no cut has that key")
	ErrCutNameTaken    = errors.New("a cut already has that name, alias or key")
	ErrBadCutName      = errors.New("a cut's name and each alias need a word a title can name")
	ErrStandardIsFixed = errors.New("the Standard cut can't be renamed or removed")
	ErrCutNotCustom    = errors.New("only a custom cut can be changed or removed")
)

func (c Cuts) All() []ListedCut { return slices.Clone(c.all) }

// For is every cut a title of Content Type typ can name: the built-in
// ones, typ's own and the custom ones.
func (c Cuts) For(typ works.TypeID) []release.Cut {
	var out []release.Cut
	for _, cut := range c.all {
		if cut.Origin != FromType || cut.Type == typ {
			out = append(out, cut.Cut)
		}
	}
	return out
}

// Get is the cut with key k, false when none has it.
func (c Cuts) Get(k release.CutKey) (ListedCut, bool) {
	i := slices.IndexFunc(c.all, func(cut ListedCut) bool { return cut.Key == k })
	if i < 0 {
		return ListedCut{}, false
	}
	return c.all[i], true
}

// check refuses e when it names a cut a Work of Content Type typ can't
// hold.
func (c Cuts) check(e Editions, typ works.TypeID) error {
	for _, k := range e.cuts {
		if !slices.ContainsFunc(c.For(typ), func(cut release.Cut) bool { return cut.Key == k }) {
			return fmt.Errorf("%w: %s", ErrNoCut, k)
		}
	}
	return nil
}

// custom refuses k unless it is a custom cut, which an Admin can change.
func (c Cuts) custom(k release.CutKey) error {
	cut, ok := c.Get(k)
	switch {
	case k == release.Standard:
		return ErrStandardIsFixed
	case !ok:
		return ErrNoCut
	case cut.Origin != Custom:
		return ErrCutNotCustom
	}
	return nil
}

// CutSpec is a custom cut as an Admin writes it: its name and the other
// phrases that name it in a title.
type CutSpec struct {
	Name    string
	Aliases []string
}

// aliases is s's aliases with any that reads as its name or an earlier
// alias left out, or ErrBadCutName when its name or one reads as no word.
func (s CutSpec) aliases() ([]string, error) {
	if release.Slug(s.Name) == "" {
		return nil, ErrBadCutName
	}
	var out []string
	for _, a := range s.Aliases {
		switch {
		case release.Slug(a) == "":
			return nil, ErrBadCutName
		case release.SamePhrase(a, s.Name) || slices.ContainsFunc(out, func(o string) bool { return release.SamePhrase(a, o) }):
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

// claimed reports whether a cut but self is named by a phrase of s.
func (c Cuts) claimed(s CutSpec, self release.CutKey) bool {
	phrases := append([]string{s.Name}, s.Aliases...)
	for _, cut := range c.all {
		if cut.Key == self {
			continue
		}
		for _, theirs := range append([]string{cut.Name}, cut.Aliases...) {
			if slices.ContainsFunc(phrases, func(p string) bool { return release.SamePhrase(p, theirs) }) {
				return true
			}
		}
	}
	return false
}

// CutsIn reads every cut in r.
func (t *Targets) CutsIn(ctx context.Context, r storage.Reader[file]) (Cuts, error) {
	var out Cuts
	for _, c := range release.BuiltIn() {
		out.all = append(out.all, ListedCut{Cut: c, Origin: BuiltIn})
	}
	for _, id := range slices.Sorted(maps.Keys(t.cfg.Types)) {
		for _, c := range t.cfg.Types[id].Cuts() {
			out.all = append(out.all, ListedCut{Cut: c, Origin: FromType, Type: id})
		}
	}
	q := db.New(r)
	rows, err := q.ListCuts(ctx)
	if err != nil {
		return Cuts{}, err
	}
	aliases, err := q.ListCutAliases(ctx)
	if err != nil {
		return Cuts{}, err
	}
	of := map[release.CutKey][]string{}
	for _, a := range aliases {
		of[a.Cut] = append(of[a.Cut], a.Alias)
	}
	for _, row := range rows {
		out.all = append(out.all, ListedCut{Cut: release.Cut{Key: row.Key, Name: row.Name, Aliases: append([]string{row.Name}, of[row.Key]...)}, Origin: Custom})
	}
	return out, nil
}

// ListCuts reads every cut.
func (t *Targets) ListCuts(ctx context.Context, _ identity.Admin) (Cuts, error) {
	var out Cuts
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = t.CutsIn(ctx, r)
		return err
	})
	return out, err
}

// CreateCut adds a custom cut keyed by its name's slug. A name, alias or
// key another cut has is ErrCutNameTaken.
func (t *Targets) CreateCut(ctx context.Context, _ identity.Admin, s CutSpec) (ListedCut, error) {
	aliases, err := s.aliases()
	if err != nil {
		return ListedCut{}, err
	}
	s.Aliases = aliases
	made := ListedCut{Cut: release.Cut{Key: release.Slug(s.Name), Name: s.Name, Aliases: append([]string{s.Name}, aliases...)}, Origin: Custom}
	err = t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		known, err := t.CutsIn(ctx, tx)
		if err != nil {
			return err
		}
		if _, taken := known.Get(made.Key); taken || made.Key == release.CutKey(AnyEdition) || known.claimed(s, "") {
			return ErrCutNameTaken
		}
		q := db.New(tx)
		if err := q.InsertCut(ctx, db.InsertCutParams{Key: made.Key, Name: s.Name, CreatedAt: storage.Now()}); err != nil {
			return err
		}
		return insertAliases(ctx, q, made.Key, aliases)
	})
	if err != nil {
		return ListedCut{}, err
	}
	t.cfg.Changed.Publish(ctx, changedOf(nil))
	return made, nil
}

func insertAliases(ctx context.Context, q *db.Queries, k release.CutKey, aliases []string) error {
	for i, a := range aliases {
		if err := q.InsertCutAlias(ctx, db.InsertCutAliasParams{Cut: k, Position: int64(i + 1), Alias: a}); err != nil {
			return err
		}
	}
	return nil
}

// Held counts the library's files of a cut, in r: files' answer, which
// DeleteCut reads in its own Write.
type Held func(ctx context.Context, r storage.Reader[storage.Library], cut release.CutKey) (int, error)

// DeleteCut removes custom cut k and its sightings, or refuses with
// *InUseError naming each Profile whose list names it, the Works' own
// lists that do, and the files held counts of it.
func (t *Targets) DeleteCut(ctx context.Context, _ identity.Admin, k release.CutKey, held Held) error {
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		known, err := t.CutsIn(ctx, tx)
		if err != nil {
			return err
		}
		if err := known.custom(k); err != nil {
			return err
		}
		q := db.New(tx)
		by, err := q.ListProfilesNamingCut(ctx, k)
		if err != nil {
			return err
		}
		own, err := q.CountOwnListsNamingCut(ctx, k)
		if err != nil {
			return err
		}
		n, err := held(ctx, tx, k)
		if err != nil {
			return err
		}
		if own > 0 {
			by = append(by, counted(int(own), "Work's own list", "Works' own lists"))
		}
		if n > 0 {
			by = append(by, counted(n, "file", "files"))
		}
		if len(by) > 0 {
			return &InUseError{By: by}
		}
		if err := q.DeleteSightingsOfCut(ctx, k); err != nil {
			return err
		}
		return q.DeleteCut(ctx, k)
	})
	if err != nil {
		return err
	}
	t.cfg.Changed.Publish(ctx, changedOf(nil))
	return nil
}

func counted(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
