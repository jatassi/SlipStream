package moves

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Dropped is a planned unit a Start left out, and why.
type Dropped struct {
	Work    works.WorkID
	Torrent *downloadclients.ID
	Label   string
	Words   string
}

// Started is a Start's batch: its units, and the planned units it left
// out.
type Started struct {
	Batch   BatchID
	Units   int
	Dropped []Dropped
}

// Start runs the plan token names, once. confirm is the movable count the
// Admin saw; a plan that now moves another count is *PlanChangedError. A
// Start again of one token answers the same batch. It checks again what
// the Controller alone knows and drops each unit red now, or whose Work
// another batch took since the preview.
func (m *Moves) Start(ctx context.Context, by identity.Admin, token Token, confirm int) (Started, error) {
	now := m.now()
	p, ok := m.plans.get(token, now)
	if !ok {
		return Started{}, ErrPlanExpired
	}
	if p.Movable() != confirm {
		return Started{}, &PlanChangedError{Was: confirm, Now: p.Movable()}
	}
	var out Started
	var inserted []Unit
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		b, err := q.BatchOfToken(ctx, token.hash())
		switch {
		case err == nil:
			out, err = startedOf(ctx, q, b.ID)
			return err
		case !errors.Is(err, storage.ErrNoRows):
			return err
		}
		if err := m.fence(ctx, tx); err != nil {
			return err
		}
		keep, dropped, err := m.recheck(ctx, tx, p)
		if err != nil {
			return err
		}
		if inserted, err = m.insert(ctx, tx, by, p, keep, now); err != nil {
			return err
		}
		out = Started{Batch: inserted[0].Batch, Units: len(inserted), Dropped: dropped}
		for _, h := range p.Homed {
			err := m.cfg.Roots.Home(ctx, tx, h.Work, h.To)
			if err != nil && !errors.Is(err, roots.ErrHasFiles) && !errors.Is(err, works.ErrNoWork) {
				return err
			}
		}
		return m.reroute(ctx, tx, p.Ask)
	})
	if err != nil {
		return Started{}, err
	}
	if len(inserted) > 0 {
		m.cfg.Changed.Publish(ctx, changedOf(inserted...))
		m.Poke()
	}
	return out, nil
}

// reroute has the routes to each client ask moves go to its destination,
// when ask repoints routing.
func (m *Moves) reroute(ctx context.Context, tx *storage.Tx[file], ask Ask) error {
	if !ask.Options.RepointRouting {
		return nil
	}
	for _, c := range ask.Clients {
		if err := m.cfg.Reroute(ctx, tx, choiceOf(c.From), choiceOf(c.To)); err != nil {
			return err
		}
	}
	return nil
}

func choiceOf(s Side) downloadclients.Choice {
	return downloadclients.Choice{Client: s.Client, BuiltInOn: s.Node}
}

// ErrNothingToMove refuses a Start whose every unit is red or taken.
var ErrNothingToMove = errors.New("nothing in the preview can move now. Preview the Move again")

func startedOf(ctx context.Context, q *db.Queries, batch BatchID) (Started, error) {
	counts, err := q.CountBatchPhases(ctx, batch)
	out := Started{Batch: batch}
	for _, c := range counts {
		out.Units += int(c.N)
	}
	return out, err
}

// recheck is p's units that may still move, judged again by what the
// Controller reads now, and the ones it drops.
func (m *Moves) recheck(ctx context.Context, tx *storage.Tx[file], p Plan) (keep []PlanUnit, dropped []Dropped, err error) {
	var ws []works.WorkID
	var ids []roots.ID
	for _, u := range p.Units {
		if u.Torrent == nil {
			ws = append(ws, u.Subject.Work)
			ids = appendOnce(ids, u.Subject.To.Root)
		}
	}
	open, err := db.New(tx).ListOpenUnitsOfWorks(ctx, ws)
	if err != nil {
		return nil, nil, err
	}
	moving := map[works.WorkID]bool{}
	for _, row := range open {
		moving[row.WorkID] = true
	}
	dests, err := m.cfg.Destinations(ctx, tx, ids)
	if err != nil {
		return nil, nil, err
	}
	for _, u := range p.Units {
		r, red := Reason{}, false
		switch {
		case u.Torrent != nil && u.Verdict.Color != Red:
			if r, red, err = recheckTorrent(ctx, tx, u, m.now(), p.Ask.Origin == OriginDrain); err != nil {
				return nil, nil, err
			}
		case u.Verdict.Color == Red:
			r, red = worst(u.Verdict), true
		case moving[u.Subject.Work]:
			r, red = movingCheck(u, planFacts{moving: true})
		case dests[u.Subject.To.Root].State != roots.Ready:
			r, red = notReady, true
		}
		if red {
			d := Dropped{Work: u.Subject.Work, Label: u.Label, Words: r.Words}
			if u.Torrent != nil {
				d.Torrent = &u.Torrent.Torrent
			}
			dropped = append(dropped, d)
			continue
		}
		keep = append(keep, u)
	}
	if len(keep) == 0 {
		return nil, nil, ErrNothingToMove
	}
	return keep, dropped, nil
}

func worst(v Verdict) Reason {
	for _, r := range v.Reasons {
		if r.Color == v.Color {
			return r
		}
	}
	return Reason{}
}

// insertUnit writes queued unit u, waiting wait, with its files.
func insertUnit(ctx context.Context, q *db.Queries, u Unit, wait Wait) error {
	notes, err := json.Marshal(u.Notes)
	if err != nil {
		return fmt.Errorf("moves: the notes of unit %s: %w", u.ID, err)
	}
	row := db.InsertUnitParams{
		ID: u.ID, BatchID: u.Batch, Seq: u.Seq, Position: int64(u.Position), Kind: "work", Lane: string(u.Lane),
		PlannedBytes: u.Planned.Bytes, PlannedAt: storage.FromTime(u.Planned.At), Notes: string(notes),
	}
	if t := u.Torrent; t != nil {
		row.Kind, row.TorrentID, row.FromNodeID, row.ToNodeID = "torrent", t.Torrent.UUID, t.From.Node, t.To.Node
		if t.From.Client != nil {
			row.FromClientID = t.From.Client.UUID
		}
		if t.To.Client != nil {
			row.ToClientID = t.To.Client.UUID
		}
	} else {
		s := u.Subject
		row.WorkID, row.ContentType, row.FromRootID, row.FromNodeID, row.ToRootID, row.ToNodeID = s.Work, &s.Type, s.From.Root, s.From.Node, s.To.Root, s.To.Node
	}
	if wait != "" {
		row.Waiting, row.WaitingAt = new(string(wait)), storage.FromTime(u.Planned.At)
	}
	if u.PairsWith != nil {
		row.PairsWithUnitID = *u.PairsWith
	}
	if err := q.InsertUnit(ctx, row); err != nil {
		return err
	}
	for _, f := range u.Files {
		if err := q.InsertUnitFile(ctx, insertFileColumns(u.ID, f)); err != nil {
			return err
		}
	}
	return nil
}

func amberOf(v Verdict) []Reason {
	var out []Reason
	for _, r := range v.Reasons {
		if r.Color == Amber {
			out = append(out, r)
		}
	}
	return out
}

// batchRow is the moves_batch row p starts as batch, with the roots it
// empties in its notes.
func batchRow(batch BatchID, by identity.Admin, p Plan, now time.Time) (db.InsertBatchParams, error) {
	b := db.InsertBatchParams{
		ID: batch, Origin: string(p.Ask.Origin), CreatedAt: storage.FromTime(now), HomedWorks: int64(len(p.Homed)), PlanTokenHash: p.Token.hash(),
	}
	if a, ok := by.Account(); ok {
		b.CreatedBy = a.Profile().ID.UUID
	}
	notes := batchNotes{}
	if p.Ask.Options.DeleteEmptiedRoots {
		b.DeleteEmptiedRoots = 1
		for _, rt := range p.Ask.Roots {
			notes.Emptying = append(notes.Emptying, emptied{From: rt.Root, To: rt.To})
		}
	}
	raw, err := json.Marshal(notes)
	if err != nil {
		return b, fmt.Errorf("moves: the notes of batch %s: %w", batch, err)
	}
	b.Notes = string(raw)
	if p.Ask.Options.IncludeLinkedTorrents {
		b.IncludeTorrents = 1
	}
	if p.Ask.Options.RepointRouting {
		b.RepointRouting = 1
	}
	return b, nil
}

// insert writes p's batch and keep's units with their files. Each unit
// but the first of its lane in this batch waits pair_busy.
func (m *Moves) insert(ctx context.Context, tx *storage.Tx[file], by identity.Admin, p Plan, keep []PlanUnit, now time.Time) ([]Unit, error) {
	q := db.New(tx)
	batch := BatchID{UUID: storage.NewUUID()}
	b, err := batchRow(batch, by, p, now)
	if err != nil {
		return nil, err
	}
	if err := q.InsertBatch(ctx, b); err != nil {
		return nil, err
	}
	seq, err := q.NextSeq(ctx)
	if err != nil {
		return nil, err
	}
	lanes := map[Lane]bool{}
	torrents := map[downloadclients.ID]UnitID{}
	out := make([]Unit, 0, len(keep))
	for i, pu := range keep {
		var wait Wait
		if lanes[pu.Lane] {
			wait = WaitPairBusy
		}
		lanes[pu.Lane] = true
		u := Unit{
			ID: UnitID{UUID: storage.NewUUID()}, Batch: batch, Seq: seq + int64(i), Position: i, Lane: pu.Lane, Subject: pu.Subject, Torrent: pu.Torrent,
			Stage: Queued{Wait: wait, Since: now}, Planned: Planned{Bytes: pu.Bytes, At: now}, Files: pu.Files, Notes: Notes{Amber: amberOf(pu.Verdict)},
		}
		if pu.Torrent != nil {
			torrents[pu.Torrent.Torrent] = u.ID
		}
		if pu.Linked != nil {
			if t, ok := torrents[*pu.Linked]; ok {
				u.PairsWith, wait = &t, WaitLinkedTorrent
				u.Stage = Queued{Wait: wait, Since: now}
			} else {
				u.Files = fromLibrary(u.Files)
			}
		}
		if err := insertUnit(ctx, q, u, wait); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}
