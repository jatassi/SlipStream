package conversions

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/conversions/internal/plan"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
)

// snap is plan.Run's snapshot with what it was read from: the rows by id,
// their source files and Targets, and the Nodes' names.
type snap struct {
	plan.Snapshot
	rows  map[ID]Row
	at    map[ID]int
	srcs  map[files.ID]files.Located
	goals map[targets.TargetID]targets.Target
	names map[role.NodeID]string
}

// snapshot reads the queue, the Nodes, their proofs, rates, room, hours
// and transcodes in one read.
func (c *Conversions) snapshot(ctx context.Context, r storage.Reader[storage.Library], ros roster) (*snap, error) {
	q := db.New(r)
	s := &snap{
		Snapshot: plan.Snapshot{
			Now: c.cfg.Clock(), AssumedRate: float64(c.cfg.Timings.AssumedRate), TestsAtOnce: c.cfg.Timings.TestsAtOnce,
			Tests: c.rec.inFlight(),
		},
		rows: map[ID]Row{}, at: map[ID]int{}, srcs: map[files.ID]files.Located{}, names: map[role.NodeID]string{},
	}
	raws, err := q.ListUnfinished(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the unfinished Conversions: %w", err)
	}
	var rows []Row
	for _, raw := range raws {
		row, err := rowOf(raw)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
		s.rows[row.ID] = row
		if row.SourceFile == nil {
			continue
		}
		if _, ok := s.srcs[*row.SourceFile]; ok {
			continue
		}
		l, err := c.cfg.Files.Located(ctx, r, *row.SourceFile)
		switch {
		case errors.Is(err, files.ErrNoFile), errors.Is(err, storage.ErrNoRows):
		case err != nil:
			return nil, fmt.Errorf("read a source file: %w", err)
		default:
			s.srcs[*row.SourceFile] = l
		}
	}
	if s.goals, err = goalsOf(ctx, r); err != nil {
		return nil, err
	}
	if s.Rates, err = transfers.Rates(ctx, r); err != nil {
		return nil, fmt.Errorf("read the Transfer rates: %w", err)
	}
	named, err := nodes.Names(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("read the Nodes' names: %w", err)
	}
	for _, n := range named {
		s.names[n.ID] = n.Name
	}
	rooms := map[roots.ID]int64{}
	for _, row := range rows {
		j, err := c.jobOf(ctx, r, s, row, rooms)
		if err != nil {
			return nil, err
		}
		s.at[row.ID] = len(s.Jobs)
		s.Jobs = append(s.Jobs, j)
	}
	if err := c.nodesOf(ctx, r, q, s, ros); err != nil {
		return nil, err
	}
	return s, nil
}

// made is the source and Target settings row converts, false once either
// is gone.
func (s *snap) made(row Row) (files.Located, encode.Settings, bool) {
	if row.SourceFile == nil || row.Target == nil {
		return files.Located{}, encode.Settings{}, false
	}
	src, ok := s.srcs[*row.SourceFile]
	t, has := s.goals[*row.Target]
	if !ok || !has || t.Conversion == nil {
		return files.Located{}, encode.Settings{}, false
	}
	return src, t.Conversion.Settings, true
}

func (c *Conversions) jobOf(ctx context.Context, r storage.Reader[storage.Library], s *snap, row Row, rooms map[roots.ID]int64) (plan.Job, error) {
	j := plan.Job{
		ID: row.ID.UUID, Stage: string(row.Stage), Attempt: row.Attempt, Node: row.Node, Pin: row.Pin,
		Exclude: map[role.NodeID]bool{}, NotBefore: row.NotBefore,
	}
	for _, n := range row.Exclude {
		j.Exclude[n] = true
	}
	src, settings, ok := s.made(row)
	if !ok {
		return j, nil
	}
	t := s.goals[*row.Target]
	w, h := encode.Out(src.Media, settings)
	j.Target, j.RunOn = t.ID.UUID, t.Conversion.RunOn
	j.Key = encode.Key{Shape: settings.Shape(), Class: encode.ClassOf(src.Media)}
	j.Source, j.SourceNode, j.SourceBytes = src.ID.UUID, src.Node, src.Size
	j.OutBytes, j.Length, j.OutPixels = encode.Bytes(src.Media, settings), src.Media.Duration, w*h
	j.Root, j.RootNode = src.Root.UUID, src.Node
	room, ok := rooms[src.Root]
	if !ok {
		free, known, err := c.cfg.RootRoom(ctx, r, src.Root)
		if err != nil {
			return plan.Job{}, fmt.Errorf("read the room in a library folder: %w", err)
		}
		if known {
			room = free
		}
		rooms[src.Root] = room
	}
	j.RootRoom = room
	return j, nil
}

// nodesOf is every Node that serves Converting or holds a row, with its
// settings, proofs under its signature, scratch room and running rows.
func (c *Conversions) nodesOf(ctx context.Context, r storage.Reader[storage.Library], q *db.Queries, s *snap, ros roster) error {
	ids := slices.Clone(ros.holders.Serving())
	for _, j := range s.Jobs {
		if j.Node != nil && !slices.Contains(ids, *j.Node) {
			ids = append(ids, *j.Node)
		}
	}
	settings, err := q.ListNodeSettings(ctx)
	if err != nil {
		return fmt.Errorf("read the Nodes' Converting settings: %w", err)
	}
	proofs, err := q.ListProofs(ctx)
	if err != nil {
		return fmt.Errorf("read the proofs: %w", err)
	}
	for _, id := range ids {
		n := c.nodeOf(s, ros, id, settings)
		f, reported := ros.facts(id)
		facts := factsOf(f, n.AtOnce, s.lengthsOn(id))
		n.Proofs = proofsOf(proofs, id, string(facts.Signature), reported)
		if facts.Scratch != nil {
			n.ScratchFree = facts.Scratch.Free - s.reservedOn(id)
		}
		n.Running = s.running(n, facts.Progress)
		if n.Holds {
			if n.Hold, err = c.transcoding(ctx, r, id); err != nil {
				return err
			}
		}
		s.Nodes = append(s.Nodes, n)
	}
	return nil
}

// nodeOf is id as the Board stands it, with its Converting settings.
func (c *Conversions) nodeOf(s *snap, ros roster, id role.NodeID, settings []db.ConversionsNode) plan.Node {
	n := plan.Node{
		ID: id, Name: s.names[id], Holds: ros.holders.Holds(id), May: ros.holders.May(id),
		Lapsed: ros.holders.Lapsed(id) || ros.holders.WrittenOff(id), AtOnce: DefaultAtOnce, Processor: true, Zone: c.cfg.Zone(),
	}
	if a, away := ros.holders.Away(id); away && !a.Lapsed {
		n.Away = true
	}
	i := slices.IndexFunc(settings, func(r db.ConversionsNode) bool { return r.NodeID == id })
	if i < 0 {
		return n
	}
	set := settings[i]
	if set.AtOnce != nil {
		n.AtOnce = int(*set.AtOnce)
	}
	if set.HoursOn {
		n.Hours = &setting.Span{Start: int(set.HoursStart), End: int(set.HoursEnd)}
	}
	n.Processor = set.Processor
	return n
}

// running is the time each of n's runs has left, from its progress and
// its proof's speed.
func (s *snap) running(n plan.Node, progress map[ID]RunProgress) []plan.Running {
	var out []plan.Running
	for id, p := range progress {
		i, ok := s.at[id]
		if !ok || s.Jobs[i].Attempt != p.Attempt {
			continue
		}
		j := s.Jobs[i]
		speed := n.Proofs[j.Key].Speed
		if speed <= 0 || j.Length <= 0 {
			continue
		}
		whole := time.Duration(float64(j.Length) * float64(j.OutPixels) / (speed * 1920 * 1080))
		out = append(out, plan.Running{Job: j.ID, Left: max(0, whole-time.Duration(float64(whole)*float64(p.Out)/float64(j.Length)))})
	}
	return out
}

// transcoding is whether a Media Server on node that pauses its
// Conversions transcodes video now.
func (c *Conversions) transcoding(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) (bool, error) {
	ts, err := c.cfg.TranscodingOn(ctx, r, node)
	if err != nil {
		return false, fmt.Errorf("read the Media Servers on a Node: %w", err)
	}
	return slices.ContainsFunc(ts, func(t mediaservers.Transcoding) bool { return t.Known && t.Now }), nil
}

// proofsOf is node's proofs under sig, or every one of its proofs while
// it has reported no facts. A Node whose signature isn't one has none.
func proofsOf(all []db.ConversionsProof, node role.NodeID, sig string, reported bool) map[encode.Key]plan.Proof {
	out := map[encode.Key]plan.Proof{}
	for _, p := range all {
		if p.NodeID != node || (reported && p.Signature != sig) {
			continue
		}
		pr := plan.Proof{
			Pipeline: encode.Pipeline{Path: encode.Path(p.Path), API: encode.API(p.Api)}, Pauses: p.Pauses,
			Failed: p.ProvenAt.IsZero(), Lost: !p.LostAt.IsZero(),
		}
		if p.Speed != nil {
			pr.Speed = *p.Speed
		}
		out[encode.Key{Shape: shapeOf(p.Shape), Class: encode.Class(p.Class)}] = pr
	}
	return out
}

// lengthsOn is the source length of each running or paused row on node.
func (s *snap) lengthsOn(node role.NodeID) map[ID]time.Duration {
	out := map[ID]time.Duration{}
	for id, row := range s.rows {
		if row.Node != nil && *row.Node == node && (row.Stage == StageRunning || row.Stage == StagePaused) {
			out[id] = s.Jobs[s.at[id]].Length
		}
	}
	return out
}

// reservedOn is the scratch bytes node's placed rows hold: each copied
// source once, and each output written to scratch.
func (s *snap) reservedOn(node role.NodeID) int64 {
	var n int64
	sources := map[storage.UUID]bool{}
	for _, j := range s.Jobs {
		if j.Node == nil || *j.Node != node || !slices.Contains(placedStages, Stage(j.Stage)) || j.SourceNode == node {
			continue
		}
		if !sources[j.Source] && Stage(j.Stage) != StageReturning {
			sources[j.Source] = true
			n += j.SourceBytes
		}
		n += j.OutBytes
	}
	return n
}
