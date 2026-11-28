package conversions

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
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

// NodeSettings is node's Converting section, with its proofs, its running
// Conversions and what its holder last reported.
func (c *Conversions) NodeSettings(ctx context.Context, n role.NodeID) (NodeView, error) {
	var out NodeView
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		q := db.New(r)
		row, err := q.GetNodeSettings(ctx, n)
		switch {
		case errors.Is(err, storage.ErrNoRows):
			row = db.ConversionsNode{HoursStart: defaultHours.Start, HoursEnd: defaultHours.End, Processor: true}
		case err != nil:
			return fmt.Errorf("read the Node's Converting settings: %w", err)
		}
		out.Settings = NodeSettings{HoursOn: row.HoursOn, Hours: setting.Span{Start: int(row.HoursStart), End: int(row.HoursEnd)}, Processor: row.Processor}
		if row.AtOnce != nil {
			n := int(*row.AtOnce)
			out.Settings.AtOnce = &n
		}
		proofs, err := q.ListProofsOf(ctx, n)
		if err != nil {
			return fmt.Errorf("read the Node's proofs: %w", err)
		}
		for _, p := range proofs {
			out.Proofs = append(out.Proofs, proofViewOf(p))
		}
		held, err := q.CountHeld(ctx, n)
		out.Running = int(held.Started)
		return err
	})
	if err != nil {
		return NodeView{}, err
	}
	if f, ok := c.cfg.Control().Facts(n); ok {
		out.Override = f.GetOverride()
		if v := f.GetScratch(); v != nil {
			out.Scratch = &Volume{Free: v.GetFree(), Total: v.GetTotal()}
		}
	}
	return out, nil
}

// defaultHours are a Node's hours until its Admin sets them: 01:00 to
// 07:00.
var defaultHours = struct{ Start, End int64 }{60, 420}

// SetNodeSettings stores node's Converting section and tells its holder.
func (c *Conversions) SetNodeSettings(ctx context.Context, n role.NodeID, s NodeSettings) (NodeView, error) {
	p := db.UpsertNodeSettingsParams{NodeID: n, HoursOn: s.HoursOn, HoursStart: int64(s.Hours.Start), HoursEnd: int64(s.Hours.End), Processor: s.Processor}
	if s.AtOnce != nil {
		at := int64(*s.AtOnce)
		p.AtOnce = &at
	}
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).UpsertNodeSettings(ctx, p)
	})
	if err != nil {
		return NodeView{}, fmt.Errorf("store the Node's Converting settings: %w", err)
	}
	c.cfg.Control().Touch(n)
	c.cfg.Poke.Wake()
	return c.NodeSettings(ctx, n)
}

func proofViewOf(p db.ConversionsProof) ProofView {
	v := ProofView{
		Shape: shapeOf(p.Shape), Class: encode.Class(p.Class),
		Pipeline: encode.Pipeline{Path: encode.Path(p.Path), API: encode.API(p.Api)}, Device: p.Device, Pauses: p.Pauses,
		ProvenAt: timeOf(p.ProvenAt), FailedAt: timeOf(p.FailedAt), LostAt: timeOf(p.LostAt),
	}
	if p.Speed != nil {
		v.Speed = *p.Speed
	}
	if p.Error != nil {
		v.Error = *p.Error
	}
	return v
}

// shapeOf reads a stored Shape key, as Shape.String writes it.
func shapeOf(key string) encode.Shape {
	parts := strings.Split(key, ".")
	depth, _ := strconv.Atoi(parts[1])
	return encode.Shape{
		Codec: encode.Codec(parts[0]), Depth: encode.Depth(depth), Range: encode.Range(parts[2]), Curve: encode.Curve(parts[3]),
		KeepDV: len(parts) == 5,
	}
}
