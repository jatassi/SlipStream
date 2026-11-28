package conversions

import (
	"context"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/conversions/internal/plan"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func (k CauseKey) String() string {
	out := string(k.Why) + "|" + string(k.Class) + "|" + k.Shape
	if k.Target != nil {
		out += "|" + k.Target.String()
	}
	return out
}

// opened is a cause a step's jobs fold into, with its jobs.
type opened struct {
	Cause
	jobs []ID
}

func foldCauses(s *snap, cs []plan.Cause) map[string]*opened {
	out := map[string]*opened{}
	for _, pc := range cs {
		row := s.rows[ID{UUID: pc.Job}]
		key := CauseKey{Why: pc.Why}
		if j := s.Jobs[s.at[row.ID]]; !j.Source.IsZero() {
			key.Class, key.Shape = j.Key.Class, j.Key.Shape.String()
		}
		if pc.Why == plan.RunsElsewhere {
			key.Target = row.Target
		}
		o, ok := out[key.String()]
		if !ok {
			o = &opened{Cause: Cause{Key: key, Node: pc.Node}}
			out[key.String()] = o
		}
		o.Count++
		o.jobs = append(o.jobs, row.ID)
	}
	return out
}

// causes diffs the step's causes against the open ones, in tx: a new one
// opens and records No Node can convert once, one that stays keeps its
// count, and one gone closes.
func (c *Conversions) causes(ctx context.Context, tx *storage.Tx[storage.Library], s *snap, cs []plan.Cause) error {
	q := db.New(tx)
	now := foldCauses(s, cs)
	was, err := q.ListCauses(ctx)
	if err != nil {
		return fmt.Errorf("read the open causes: %w", err)
	}
	for _, w := range was {
		o, ok := now[w.Key]
		switch {
		case !ok:
			err = q.DeleteCause(ctx, w.Key)
		case int64(o.Count) != w.Count:
			err = q.SetCauseCount(ctx, db.SetCauseCountParams{Count: int64(o.Count), Key: w.Key})
		}
		if err != nil {
			return fmt.Errorf("store a cause: %w", err)
		}
		delete(now, w.Key)
	}
	keys := make([]string, 0, len(now))
	for k := range now {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	at := storage.FromTime(s.Now)
	for _, k := range keys {
		o := now[k]
		class, shape := string(o.Key.Class), o.Key.Shape
		row := db.InsertCauseParams{Key: k, Why: string(o.Key.Why), Count: int64(o.Count), OpenedAt: at, NotifiedAt: at, NodeID: value(o.Node), TargetID: value(o.Key.Target)}
		if class != "" {
			row.Class, row.Shape = &class, &shape
		}
		if err := q.InsertCause(ctx, row); err != nil {
			return fmt.Errorf("open a cause: %w", err)
		}
		if err := c.recordCause(ctx, tx, s, o); err != nil {
			return err
		}
	}
	return nil
}

func (c *Conversions) recordCause(ctx context.Context, tx *storage.Tx[storage.Library], s *snap, o *opened) error {
	var ws []works.WorkID
	var kind works.TypeID
	for _, id := range o.jobs {
		r := s.rows[id]
		kind = r.Type
		if !slices.Contains(ws, r.Work) {
			ws = append(ws, r.Work)
		}
	}
	return NoNodeCanConvert.Record(ctx, tx, c.cfg.History, NoNodeFacts{
		Type: kind, Noun: works.Noun{One: "Work", Many: "Works"}, Works: len(ws), Words: causeWords(o.Cause, s.names, s.goals),
	})
}

// causeWords is a cause's sentence: why no Node can convert and what to
// do.
func causeWords(c Cause, names map[role.NodeID]string, goals map[targets.TargetID]targets.Target) string {
	node := ""
	if c.Node != nil {
		node = names[*c.Node]
	}
	switch c.Key.Why {
	case plan.NoHolder:
		return "No Node has the Converting Role. Give one the Role on the Nodes page."
	case plan.NoneOnline:
		return "No Node with the Converting Role is online. They start when one is back."
	case plan.RunsElsewhere:
		t := goals[value(c.Key.Target)].Name
		return fmt.Sprintf("%s runs only on %s, which is offline. They start when %s is back, or change Run on for %s.", t, node, node, t)
	case plan.ProcessorOff:
		return fmt.Sprintf("Only the processor can convert them, and it's turned off on %s. Turn it on in %s's Converting section.", node, node)
	case plan.TestFailed:
		return fmt.Sprintf("%s's test run failed. Rerun the test run on %s.", node, node)
	case plan.ProofLost:
		return fmt.Sprintf("A Conversion failed on %s, so it needs a new test run. Rerun the test run on %s.", node, node)
	case plan.Lapsed:
		return fmt.Sprintf("%s is offline. They start when %s is back.", node, node)
	case plan.Fits, plan.NoRole, plan.PinnedElsewhere, plan.FailedHere, plan.Untested, plan.Retesting:
	}
	return "No Node with the Converting Role can convert them."
}

// Causes is the open causes of each of ts: those naming it, and those of
// its settings' shape.
func (c *Conversions) Causes(ctx context.Context, ts []targets.TargetID) (map[targets.TargetID][]Cause, error) {
	out := map[targets.TargetID][]Cause{}
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := db.New(r).ListCauses(ctx)
		if err != nil {
			return fmt.Errorf("read the open causes: %w", err)
		}
		goals, err := goalsOf(ctx, r)
		if err != nil {
			return err
		}
		for _, row := range rows {
			cause := causeOf(row)
			for _, t := range ts {
				g, ok := goals[t]
				switch {
				case !ok || g.Conversion == nil:
				case cause.Key.Target != nil && *cause.Key.Target != t:
				case cause.Key.Target != nil, cause.Key.Shape == "", cause.Key.Shape == g.Conversion.Settings.Shape().String():
					out[t] = append(out[t], cause)
				}
			}
		}
		return nil
	})
	return out, err
}

func causeOf(row db.ConversionsCause) Cause {
	c := Cause{Key: CauseKey{Why: Why(row.Why), Target: some(row.TargetID)}, Count: int(row.Count), Node: some(row.NodeID)}
	if row.Class != nil {
		c.Key.Class, c.Key.Shape = encode.Class(*row.Class), *row.Shape
	}
	return c
}
