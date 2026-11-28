package conversions

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
)

// NodeSettings is a Node's Converting section. A nil AtOnce is
// DefaultAtOnce.
type NodeSettings struct {
	AtOnce    *int
	HoursOn   bool
	Hours     setting.Span
	Processor bool
}

// NodeView is a Node's Converting section with what its holder reports.
type NodeView struct {
	Settings NodeSettings
	Proofs   []ProofView
	// Scratch is nil while the volume is unknown.
	Scratch  *Volume
	Override bool
	Running  int
}

type ProofView struct {
	Shape                      encode.Shape
	Class                      encode.Class
	Pipeline                   encode.Pipeline
	Device                     string
	Speed                      float64
	Pauses                     bool
	ProvenAt, FailedAt, LostAt *time.Time
	Error                      string
}

type Volume struct{ Free, Total int64 }

func (c *Conversions) NodeSettings(context.Context, role.NodeID) (NodeView, error) {
	return NodeView{}, errNotBuilt
}

func (c *Conversions) SetNodeSettings(context.Context, role.NodeID, NodeSettings) (NodeView, error) {
	return NodeView{}, errNotBuilt
}
