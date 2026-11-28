package conversions

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// key is the proof the Conversion runs under: its Target's shape and its
// source's class.
func (m *move) key(ctx context.Context) (encode.Key, error) {
	src, err := m.source(ctx)
	if err != nil {
		return encode.Key{}, err
	}
	t, err := m.target(ctx)
	if err != nil {
		return encode.Key{}, err
	}
	return encode.Key{Shape: t.Settings.Shape(), Class: encode.ClassOf(src.Media)}, nil
}

// loseProof takes the Node's proof of the Conversion's key until a new
// test run passes.
func (m *move) loseProof(ctx context.Context) error {
	k, err := m.key(ctx)
	if err != nil {
		return err
	}
	return m.q.LoseProof(ctx, db.LoseProofParams{LostAt: storage.FromTime(m.now), NodeID: *m.was.Node, Shape: k.Shape.String(), Class: string(k.Class)})
}

// proof is node's proven speed for k, false when it has no proof of it.
func (m *move) proof(ctx context.Context, node role.NodeID, k encode.Key) (speed float64, ok bool, err error) {
	p, err := m.q.GetProof(ctx, db.GetProofParams{NodeID: node, Shape: k.Shape.String(), Class: string(k.Class)})
	if errors.Is(err, storage.ErrNoRows) || (err == nil && p.Speed == nil) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read the proof: %w", err)
	}
	return *p.Speed, true, nil
}

func (m *move) setSpeed(ctx context.Context, node role.NodeID, k encode.Key, speed float64) error {
	return m.q.SetProofSpeed(ctx, db.SetProofSpeedParams{Speed: &speed, NodeID: node, Shape: k.Shape.String(), Class: string(k.Class)})
}

// prove stores node's test run of k, in tx, when it ran under the Node's
// current signature. A failed one moves every retry pinned to node for k
// on to another Node. A failed run names its last try, which proofOf
// requires.
func (c *Conversions) prove(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID, current converting.Signature, k encode.Key, p converting.Proved) error {
	if p.Signature != current {
		return nil
	}
	q := db.New(tx)
	at := storage.FromTime(c.cfg.Clock())
	row := db.UpsertProofParams{
		NodeID: node, Signature: string(p.Signature), Shape: k.Shape.String(), Class: string(k.Class),
		Path: string(p.Pipeline.Path), Api: string(p.Pipeline.API), Device: p.Device, Pauses: p.Pauses,
	}
	if p.Passed {
		row.Speed, row.ProvenAt = &p.Speed, at
	} else {
		last := p.Tries[len(p.Tries)-1]
		row.FailedAt, row.Error = at, &last.Err
		row.Path, row.Api = string(last.Pipeline.Path), string(last.Pipeline.API)
	}
	if err := q.UpsertProof(ctx, row); err != nil {
		return fmt.Errorf("store the test run: %w", err)
	}
	if p.Passed {
		return nil
	}
	pinned, err := q.ListPinnedTo(ctx, node)
	if err != nil {
		return err
	}
	for _, id := range pinned {
		m, err := c.moveOf(ctx, tx, id)
		if err != nil {
			return err
		}
		pk, err := m.key(ctx)
		if err != nil {
			return err
		}
		if pk != k {
			continue
		}
		if err := c.advance(ctx, tx, id, Event{On: Unpin, Node: node}); err != nil {
			return err
		}
	}
	return nil
}

// moveOf is a read-only move of row id, for its key.
func (c *Conversions) moveOf(ctx context.Context, tx *storage.Tx[storage.Library], id ID) (*move, error) {
	q := db.New(tx)
	raw, err := q.GetConversion(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read the Conversion: %w", err)
	}
	r, err := rowOf(raw)
	if err != nil {
		return nil, err
	}
	return &move{c: c, tx: tx, q: q, was: r, row: r, now: c.cfg.Clock()}, nil
}
