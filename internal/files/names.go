package files

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
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

// names names every file of w under fills. A filling file takes the first
// Target in Profile order it fills. A Kept file takes the first Target it
// meets, else its current label, plus " (extra N)": its N when no other
// Kept file in its folder holds it, else the lowest free there. A Kept
// file that never had a label and meets no Target takes the Target its
// Grab was meant to fill, else its quality's name. A file whose label, N
// and cut's name are unchanged keeps its path; one that changes keeps its
// folder and takes the template's name, its cut tagging it unless
// Standard. Two live files on one path is ErrNameClash; a gone file keeps
// its name and path whatever the live files take.
func names(w world, fills map[CellKey]ID) (map[ID]name, error) {
	out, err := labels(w, fills)
	if err != nil {
		return nil, err
	}
	held := map[place]bool{}
	for _, f := range w.files {
		nm := out[f.ID]
		nm.Path = f.Path
		if f.Gone() {
			out[f.ID] = nm
			continue
		}
		if renamed(w, f, nm) {
			if nm.Path, err = pathOf(w, f, nm); err != nil {
				return nil, err
			}
		}
		at := place{f.RootID, nm.Path}
		if held[at] {
			return nil, fmt.Errorf("%w: %s", ErrNameClash, nm.Path)
		}
		held[at] = true
		out[f.ID] = nm
	}
	return out, nil
}

// labels gives every file of w its label and N under fills, with no path.
// A gone file keeps its own and holds its N. Kept files keep their N
// next, in id order, then the rest take the lowest free.
func labels(w world, fills map[CellKey]ID) (map[ID]name, error) {
	out := make(map[ID]name, len(w.files))
	taken := counters{}
	for _, f := range w.files {
		if f.Gone() {
			out[f.ID] = name{Label: f.Label, Extra: f.Extra}
			taken.hold(path.Dir(f.Path), f.Extra)
		}
	}
	folders := map[ID]string{}
	var waiting []File
	for _, f := range w.files {
		if f.Gone() {
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
		if f.Extra > 0 && !taken[folder][f.Extra] {
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

// counters is each folder's N that Kept files hold.
type counters map[string]map[int64]bool

// hold takes n in folder; 0 is no N.
func (c counters) hold(folder string, n int64) {
	if n == 0 {
		return
	}
	if c[folder] == nil {
		c[folder] = map[int64]bool{}
	}
	c[folder][n] = true
}

// lowest takes the lowest N free in folder.
func (c counters) lowest(folder string) int64 {
	n := int64(1)
	for c[folder][n] {
		n++
	}
	c.hold(folder, n)
	return n
}

// unlabeled is the label of f, a Kept file that never had one and meets
// no Target: the Target its Grab was meant to fill, else its quality's
// name.
func (w world) unlabeled(f File) string {
	if f.ID == w.fresh && w.meant != "" {
		return w.meant
	}
	return f.Quality.Info().Name
}

// renamed reports whether f takes a new path as nm: the file being placed
// does, and so does one whose label, N or cut's name changes, unless a
// scan keeps every path.
func renamed(w world, f File, nm name) bool {
	return !w.keepPaths && (f.ID == w.fresh || nm.Label != f.Label || nm.Extra != f.Extra || w.retagged[f.Cut])
}

// filledBy is the first Target in Profile order f fills.
func filledBy(g targets.Goals, fills map[CellKey]ID, f File) (string, bool) {
	e := g.Editions.Filling(f.Cut)
	for _, t := range g.Targets {
		if fills[CellKey{Item: f.ItemID, Edition: e, Target: t.ID}] == f.ID {
			return t.Name, true
		}
	}
	return "", false
}

// firstMet is the first Target in Profile order f meets.
func firstMet(ts []targets.Target, f File) (string, bool) {
	for _, t := range ts {
		if targets.Judge(t, f.Facts(), nil).Meets() {
			return t.Name, true
		}
	}
	return "", false
}

// folderOf is f's folder: its own, or for the file being placed the
// folder of the oldest live file beside it at its Item's level. Without
// one it is the Work's folder, the first folder of the oldest live file
// beside it that sits in one or else the template's, then the level
// folders the template gives its Item when the Work keeps a folder per
// level.
func folderOf(w world, f File) (string, error) {
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
	tokens := w.tokens[f.ItemID]
	if work == "" {
		var err error
		if work, err = naming.Render(w.templates.Folder, tokens, naming.Core{}); err != nil {
			return "", err
		}
	}
	levels, err := naming.Folders(w.templates, tokens, w.tree.Items[i].Aside, w.tree.LevelFolders)
	if err != nil {
		return "", err
	}
	return path.Join(append([]string{work}, levels...)...), nil
}

func pathOf(w world, f File, nm name) (string, error) {
	folder, err := folderOf(w, f)
	if err != nil {
		return "", err
	}
	core := naming.Core{Target: nm.Label}
	if nm.Extra > 0 {
		core.Target += fmt.Sprintf(" (extra %d)", nm.Extra)
	}
	if f.Cut != release.Standard {
		cut, ok := w.cuts.Get(f.Cut)
		if !ok {
			return "", fmt.Errorf("file %s holds cut %s, which no cut has", f.ID, f.Cut)
		}
		core.EditionTag, core.Edition = "{edition-"+cut.Name+"}", cut.Name
	}
	file, err := naming.Render(w.templates.File, w.tokens[f.ItemID], core)
	if err != nil {
		return "", err
	}
	return path.Join(folder, file+path.Ext(f.Path)), nil
}
