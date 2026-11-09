package files

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// name is a file's place after a refill.
type name struct {
	Label string
	Extra int64
	Path  string
}

var ErrNameClash = errors.New("two files of the Work would take one name")

// renaming is which files names gives a path from the templates.
type renaming uint8

const (
	// changedOnly is an import's and a Change's: the fresh file, and each
	// file whose label, N or cut's name changes, in its own folder.
	changedOnly renaming = iota
	// keepEvery is a scan's: labels and N only.
	keepEvery
	// everyFile is a rename plan's: every live file where the templates
	// put it, its N counted in that folder.
	everyFile
)

// names names every file of w under fills. A filling file takes the first
// Target in Profile order it fills. A Kept file takes the first Target it
// meets, else its current label, plus " (extra N)": its N when no other
// Kept file in its folder holds it, else the lowest free there. A Kept
// file that never had a label and meets no Target takes the Target its
// Grab was meant to fill, else its quality's name. A gone file, and a file
// the disk queue still moves, keeps its label, N and path.
//
// Which other files take a path from the templates is w.mode's. One whose
// label, N and cut's name are unchanged keeps its path in changedOnly
// mode; one that changes keeps its folder and takes the template's name,
// its cut tagging it unless Standard. Two live files ending on one path
// without case, one of them moving there, is ErrNameClash, or a
// *NameTakenError when the file being placed is one that moves.
//
// In everyFile mode a file that doesn't fit takes "" and is blocked
// TooLong, and with renaming off the moving files among those ending on
// one path are blocked Clash with their own Work. A blocked file meets no
// other: a file taking its path joins its group, which waits for it.
func names(w world, fills map[CellKey]Fill) (out map[ID]name, blocked map[ID]Blocked, err error) {
	if out, err = labels(w, fills); err != nil {
		return nil, nil, err
	}
	blocked = map[ID]Blocked{}
	for _, f := range w.files {
		nm := out[f.ID]
		nm.Path = f.Path
		if !w.fixed(f) && renamed(w, f, nm) {
			var long *naming.TooLongError
			nm.Path, err = pathOf(w, f, nm)
			switch {
			case w.mode == everyFile && errors.As(err, &long):
				blocked[f.ID] = Blocked{Reason: TooLong, Long: long}
			case err != nil:
				return nil, nil, err
			}
		}
		out[f.ID] = nm
	}
	return out, blocked, clashes(w, out, blocked)
}

// clashes finds the live files of w that end on one path without case,
// at least one of them moving there. In everyFile mode with renaming off
// they are kept names that meet, and the moving ones are blocked Clash
// with their own Work. Anywhere else it is a *NameTakenError when the
// file being placed moves there, else ErrNameClash.
func clashes(w world, nm map[ID]name, blocked map[ID]Blocked) error {
	at := map[place][]File{}
	for _, f := range w.files {
		if _, ok := blocked[f.ID]; !ok && !f.Gone() {
			end := folded(f.RootID, nm[f.ID].Path)
			at[end] = append(at[end], f)
		}
	}
	for _, met := range at {
		moving := slices.DeleteFunc(slices.Clone(met), func(g File) bool { return nm[g.ID].Path == g.Path })
		if len(met) < 2 || len(moving) == 0 {
			continue
		}
		if w.mode != everyFile || w.scheme.Renames() {
			if slices.ContainsFunc(moving, func(g File) bool { return g.ID == w.fresh }) {
				return &NameTakenError{Path: nm[w.fresh].Path}
			}
			return fmt.Errorf("%w: %s", ErrNameClash, nm[moving[0].ID].Path)
		}
		for _, g := range moving {
			blocked[g.ID] = Blocked{Reason: Clash, Other: w.tree.ID}
		}
	}
	return nil
}

// labels gives every file of w its label and N under fills, with no path.
// A fixed file keeps its own and holds its N in its folder. Kept files
// keep their N next, in id order, in the folder they end in, then the
// rest take the lowest free there.
func labels(w world, fills map[CellKey]Fill) (map[ID]name, error) {
	out := make(map[ID]name, len(w.files))
	taken := counters{}
	for _, f := range w.files {
		if w.fixed(f) {
			out[f.ID] = name{Label: f.Label, Extra: f.Extra}
			taken.hold(path.Dir(f.Path), f.Extra)
		}
	}
	folders := map[ID]string{}
	var waiting []File
	for _, f := range w.files {
		if w.fixed(f) {
			continue
		}
		if label, ok := filledBy(w.goals, fills, f); ok {
			out[f.ID] = name{Label: label}
			continue
		}
		label := f.Label
		if t, ok := firstMet(w.goals.Targets, f); ok {
			label = t
		} else if label == "" {
			label = w.unlabeled(f)
		}
		folder, err := folderOf(w, f)
		if err != nil {
			return nil, err
		}
		folders[f.ID] = folder
		if f.Extra > 0 && !taken.holds(folder, f.Extra) {
			taken.hold(folder, f.Extra)
			out[f.ID] = name{Label: label, Extra: f.Extra}
			continue
		}
		out[f.ID] = name{Label: label}
		waiting = append(waiting, f)
	}
	for _, f := range waiting {
		out[f.ID] = name{Label: out[f.ID].Label, Extra: taken.lowest(folders[f.ID])}
	}
	return out, nil
}

// counters is each folder's N that Kept files hold, by the folder's name
// without case, as a volume that ignores case sees it.
type counters map[string]map[int64]bool

func (c counters) holds(folder string, n int64) bool { return c[storing.Fold(folder)][n] }

// hold takes n in folder; 0 is no N.
func (c counters) hold(folder string, n int64) {
	if n == 0 {
		return
	}
	k := storing.Fold(folder)
	if c[k] == nil {
		c[k] = map[int64]bool{}
	}
	c[k][n] = true
}

// lowest takes the lowest N free in folder.
func (c counters) lowest(folder string) int64 {
	n := int64(1)
	for c.holds(folder, n) {
		n++
	}
	c.hold(folder, n)
	return n
}

// fixed reports whether f keeps its label, N and path whatever the rest
// take: a gone file, and a file the disk queue still moves.
func (w world) fixed(f File) bool { return f.Gone() || w.moving[f.ID] }

// unlabeled is the label of f, a Kept file that never had one and meets
// no Target: the Target its Grab was meant to fill, else its quality's
// name.
func (w world) unlabeled(f File) string {
	if f.ID == w.fresh && w.meant != "" {
		return w.meant
	}
	return f.Quality.Info().Name
}

// renamed reports whether f takes a path from the templates as nm, by
// w.mode: in changedOnly mode the file being placed does, and so does one
// whose label, N or cut's name changes.
func renamed(w world, f File, nm name) bool {
	switch w.mode {
	case changedOnly:
		return f.ID == w.fresh || nm.Label != f.Label || nm.Extra != f.Extra || w.retagged[f.Cut]
	case keepEvery:
		return false
	case everyFile:
		return true
	}
	panic(fmt.Sprintf("files: renaming mode %d", w.mode))
}

// filledBy is the first Target in Profile order f fills.
func filledBy(g targets.Goals, fills map[CellKey]Fill, f File) (string, bool) {
	e := g.Editions.Filling(f.Cut)
	for _, t := range g.Targets {
		if fills[CellKey{Item: f.ItemID, Edition: e, Target: t.ID}].File == f.ID {
			return t.Name, true
		}
	}
	return "", false
}

// firstMet is the first Target in Profile order f meets.
func firstMet(ts []targets.Target, f File) (string, bool) {
	for _, t := range ts {
		if targets.Judge(t, targets.Scored{Facts: f.Facts()}, nil).Meets() {
			return t.Name, true
		}
	}
	return "", false
}

// folderOf is the folder f ends in. In everyFile mode it is the one the
// templates give it. Otherwise it is its own, or for the file being placed
// the folder of the oldest live file beside it at its Item's level.
// Without one it lands in the Work's folder, the first folder of the
// oldest live file beside it that sits in one or else the template's,
// under the level folders the templates give its Item when the Work keeps
// a folder per level.
func folderOf(w world, f File) (string, error) {
	if w.mode == everyFile {
		return w.scheme.Dir(naming.Placing{Subject: w.subject(f), LevelFolders: w.tree.LevelFolders, Landing: naming.Rendered()})
	}
	if f.ID != w.fresh {
		return path.Dir(f.Path), nil
	}
	i := slices.IndexFunc(w.tree.Items, func(it works.Item) bool { return it.ID == f.ItemID })
	if i < 0 {
		return "", fmt.Errorf("file %s names item %s, which work %s doesn't hold", f.ID, f.ItemID, w.tree.ID)
	}
	level := map[works.ItemID]works.LevelID{}
	for _, it := range w.tree.Items {
		level[it.ID] = it.LevelID
	}
	work := ""
	for _, g := range w.beside {
		if g.Gone() {
			continue
		}
		if level[g.ItemID] == w.tree.Items[i].LevelID {
			return path.Dir(g.Path), nil
		}
		if folder, _, ok := strings.Cut(g.Path, "/"); ok && work == "" {
			work = folder
		}
	}
	landing := naming.Rendered()
	if work != "" {
		landing = naming.InWork(work)
	}
	return w.scheme.Dir(naming.Placing{Subject: w.subject(f), LevelFolders: w.tree.LevelFolders, Landing: landing})
}

// pathOf is the path the templates give f, named nm, in its folder.
func pathOf(w world, f File, nm name) (string, error) {
	folder, err := folderOf(w, f)
	if err != nil {
		return "", err
	}
	facts := naming.FileFacts{
		Target: nm.Label, Facts: w.scoring.facts(f), Media: f.Media, Release: f.ReleaseName, Source: f.SourceName, Keep: w.kept(f),
	}
	if nm.Extra > 0 {
		facts.Target += fmt.Sprintf(" (extra %d)", nm.Extra)
	}
	if f.Cut != release.Standard {
		cut, ok := w.cuts.Get(f.Cut)
		if !ok {
			return "", fmt.Errorf("file %s holds cut %s, which no cut has", f.ID, f.Cut)
		}
		facts.Cut = cut.Name
	}
	named, err := w.scheme.Name(naming.Placing{
		Subject: w.subject(f), LevelFolders: w.tree.LevelFolders, Landing: naming.InDir(folder), File: facts, Ext: path.Ext(f.Path), Budget: w.budgets[f.RootID],
	})
	return named.Rel, err
}

// kept is the name without extension f keeps with renaming off: for the
// file being placed the name it had in its download, else its own.
func (w world) kept(f File) string {
	name := path.Base(f.Path)
	if f.ID == w.fresh {
		name = f.SourceName
	}
	return strings.TrimSuffix(name, path.Ext(name))
}

// subject is what f's name reads of its Work.
func (w world) subject(f File) naming.Subject {
	return w.values.Of(w.tree, []works.ItemID{f.ItemID})
}
