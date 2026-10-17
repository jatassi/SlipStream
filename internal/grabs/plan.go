package grabs

import (
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// errFillsNothing is a probed file that fills no unmet cell.
type errFillsNothing struct{ res release.Resolution }

func (e errFillsNothing) Error() string {
	return "a " + e.res.String() + " file fills no Target the Work still wants"
}

// plan places a probed file of item: it fills each of the item's Standard
// cells that are not met whose Target res meets, in rank order, and is
// named after the first. ext is the file's extension.
func plan(cells []files.Cell, item works.ItemID, res release.Resolution, root roots.Root, t naming.Templates, tokens map[string]string, ext string) (Plan, error) {
	var fills []targets.Target
	for _, c := range cells {
		if c.Item.ID == item && c.Edition == files.Standard && c.State != files.Met && slices.Contains(c.Target.Resolutions, res) {
			fills = append(fills, c.Target)
		}
	}
	if len(fills) == 0 {
		return Plan{}, errFillsNothing{res}
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
