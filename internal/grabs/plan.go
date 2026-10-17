package grabs

import (
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// errFillsNothing is a probed file that fills no unfilled cell.
type errFillsNothing struct{ quality release.Fact[release.Quality] }

func (e errFillsNothing) Error() string {
	q, known := e.quality.Get()
	if !known {
		return "a file of unknown quality fills no Target the Work still wants"
	}
	return "a " + q.Info().Name + " file fills no Target the Work still wants"
}

// plan places a probed file of item with facts: it fills each of the
// item's Standard cells no file fills whose Target it meets, in Profile
// order, and is named after the first. ext is the file's extension.
func plan(cells []files.Cell, item works.ItemID, facts release.Facts, root roots.Root, t naming.Templates, tokens map[string]string, ext string) (Plan, error) {
	var fills []targets.Target
	for _, c := range cells {
		if c.Item.ID == item && c.Edition == files.Standard && c.File == nil && targets.Judge(c.Target, facts, nil).Meets() {
			fills = append(fills, c.Target)
		}
	}
	if len(fills) == 0 {
		return Plan{}, errFillsNothing{facts.Quality}
	}
	core := naming.Core{Target: fills[0].Name}
	folder, err := naming.Render(t.Folder, tokens, core)
	if err != nil {
		return Plan{}, err
	}
	name, err := naming.Render(t.File, tokens, core)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Root: root, Path: folder + "/" + name + "." + ext}
	for _, f := range fills {
		p.Fills = append(p.Fills, f.ID)
	}
	return p, nil
}
