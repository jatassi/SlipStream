package conversions

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/works"
)

// Band is a part of the Conversions tab.
type Band string

const (
	BandAttention  Band = "attention"
	BandConverting Band = "converting"
	BandNext       Band = "next"
)

// Cursor pages Up next by lane and position.
type Cursor string

type ListQuery struct {
	Band   Band
	Node   *role.NodeID
	Type   works.TypeID
	Q      string
	Work   *works.WorkID
	Cursor Cursor
}

type List struct {
	Counts     Counts
	Attention  []ListRow
	Converting []ListRow
	Next       []LaneRows
	Cursor     Cursor
}

// Counts are the tab's sentence numbers. ClearsAt is the latest finish.
type Counts struct {
	Converting, Paused, Waiting, Failed int
	ClearsAt                            *time.Time
}

type LaneRows struct {
	Lane Lane
	Rows []Entry
}

// Entry is one row of Up next: a Conversion or a folded group.
type Entry struct {
	Row   *ListRow
	Group *GroupRow
}

// ListRow is a Conversion with its forecast and live progress.
type ListRow struct {
	Row           Row
	NodeName      string
	Progress      *LiveRow
	Wait          *Wait
	Start, Finish *time.Time
	// FailedOn names the Nodes its counted attempts failed on.
	FailedOn []string
	Attempts []Attempt
}

// GroupID names a backfill's group or a series season's.
type GroupID string

type GroupKind string

const (
	GroupBackfill GroupKind = "backfill"
	GroupSeason   GroupKind = "season"
)

type GroupRow struct {
	ID            GroupID
	Kind          GroupKind
	Label         string
	Count         int
	Nodes         []string
	Seconds       int64
	Bytes         int64
	Start, Finish *time.Time
}

type Group struct {
	Rows   []ListRow
	Cursor Cursor
}

// List is the Conversions tab, forecast to the horizon at read time.
func (c *Conversions) List(context.Context, ListQuery) (List, error) { return List{}, errNotBuilt }

// Group is one folded group's rows.
func (c *Conversions) Group(context.Context, GroupID, Cursor) (Group, error) {
	return Group{}, errNotBuilt
}
