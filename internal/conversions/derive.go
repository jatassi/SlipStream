package conversions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/files"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

var stageOf = map[Stage]cv1.Stage{StageReady: cv1.Stage_STAGE_READY, StageRunning: cv1.Stage_STAGE_RUNNING, StagePaused: cv1.Stage_STAGE_PAUSED}

// derive is node's Converting intent: its ready, running and paused
// Conversions, with start set inside its hours, hold set while a Media
// Server there transcodes video and every run can pause, and every
// scratch name an unfinished Conversion there reads or writes.
func (c *Conversions) derive(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) (*cv1.Intent, error) {
	q := db.New(r)
	set, err := q.GetNodeSettings(ctx, node)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		set = db.ConversionsNode{Processor: true}
	case err != nil:
		return nil, fmt.Errorf("read the Node's Converting settings: %w", err)
	}
	in := &cv1.Intent{AtOnce: uint32(DefaultAtOnce)}
	if set.AtOnce != nil {
		in.AtOnce = uint32(*set.AtOnce) //nolint:gosec // conversions_node holds 1 to 16
	}
	start := !set.HoursOn || insideHours(setting.Span{Start: int(set.HoursStart), End: int(set.HoursEnd)}, c.cfg.Clock().In(c.cfg.Zone()))
	raws, err := q.ListOnNode(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("read the Node's Conversions: %w", err)
	}
	goals, err := goalsOf(ctx, r)
	if err != nil {
		return nil, err
	}
	pauses := true
	for _, raw := range raws {
		row, err := rowOf(raw)
		if err != nil {
			return nil, err
		}
		j, pausing, err := c.intentOf(ctx, r, in, row, goals, node)
		if err != nil {
			return nil, err
		}
		if j == nil {
			continue
		}
		j.Start = start
		pauses = pauses && pausing
		in.Jobs = append(in.Jobs, j)
	}
	transcoding, err := c.transcoding(ctx, r, node)
	if err != nil {
		return nil, err
	}
	in.Hold = transcoding && pauses
	return in, nil
}

// intentOf adds row's scratch names to in's keep, and is its job while it
// is ready, running or paused, with whether it can pause.
func (c *Conversions) intentOf(ctx context.Context, r storage.Reader[storage.Library], in *cv1.Intent, row Row, goals map[targets.TargetID]targets.Target, node role.NodeID) (*cv1.Job, bool, error) {
	if row.SourceFile == nil || row.Target == nil || goals[*row.Target].Conversion == nil {
		return nil, true, nil
	}
	src, err := c.cfg.Files.Located(ctx, r, *row.SourceFile)
	if err != nil {
		return nil, false, fmt.Errorf("read a source file: %w", err)
	}
	s := goals[*row.Target].Conversion.Settings
	remote := src.Node != node
	if remote && row.Stage != StageReturning {
		in.Keep = appendNew(in.Keep, scratchSource(src))
	}
	if remote && row.Stage != StageCopying {
		in.Keep = appendNew(in.Keep, scratchOutput(row.ID, row.Attempt, s.Container))
	}
	stage, ok := stageOf[row.Stage]
	if !ok {
		return nil, true, nil
	}
	j, proof, err := jobOf(ctx, db.New(r), row, src, s, remote)
	if err != nil {
		return nil, false, err
	}
	j.Stage = stage
	return j, row.Stage == StageReady || proof.Pauses, nil
}

func jobOf(ctx context.Context, q *db.Queries, row Row, src files.Located, s encode.Settings, remote bool) (*cv1.Job, db.ConversionsProof, error) {
	record, err := json.Marshal(src.Media)
	if err != nil {
		return nil, db.ConversionsProof{}, fmt.Errorf("encode the source's record: %w", err)
	}
	a, err := q.GetAttempt(ctx, db.GetAttemptParams{ConversionID: row.ID, N: int64(row.Attempt)})
	if err != nil {
		return nil, db.ConversionsProof{}, fmt.Errorf("read the Conversion's attempt: %w", err)
	}
	k := encode.Key{Shape: s.Shape(), Class: encode.ClassOf(src.Media)}
	proof, err := q.GetProof(ctx, db.GetProofParams{NodeID: *row.Node, Shape: k.Shape.String(), Class: string(k.Class)})
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return nil, db.ConversionsProof{}, fmt.Errorf("read the Conversion's proof: %w", err)
	}
	j := &cv1.Job{
		ConversionId: row.ID.Bytes(), Attempt: uint32(row.Attempt), //nolint:gosec // attempts count up from 1
		Settings: &cv1.Settings{
			Codec: string(s.Codec), MaxHeight: uint32(s.MaxHeight), //nolint:gosec // encode.Parse bounds every number Kbps: uint32(s.Kbps), Depth: uint32(s.Depth), Range: string(s.Range),
			KeepDv: s.KeepDV, Curve: string(s.Curve), Audio: string(s.Audio), DropCommentary: s.DropCommentary, Container: string(s.Container),
		},
		Source:   &cv1.Source{Record: record},
		Pipeline: &cv1.Pipeline{Path: a.Path, Api: a.Api},
		Device:   proof.Device,
	}
	if remote {
		j.Input = &cv1.Job_Scratch{Scratch: scratchSource(src)}
		j.Output = &cv1.Job_ScratchOut{ScratchOut: scratchOutput(row.ID, row.Attempt, s.Container)}
	} else {
		j.Input = &cv1.Job_Library{Library: &cv1.LibraryPath{RootId: src.Root.Bytes(), Rel: src.Rel.String()}}
		j.Output = &cv1.Job_Holding{Holding: &cv1.LibraryPath{RootId: src.Root.Bytes(), Rel: holdingRel(row.ID, row.Attempt, s.Container).String()}}
	}
	return j, proof, nil
}

func appendNew(names []string, name string) []string {
	if slices.Contains(names, name) {
		return names
	}
	return append(names, name)
}

// insideHours is whether local's minute of the day is inside span,
// which may run past midnight.
func insideHours(span setting.Span, local time.Time) bool {
	m := local.Hour()*60 + local.Minute()
	if span.Start < span.End {
		return m >= span.Start && m < span.End
	}
	return m >= span.Start || m < span.End
}

// pausedBy names the Media Servers whose video transcodes paused the
// run, at most eight.
func pausedBy(ctx context.Context, m *move) error {
	ts, err := m.c.cfg.TranscodingOn(ctx, m.tx, *m.row.Node)
	if err != nil {
		return fmt.Errorf("read the Media Servers on the Node: %w", err)
	}
	m.row.PausedBy = nil
	for _, t := range ts {
		if t.Known && t.Now && len(m.row.PausedBy) < maxDevices {
			m.row.PausedBy = append(m.row.PausedBy, capName(t.Name))
		}
	}
	return nil
}
