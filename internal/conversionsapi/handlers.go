package conversionsapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func (h handlers) converts() *conversions.Conversions {
	c, ok := h.owner.Conversions()
	if !ok {
		panic("conversionsapi: an operation ran on a Node that is no Controller")
	}
	return c
}

var codes = []struct {
	err  error
	code *api.Code
}{
	{conversions.ErrNoConversion, conversionNotFound},
	{conversions.ErrFinished, conversionFinished},
	{conversions.ErrNotWaiting, conversionNotWaiting},
	{conversions.ErrNotFailed, conversionNotFailed},
	{conversions.ErrPlacing, conversionPlacing},
}

func answer(err error) error {
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

func conversionID(s string) (conversions.ID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return conversions.ID{}, conversionNotFound
	}
	return conversions.ID{UUID: u}, nil
}

func (h handlers) listConversions(ctx context.Context, _ identity.Admin, in *listInput) (*listOutput, error) {
	q := conversions.ListQuery{Band: conversions.Band(in.Band), Type: works.TypeID(in.Type), Q: in.Q, Cursor: conversions.Cursor(in.Cursor)}
	if in.Node != "" {
		u, err := storage.ParseUUID(in.Node)
		if err != nil {
			return nil, api.Invalid.Fields(api.FieldError{Location: "query.node", Message: "No Node has this id."})
		}
		q.Node = &role.NodeID{UUID: u}
	}
	if in.Work != "" {
		u, err := storage.ParseUUID(in.Work)
		if err != nil {
			return nil, api.Invalid.Fields(api.FieldError{Location: "query.work", Message: "No Work has this id."})
		}
		q.Work = &works.WorkID{UUID: u}
	}
	l, err := h.converts().List(ctx, q)
	if err != nil {
		return nil, err
	}
	out := conversionList{
		Counts: counts{
			Converting: l.Counts.Converting, Paused: l.Counts.Paused, Waiting: l.Counts.Waiting, Failed: l.Counts.Failed,
			ClearsAt: l.Counts.ClearsAt,
		},
		Attention: rowsOf(l.Attention), Converting: rowsOf(l.Converting), Next: []laneRows{}, Cursor: cursorOf(l.Cursor),
	}
	for _, lane := range l.Next {
		lr := laneRows{Lane: string(lane.Lane), Rows: []api.OneOf[upNext]{}}
		for _, e := range lane.Rows {
			if e.Group != nil {
				lr.Rows = append(lr.Rows, api.NewOneOf[upNext](groupOf(*e.Group)))
			} else {
				lr.Rows = append(lr.Rows, api.NewOneOf[upNext](rowOf(*e.Row)))
			}
		}
		out.Next = append(out.Next, lr)
	}
	return &listOutput{Body: out}, nil
}

func (h handlers) getConversionGroup(ctx context.Context, _ identity.Admin, in *groupInput) (*groupOutput, error) {
	g, err := h.converts().Group(ctx, conversions.GroupID(in.GroupID), conversions.Cursor(in.Cursor))
	if err != nil {
		return nil, answer(err)
	}
	return &groupOutput{Body: groupPage{Rows: rowsOf(g.Rows), Cursor: cursorOf(g.Cursor)}}, nil
}

func (h handlers) moveConversionUp(ctx context.Context, _ identity.Admin, in *conversionInput) (*rowOutput, error) {
	return h.act(in.ID, func(id conversions.ID) (conversions.ListRow, error) { return h.converts().MoveUp(ctx, id) })
}

func (h handlers) moveConversion(ctx context.Context, _ identity.Admin, in *moveInput) (*rowOutput, error) {
	var after *conversions.ID
	if v := in.Body.After.Value(); v != nil {
		a, err := conversionID(*v)
		if err != nil {
			return nil, api.Invalid.On("after", "No Conversion has this id.")
		}
		after = &a
	}
	return h.act(in.ID, func(id conversions.ID) (conversions.ListRow, error) {
		return h.converts().MoveAfter(ctx, id, after)
	})
}

func (h handlers) retryConversion(ctx context.Context, _ identity.Admin, in *conversionInput) (*rowOutput, error) {
	return h.act(in.ID, func(id conversions.ID) (conversions.ListRow, error) { return h.converts().Retry(ctx, id) })
}

func (h handlers) cancelConversion(ctx context.Context, _ identity.Admin, in *conversionInput) (*rowOutput, error) {
	return h.act(in.ID, func(id conversions.ID) (conversions.ListRow, error) { return h.converts().Cancel(ctx, id) })
}

func (h handlers) act(raw string, do func(conversions.ID) (conversions.ListRow, error)) (*rowOutput, error) {
	id, err := conversionID(raw)
	if err != nil {
		return nil, err
	}
	r, err := do(id)
	if err != nil {
		return nil, answer(err)
	}
	return &rowOutput{Body: rowOf(r)}, nil
}

func (h handlers) cancelConversionGroup(ctx context.Context, _ identity.Admin, in *groupCancelInput) (*canceledOutput, error) {
	n, err := h.converts().CancelGroup(ctx, conversions.GroupID(in.GroupID))
	if err != nil {
		return nil, answer(err)
	}
	out := &canceledOutput{}
	out.Body.Canceled = n
	return out, nil
}

// node is the Node the path names, 404 when none has its id.
func (h handlers) node(ctx context.Context, raw string) (role.NodeID, error) {
	u, err := storage.ParseUUID(raw)
	if err != nil {
		return role.NodeID{}, nodeNotFound
	}
	roster, ok := h.owner.Nodes()
	if !ok {
		panic("conversionsapi: an operation ran on a Node that is no Controller")
	}
	id := role.NodeID{UUID: u}
	names, err := roster.Names(ctx)
	if err != nil {
		return role.NodeID{}, err
	}
	if n, ok := names[id]; !ok || n.Removed {
		return role.NodeID{}, nodeNotFound
	}
	return id, nil
}

func (h handlers) getNodeConverting(ctx context.Context, _ identity.Admin, in *nodeInput) (*nodeOutput, error) {
	id, err := h.node(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	v, err := h.converts().NodeSettings(ctx, id)
	if err != nil {
		return nil, err
	}
	return &nodeOutput{Body: nodeOf(v)}, nil
}

func (h handlers) updateNodeConverting(ctx context.Context, _ identity.Admin, in *updateNodeInput) (*nodeOutput, error) {
	id, err := h.node(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	span, err := spanOf(in.Body.Hours.Window)
	if err != nil {
		return nil, err
	}
	v, err := h.converts().SetNodeSettings(ctx, id, conversions.NodeSettings{
		AtOnce: in.Body.AtOnce, HoursOn: in.Body.Hours.On, Hours: span, Processor: in.Body.Processor,
	})
	if err != nil {
		return nil, err
	}
	return &nodeOutput{Body: nodeOf(v)}, nil
}

func (h handlers) rerunNodeTests(ctx context.Context, _ identity.Admin, in *nodeInput) (*struct{}, error) {
	id, err := h.node(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	return nil, h.converts().Rerun(ctx, id)
}

func spanOf(t setting.Times) (setting.Span, error) {
	start, okStart := minutesOf(t.Start)
	end, okEnd := minutesOf(t.End)
	switch {
	case !okStart || !okEnd:
		return setting.Span{}, api.Invalid.On("hours.window", "Times are 00:00 to 23:59.")
	case start == end:
		return setting.Span{}, api.Invalid.On("hours.window", "The start and end are the same time. Choose a different end. A window can cross midnight, such as 22:00 to 04:00.")
	}
	return setting.Span{Start: start, End: end}, nil
}

func minutesOf(s string) (int, bool) {
	t, err := time.Parse("15:04", s)
	if err != nil || len(s) != len("15:04") {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

func clockOf(minutes int) string { return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60) }

func nodeOf(v conversions.NodeView) nodeConverting {
	s := v.Settings
	out := nodeConverting{
		AtOnce: conversions.DefaultAtOnce, AtOnceDefault: conversions.DefaultAtOnce,
		Hours:     hours{On: s.HoursOn, Window: setting.Times{Start: clockOf(s.Hours.Start), End: clockOf(s.Hours.End)}},
		Processor: s.Processor, Proofs: []proof{}, Override: v.Override, Running: v.Running,
	}
	if s.AtOnce != nil {
		out.AtOnce = *s.AtOnce
	}
	if v.Scratch != nil {
		out.Scratch = api.Some(volume{Free: v.Scratch.Free, Total: v.Scratch.Total})
	}
	for _, p := range v.Proofs {
		out.Proofs = append(out.Proofs, proofOf(p))
		if p.ProvenAt == nil || p.LostAt != nil {
			continue
		}
		switch {
		case p.Pipeline.Path == encode.Processor:
			out.Sentence.Processor = out.Sentence.Processor || s.Processor
		case out.Sentence.API == nil:
			out.Sentence.API = new(string(p.Pipeline.API))
		}
	}
	out.Sentence.None = out.Sentence.API == nil && !out.Sentence.Processor
	return out
}

func proofOf(p conversions.ProofView) proof {
	out := proof{
		Shape: shape{
			Codec: string(p.Shape.Codec), Depth: int(p.Shape.Depth), Range: string(p.Shape.Range), KeepDV: p.Shape.KeepDV,
			Curve: string(p.Shape.Curve),
		},
		Class: string(p.Class), Path: string(p.Pipeline.Path), Device: p.Device, Speed: p.Speed, Pauses: p.Pauses,
		ProvenAt: p.ProvenAt, FailedAt: p.FailedAt, LostAt: p.LostAt, Error: p.Error,
	}
	if p.Pipeline.API != "" {
		out.API = new(string(p.Pipeline.API))
	}
	return out
}

func cursorOf(c conversions.Cursor) *string {
	if c == "" {
		return nil
	}
	return new(string(c))
}

func rowsOf(rs []conversions.ListRow) []ConversionRow {
	out := make([]ConversionRow, len(rs))
	for i, r := range rs {
		out[i] = rowOf(r)
	}
	return out
}

func rowOf(lr conversions.ListRow) ConversionRow {
	r := lr.Row
	out := ConversionRow{
		Kind: "conversion", ID: r.ID.String(), Stage: string(r.Stage), Lane: string(r.Lane),
		Work:   work{ID: r.Work.String(), Title: r.Snapshot.Work, Type: string(r.Type)},
		Items:  []item{},
		Target: target{Name: r.Snapshot.Target},
		Source: source{Name: r.Snapshot.SourceName, Target: lr.Source.Name, Edition: string(r.Edition)},
		Start:  lr.Start, Finish: lr.Finish, Attempts: []attempt{},
	}
	for _, it := range lr.Items {
		out.Items = append(out.Items, item{ID: it.ID.String(), Label: it.Label})
	}
	if r.Target != nil {
		out.Target.ID = new(r.Target.String())
	}
	if r.SourceFile != nil {
		out.Source.FileID = new(r.SourceFile.String())
	}
	if lr.Source.Target != nil {
		out.Source.TargetID = new(lr.Source.Target.String())
	}
	if r.Node != nil {
		out.Node = api.Some(named{ID: r.Node.String(), Name: lr.NodeName})
	}
	if p := lr.Progress; p != nil {
		out.Progress = api.Some(progress{Percent: p.Percent, Speed: p.Speed, SecondsLeft: p.SecondsLeft})
	}
	if r.Stage == conversions.StagePaused {
		out.PausedBy = append([]string{}, r.PausedBy...)
	}
	if w := lr.Wait; w != nil {
		out.Wait = api.Some(waitOf(lr, *w))
	}
	if f := r.Failure; f != nil {
		out.Failure = api.Some(failure{Code: string(f.Code), Class: string(f.Class), Nodes: append([]string{}, lr.FailedOn...), Detail: f.Detail})
	}
	for _, a := range lr.Attempts {
		out.Attempts = append(out.Attempts, attempt{
			N: a.N, Node: a.NodeName, Outcome: string(a.Outcome), Code: string(a.Code), StartedAt: a.Started, EndedAt: a.Ended,
		})
	}
	return out
}

func waitOf(lr conversions.ListRow, w conversions.Wait) wait {
	out := wait{Code: string(w.Code), Until: w.Until, Cause: string(w.Cause)}
	if w.Node != nil {
		out.Node = &named{ID: w.Node.String(), Name: lr.WaitNode}
	}
	switch w.Code {
	case "room":
		out.Free, out.Need = &w.Free, &w.Need
	case "in_line":
		if w.Place > 0 {
			out.Place = &w.Place
		}
	case "transcode":
		out.Servers = append([]string{}, lr.Servers...)
	case "download", "no_source":
		out.SourceTarget = lr.Source.Name
	case "hours", "copying", "away", "testing", "no_node":
	}
	return out
}

func groupOf(g conversions.GroupRow) conversionGroup {
	return conversionGroup{
		Kind: "group", ID: string(g.ID), GroupKind: string(g.Kind), Label: g.Label, Count: g.Count, Nodes: append([]string{}, g.Nodes...),
		Seconds: g.Seconds, Bytes: g.Bytes, Start: g.Start, Finish: g.Finish,
	}
}
