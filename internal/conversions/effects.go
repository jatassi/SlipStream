package conversions

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/conversions/internal/order"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// placedTo copies the source first, or waits for the copy another
// Conversion of it on the Node is making.
func placedTo(ctx context.Context, m *move) (Stage, error) {
	if m.row.NotBefore.After(m.now) {
		return "", nil
	}
	if m.ev.Copy {
		return StageCopying, nil
	}
	copying, err := m.q.ListCopying(ctx, db.ListCopyingParams{NodeID: m.ev.Node, SourceFileID: *m.row.SourceFile})
	if err != nil {
		return "", fmt.Errorf("read the Conversions copying the source: %w", err)
	}
	if len(copying) > 0 {
		return StageCopying, nil
	}
	return StageReady, nil
}

// place opens the next attempt on the event's Node with the proof's
// pipeline, in the source's root.
func place(ctx context.Context, m *move) error {
	src, err := m.source(ctx)
	if err != nil {
		return err
	}
	m.row.Attempt++
	node, root := m.ev.Node, src.Root
	m.row.Node, m.row.Root, m.row.Pin = &node, &root, nil
	return m.openAttempt(ctx, node, m.ev.Proof.Pipeline)
}

// wantSource copies the source to the Node's scratch folder, sharing the
// one record of that Node and source.
func wantSource(ctx context.Context, m *move) error {
	if m.row.Stage != StageCopying {
		return nil
	}
	src, err := m.source(ctx)
	if err != nil {
		return err
	}
	_, err = m.c.cfg.Transfers.Want(ctx, m.tx, transfers.Request{
		Purpose: transfers.Conversion, Job: sourceJob(*m.row.Node, src.ID), Source: src.Node,
		Destination: transfers.Scratch{Node: *m.row.Node, Name: scratchSource(src)}, Priority: transfers.Low,
		Files: []transfers.FileRequest{{Name: src.Name, Source: source.Library{Root: [16]byte(src.Root.Bytes()), Rel: src.Rel.String()}, Size: src.Size}},
	})
	if err != nil {
		return fmt.Errorf("copy the source: %w", err)
	}
	return nil
}

// unfitTo fails a queued Conversion whose source its settings can't take,
// found at placement.
func unfitTo(_ context.Context, m *move) (Stage, error) {
	if faults[m.code].Class != SourceClass {
		return "", nil
	}
	return StageFailed, nil
}

func unpinTo(_ context.Context, m *move) (Stage, error) {
	if m.row.Pin == nil {
		return "", nil
	}
	return StageQueued, nil
}

// unpinned moves a pinned retry on to another Node.
func unpinned(_ context.Context, m *move) error {
	if !nodeIn(m.row.Exclude, m.ev.Node) {
		m.row.Exclude = append(slices.Clone(m.row.Exclude), m.ev.Node)
	}
	m.row.Pin = nil
	return nil
}

func started(ctx context.Context, m *move) error {
	t, err := m.target(ctx)
	if err != nil {
		return err
	}
	if m.row.StartedAt == nil {
		at := m.now
		m.row.StartedAt = &at
	}
	m.row.Version = t.Version
	return nil
}

func resumed(_ context.Context, m *move) error {
	m.row.PausedBy = nil
	return nil
}

// exitedTo checks the output where ffmpeg wrote it when its root is on
// the converting Node, and returns it to the root first when it isn't.
func exitedTo(ctx context.Context, m *move) (Stage, error) {
	src, err := m.source(ctx)
	if err != nil {
		return "", err
	}
	if src.Node == *m.row.Node {
		return StageChecking, nil
	}
	return StageReturning, nil
}

// returnOutput sends a returning output from the scratch folder to its
// root. The run read its source copy for the last time.
func returnOutput(ctx context.Context, m *move) error {
	if m.row.Stage != StageReturning {
		return nil
	}
	if err := m.doneReading(ctx, *m.row.Node); err != nil {
		return err
	}
	_, err := m.outputTransfer(ctx, m.ev.Bytes)
	return err
}

func (m *move) outputTransfer(ctx context.Context, size int64) (transfers.ID, error) {
	t, err := m.target(ctx)
	if err != nil {
		return transfers.ID{}, err
	}
	c := t.Settings.Container
	id, err := m.c.cfg.Transfers.Want(ctx, m.tx, transfers.Request{
		Purpose: transfers.Conversion, Job: outputJob(m.row.ID, m.row.Attempt), Source: *m.row.Node,
		Destination: transfers.Root{ID: *m.row.Root}, Priority: transfers.Low,
		Files: []transfers.FileRequest{{Name: outputName(m.row.ID, m.row.Attempt, c), Source: source.Scratch{Name: scratchOutput(m.row.ID, m.row.Attempt, c)}, Size: size}},
	})
	if err != nil {
		return transfers.ID{}, fmt.Errorf("return the output: %w", err)
	}
	return id, nil
}

func checkedTo(ctx context.Context, m *move) (Stage, error) {
	if m.code == "" {
		return StageLanding, nil
	}
	return faultTo(ctx, m)
}

// faultTo is where a fault takes a placed Conversion: a source fault
// fails it, and a run fault retries it or fails it by decision 8.
func faultTo(ctx context.Context, m *move) (Stage, error) {
	if faults[m.code].Class == SourceClass {
		return StageFailed, nil
	}
	attempts, err := m.attempts(ctx)
	if err != nil {
		return "", err
	}
	now := slices.IndexFunc(attempts, func(a Attempt) bool { return a.N == m.row.Attempt })
	attempts[now].Class = RunClass
	n := counted(attempts, m.row.BudgetFrom)
	m.retry = retryOf(n, true)
	if m.retry.Exclude != nil {
		others, err := m.othersCan(ctx, m.retry.Exclude)
		if err != nil {
			return "", err
		}
		m.retry = retryOf(n, others)
	}
	m.failedOn = names(n)
	if m.retry.Fail {
		return StageFailed, nil
	}
	return StageQueued, nil
}

// othersCan is whether a Node besides tried holds the Converting Role and
// hasn't lapsed, within the Target's Run on.
func (m *move) othersCan(ctx context.Context, tried []role.NodeID) (bool, error) {
	t, err := m.target(ctx)
	if err != nil {
		return false, err
	}
	holders := m.c.cfg.Board().Who(role.Converting)
	for _, n := range holders.Holding() {
		if !nodeIn(tried, n) && !holders.Lapsed(n) && (t.RunOn == nil || *t.RunOn == n) {
			return true, nil
		}
	}
	return false, nil
}

// faulted closes the attempt a fault ended, takes the Node's proof when
// the code costs it, and queues the retry or fails the Conversion.
func faulted(ctx context.Context, m *move) error {
	if m.code == "" {
		return nil
	}
	rule := faults[m.code]
	outcome := outcomeOf(m.code)
	if err := m.closeAttempt(ctx, outcome, m.code, rule.Class, capTail(m.ev.Tail)); err != nil {
		return err
	}
	if rule.Costs {
		if err := m.loseProof(ctx); err != nil {
			return err
		}
	}
	if err := m.leaveNode(ctx); err != nil {
		return err
	}
	if m.row.Stage == StageQueued {
		m.row.Pin, m.row.Exclude = m.retry.Pin, append(slices.Clone(m.row.Exclude), m.retry.Exclude...)
		return nil
	}
	return m.failed(ctx, rule.Class)
}

func fail(ctx context.Context, m *move) error { return m.failed(ctx, faults[m.code].Class) }

// leave takes a placed Conversion off its Node, closing its attempt with
// outcome.
func leave(outcome Outcome) effect {
	return func(ctx context.Context, m *move) error {
		if slices.Contains(faultStages, m.was.Stage) {
			if err := m.closeAttempt(ctx, outcome, "", "", ""); err != nil {
				return err
			}
		}
		m.row.PausedBy = nil
		return m.leaveNode(ctx)
	}
}

// leaveNode ends the Transfers the Conversion held on its Node: the
// source copy once nothing else there reads it, and a returning output.
func (m *move) leaveNode(ctx context.Context) error {
	if m.was.Node == nil {
		return nil
	}
	node := *m.was.Node
	m.row.Node = nil
	if m.was.Stage == StageReturning {
		if _, err := m.c.cfg.Transfers.Want(ctx, m.tx, transfers.Request{Purpose: transfers.Conversion, Job: outputJob(m.row.ID, m.was.Attempt)}); err != nil {
			return fmt.Errorf("end the output's Transfer: %w", err)
		}
	}
	if !slices.Contains(readingStages, m.was.Stage) {
		return nil
	}
	return m.doneReading(ctx, node)
}

// readingStages are where a Conversion reads its source on its Node.
var readingStages = []Stage{StageCopying, StageReady, StageRunning, StagePaused}

// doneReading ends the source copy on node once no other Conversion there
// reads it.
func (m *move) doneReading(ctx context.Context, node role.NodeID) error {
	if m.row.SourceFile == nil {
		return nil
	}
	readers, err := m.q.CountReaders(ctx, db.CountReadersParams{NodeID: node, SourceFileID: *m.row.SourceFile, ID: m.row.ID})
	if err != nil || readers > 0 {
		return err
	}
	if _, err := m.c.cfg.Transfers.Want(ctx, m.tx, transfers.Request{Purpose: transfers.Conversion, Job: sourceJob(node, *m.row.SourceFile)}); err != nil {
		return fmt.Errorf("end the source copy: %w", err)
	}
	return nil
}

// relaunch starts a lost run again on the same Node at the next attempt,
// with the pipeline it ran.
func relaunch(ctx context.Context, m *move) error {
	if err := m.closeAttempt(ctx, AttemptPutBack, "", "", ""); err != nil {
		return err
	}
	last, err := m.attempt(ctx, m.row.Attempt)
	if err != nil {
		return err
	}
	m.row.Attempt++
	m.row.PausedBy = nil
	return m.openAttempt(ctx, *m.row.Node, last.Pipeline)
}

func landed(_ context.Context, m *move) error {
	file := m.ev.File
	m.row.Output = &file
	return nil
}

// canceled records why the Conversion ended unfinished.
func canceled(_ context.Context, m *move) error {
	m.row.Failure, m.row.Pin = nil, nil
	m.row.CancelWhy = m.ev.Why
	if m.ev.On == Canceled {
		m.row.CancelWhy = CanceledByAdmin
	}
	return nil
}

// unneededTo cancels a Conversion no cell needs, except a landing one
// whose file is placing: that one lands, and the cell's next change
// decides it.
func unneededTo(ctx context.Context, m *move) (Stage, error) {
	err := m.endImport(ctx)
	if errors.Is(err, ErrPlacing) {
		return "", nil
	}
	return StageCanceled, err
}

func canceledTo(ctx context.Context, m *move) (Stage, error) {
	return StageCanceled, m.endImport(ctx)
}

func (m *move) endImport(ctx context.Context) error {
	if m.was.Stage != StageLanding {
		return nil
	}
	return m.c.importer.End(ctx, m.tx, m.row.ID)
}

// retried queues a failed Conversion at the head of its lane, with a
// budget counted from its next attempt.
func retried(ctx context.Context, m *move) error {
	head, err := m.q.HeadOfLane(ctx, string(m.row.Lane))
	if err != nil {
		return err
	}
	m.row.Position = order.Before(order.Key(head))
	m.row.BudgetFrom = m.row.Attempt + 1
	m.row.Pin, m.row.Exclude, m.row.Failure, m.row.Node = nil, nil, nil, nil
	return nil
}

// moveSpeed moves the proof's speed toward this run's, measured over its
// attempt and scaled to 1080p.
func moveSpeed(ctx context.Context, m *move) error {
	src, err := m.source(ctx)
	if err != nil {
		return err
	}
	t, err := m.target(ctx)
	if err != nil {
		return err
	}
	a, err := m.attempt(ctx, m.row.Attempt)
	if err != nil {
		return err
	}
	took := m.now.Sub(a.Started)
	key := encode.Key{Shape: t.Settings.Shape(), Class: encode.ClassOf(src.Media)}
	p, ok, err := m.proof(ctx, *m.row.Node, key)
	if err != nil || !ok || took <= 0 || src.Media.Duration <= 0 {
		return err
	}
	w, h := encode.Out(src.Media, t.Settings)
	measured := src.Media.Duration.Seconds() * float64(w*h) / (took.Seconds() * 1920 * 1080)
	return m.setSpeed(ctx, *m.row.Node, key, p+m.c.cfg.Run.SpeedWeight*(measured-p))
}
