package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Seam is files.Config.Conversions. It holds no state, so it is built
// before files.
type Seam struct{ Poke *Poke }

var _ files.Conversions = Seam{}

func (Seam) Cells(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[files.CellKey]files.ConversionCell, error) {
	return nil, errNotBuilt
}

// Holding is c-<id>-<n>.<ext> and its .part for each unfinished
// Conversion writing into a root's holding folder.
func (Seam) Holding(context.Context, storage.Reader[storage.Library]) (map[roots.ID]map[string]bool, error) {
	return nil, errNotBuilt
}

func (Seam) Queue(context.Context, *storage.Tx[storage.Library], ...files.Ask) error {
	return errNotBuilt
}

func (Seam) Leaving(context.Context, *storage.Tx[storage.Library], files.File, files.ID) error {
	return errNotBuilt
}

func (Seam) Changed(context.Context, *storage.Tx[storage.Library], []files.PathChange) error {
	return errNotBuilt
}

func (Seam) Remakes(context.Context, *storage.Tx[storage.Library], targets.TargetID, int, []works.WorkID, bool) (files.RemakeImpact, error) {
	return files.RemakeImpact{}, errNotBuilt
}
