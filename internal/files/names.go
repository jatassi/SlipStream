package files

import (
	"errors"
	"fmt"
	"path"

	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/targets"
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
// Kept file in its folder holds it, else the lowest free there. A file
// whose label and N are unchanged keeps its path; one that changes keeps
// its folder and takes the template's name. Two files on one path is
// ErrNameClash.
func names(w world, fills map[CellKey]ID) (map[ID]name, error) {
	out, err := labels(w, fills)
	if err != nil {
		return nil, err
	}
	held := map[place]bool{}
	for _, f := range w.files {
		nm := out[f.ID]
		nm.Path = f.Path
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
// Kept files keep their N first, in id order, then the rest take the
// lowest free.
func labels(w world, fills map[CellKey]ID) (map[ID]name, error) {
	out := make(map[ID]name, len(w.files))
	taken := map[string]map[int64]bool{}
	folders := map[ID]string{}
	var waiting []File
	for _, f := range w.files {
		if label, ok := filledBy(w.goals.Targets, fills, f); ok {
			out[f.ID] = name{Label: label}
			continue
		}
		label := f.Label
		if t, ok := firstMet(w.goals.Targets, f); ok {
			label = t
		}
		folder, err := folderOf(w, f)
		if err != nil {
			return nil, err
		}
		folders[f.ID] = folder
		if taken[folder] == nil {
			taken[folder] = map[int64]bool{}
		}
		if f.Extra > 0 && !taken[folder][f.Extra] {
			taken[folder][f.Extra] = true
			out[f.ID] = name{Label: label, Extra: f.Extra}
			continue
		}
		out[f.ID] = name{Label: label}
		waiting = append(waiting, f)
	}
	for _, f := range waiting {
		in := taken[folders[f.ID]]
		n := int64(1)
		for in[n] {
			n++
		}
		in[n] = true
		out[f.ID] = name{Label: out[f.ID].Label, Extra: n}
	}
	return out, nil
}

// renamed reports whether f takes a new path as nm: the file being placed
// does, and so does one whose label or N changes.
func renamed(w world, f File, nm name) bool {
	return f.ID == w.fresh || nm.Label != f.Label || nm.Extra != f.Extra
}

// filledBy is the first Target in Profile order f fills.
func filledBy(ts []targets.Target, fills map[CellKey]ID, f File) (string, bool) {
	for _, t := range ts {
		if fills[CellKey{Item: f.ItemID, Edition: f.Edition, Target: t.ID}] == f.ID {
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

// folderOf is f's folder: its own, or for the file being placed the one
// the template gives its Item.
func folderOf(w world, f File) (string, error) {
	if f.ID != w.fresh {
		return path.Dir(f.Path), nil
	}
	return naming.Render(w.templates.Folder, w.tokens[f.ItemID], naming.Core{})
}

func pathOf(w world, f File, nm name) (string, error) {
	folder, err := folderOf(w, f)
	if err != nil {
		return "", err
	}
	label := nm.Label
	if nm.Extra > 0 {
		label += fmt.Sprintf(" (extra %d)", nm.Extra)
	}
	file, err := naming.Render(w.templates.File, w.tokens[f.ItemID], naming.Core{Target: label})
	if err != nil {
		return "", err
	}
	return path.Join(folder, file+path.Ext(f.Path)), nil
}
