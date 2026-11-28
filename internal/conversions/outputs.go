package conversions

import (
	"context"
	"fmt"
	"path"
	"strconv"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
)

// sourceJob is the source copy's Transfer job: one per Node and source
// file, which every Conversion of that source on that Node reads.
func sourceJob(node role.NodeID, src files.ID) string {
	return "src/" + node.String() + "/" + src.String()
}

// scratchSource is the source copy's name under the scratch folder.
func scratchSource(src files.Located) string {
	return "in/s-" + src.ID.String() + path.Ext(src.Rel.String())
}

// outputJob is the output Transfer's job for one attempt.
func outputJob(id ID, attempt int) string { return "out/" + id.String() + "/" + strconv.Itoa(attempt) }

// outputName is an attempt's output file name: c-<id>-<attempt>.<ext>.
func outputName(id ID, attempt int, c encode.Container) string {
	return "c-" + id.String() + "-" + strconv.Itoa(attempt) + "." + string(c)
}

// scratchOutput is an attempt's output name under the scratch folder.
func scratchOutput(id ID, attempt int, c encode.Container) string {
	return "out/" + outputName(id, attempt, c)
}

// holdingRel is an attempt's output in its root's holding folder.
func holdingRel(id ID, attempt int, c encode.Container) storing.Rel {
	rel, err := storing.ParseRel(storing.HoldingDir + "/" + outputName(id, attempt, c))
	if err != nil {
		panic(fmt.Sprintf("conversions: holding name: %v", err))
	}
	return rel
}

// OutputsIn is the bytes unfinished Conversions on node have yet to write
// into a root's holding folder there, for imports.OtherCopies.
func (c *Conversions) OutputsIn(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID, _ string) (int64, error) {
	rows, err := db.New(r).ListWritingOn(ctx, node)
	if err != nil || len(rows) == 0 {
		return 0, err
	}
	all, err := targets.All(ctx, r)
	if err != nil {
		return 0, err
	}
	settings := map[targets.TargetID]encode.Settings{}
	for _, t := range all {
		if t.Conversion != nil {
			settings[t.ID] = t.Conversion.Settings
		}
	}
	var n int64
	for _, row := range rows {
		s, ok := settings[row.TargetID]
		if !ok || row.SourceFileID.IsZero() {
			continue
		}
		src, err := c.cfg.Files.Located(ctx, r, row.SourceFileID)
		if err != nil {
			return 0, err
		}
		if src.Node == node {
			n += encode.Bytes(src.Media, s)
		}
	}
	return n, nil
}
