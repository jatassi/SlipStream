package files

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Conversions is what files asks of S44 (Conversion). NoConversions when
// nil.
type Conversions interface {
	// Cells is the Conversion behind each made-from cell of ws.
	Cells(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[CellKey]ConversionCell, error)
	// Holding is the names in each root's holding folder that an
	// unfinished Conversion writes, which the disk sweep keeps.
	Holding(ctx context.Context, r storage.Reader[file]) (map[roots.ID]map[string]bool, error)
	// Queue asks for the Works' Conversions to be converged after tx
	// commits.
	Queue(ctx context.Context, tx *storage.Tx[file], asks ...Ask) error
	// Leaving is told, in its doom's Write, that d goes, replaced by by or
	// by nothing when by is zero.
	Leaving(ctx context.Context, tx *storage.Tx[file], d File, by ID) error
	Changed(ctx context.Context, tx *storage.Tx[file], cs []PathChange) error
	// Cost is what making made would take.
	Cost(ctx context.Context, tx *storage.Tx[file], made []MadePair) (MadeCost, error)
	// Remakes is the Works of ws whose files t made with a version before
	// version, queued for a remake unless preview.
	Remakes(ctx context.Context, tx *storage.Tx[file], t targets.TargetID, version int, ws []works.WorkID, preview bool) (RemakeImpact, error)
}

// NoConversions is the Conversions of a library nothing converts.
type NoConversions struct{}

func (NoConversions) Cells(context.Context, storage.Reader[file], works.Among[works.WorkID]) (map[CellKey]ConversionCell, error) {
	return nil, nil
}

func (NoConversions) Holding(context.Context, storage.Reader[file]) (map[roots.ID]map[string]bool, error) {
	return nil, nil
}

func (NoConversions) Queue(context.Context, *storage.Tx[file], ...Ask) error { return nil }

func (NoConversions) Leaving(context.Context, *storage.Tx[file], File, ID) error { return nil }

func (NoConversions) Changed(context.Context, *storage.Tx[file], []PathChange) error { return nil }

func (NoConversions) Cost(context.Context, *storage.Tx[file], []MadePair) (MadeCost, error) {
	return MadeCost{}, nil
}

func (NoConversions) Remakes(context.Context, *storage.Tx[file], targets.TargetID, int, []works.WorkID, bool) (RemakeImpact, error) {
	return RemakeImpact{}, nil
}

// ConversionCell is a made-from cell's Conversion: Due once its download
// wait has passed. Node is where it is placed, nil while it is not, Code a
// failed one's fault and Class that fault's class, run or source.
type ConversionCell struct {
	ID                 storage.UUID
	Unfinished, Failed bool
	Due                bool
	Stage              string
	Node               *role.NodeID
	NodeName           string
	Code               string
	Class              string
}

func (v ConversionCell) Converting() bool { return v.Unfinished && !v.Failed && v.Due }

// Arrival is why a Work's Conversions are converged.
type Arrival string

const (
	ArrivalNew     Arrival = "new"
	ArrivalAdopted Arrival = "adopted"
	ArrivalProfile Arrival = "profile"
	ArrivalRemake  Arrival = "remake"
	ArrivalChanged Arrival = "changed"
	ArrivalMade    Arrival = "made"
)

// Ask is one Work to converge. Since is the Write's own time, from which a
// "Downloaded, or made from" row's wait counts.
type Ask struct {
	Work     works.WorkID
	Why      Arrival
	Since    time.Time
	Backfill bool
}

// Derived is a made file's provenance: the file it was made from, the
// Target whose Conversion settings made it, and their version. From is
// zero once that file is gone.
type Derived struct {
	From    ID
	Target  targets.TargetID
	Version int
}

// MadePair is a cell a Change makes made-from-wanted, with the file it
// would be made from.
type MadePair struct {
	Work   works.WorkID
	Source ID
	Target targets.TargetID
}

// MadeCost is what making a set of pairs would take: Unconvertible are the
// Works no Node can convert today.
type MadeCost struct {
	Works         int
	Seconds       int64
	Bytes         int64
	Nodes         []string
	Unconvertible []works.WorkID
	Rows          []MadeRow
}

// MadeRow is one Work of a backfill or a remake.
type MadeRow struct {
	Work    works.WorkID
	Files   int
	Seconds int64
	Bytes   int64
	Added   time.Time
}

// MadeImpact is the files a Change would make.
type MadeImpact struct {
	Files         int
	Works         int
	Seconds       int64
	Bytes         int64
	Nodes         []string
	Unconvertible []works.WorkID
	Matching      int
	Rows          []MadeRow
}

// RemakeImpact is the made files a Target's older settings made.
type RemakeImpact struct {
	Older int
	Rows  []MadeRow
}
