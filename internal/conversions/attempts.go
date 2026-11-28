package conversions

import (
	"context"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// openAttempt opens the row's current attempt on node, naming the Node
// for after it is gone.
func (m *move) openAttempt(ctx context.Context, node role.NodeID, p encode.Pipeline) error {
	named, err := nodes.Names(ctx, m.tx)
	if err != nil {
		return fmt.Errorf("read the Nodes' names: %w", err)
	}
	name := ""
	if i := slices.IndexFunc(named, func(n nodes.Named) bool { return n.ID == node }); i >= 0 {
		name = named[i].Name
	}
	return m.q.OpenAttempt(ctx, db.OpenAttemptParams{
		ConversionID: m.row.ID, N: int64(m.row.Attempt), NodeID: node, NodeName: name,
		Path: string(p.Path), Api: string(p.API), StartedAt: storage.FromTime(m.now),
	})
}

// closeAttempt ends the row's current attempt, if it is open.
func (m *move) closeAttempt(ctx context.Context, o Outcome, code Code, class Class, tail string) error {
	if m.row.Attempt == 0 {
		return nil
	}
	out := string(o)
	p := db.CloseAttemptParams{ConversionID: m.row.ID, N: int64(m.row.Attempt), EndedAt: storage.FromTime(m.now), Outcome: &out, Tail: tail}
	if code != "" {
		c, k := string(code), string(class)
		p.Code, p.Class = &c, &k
	}
	return m.q.CloseAttempt(ctx, p)
}

func (m *move) attempt(ctx context.Context, n int) (Attempt, error) {
	a, err := m.q.GetAttempt(ctx, db.GetAttemptParams{ConversionID: m.row.ID, N: int64(n)})
	if err != nil {
		return Attempt{}, fmt.Errorf("read attempt %d: %w", n, err)
	}
	return attemptOf(a), nil
}

func (m *move) attempts(ctx context.Context) ([]Attempt, error) {
	return attemptsOf(ctx, m.q, m.row.ID)
}

func attemptsOf(ctx context.Context, q *db.Queries, id ID) ([]Attempt, error) {
	rows, err := q.ListAttempts(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read the attempts: %w", err)
	}
	out := make([]Attempt, len(rows))
	for i, r := range rows {
		out[i] = attemptOf(r)
	}
	return out, nil
}

func attemptOf(a db.ConversionsAttempt) Attempt {
	out := Attempt{
		N: int(a.N), Node: a.NodeID, NodeName: a.NodeName, Started: a.StartedAt.Time(), Ended: timeOf(a.EndedAt),
		Pipeline: encode.Pipeline{Path: encode.Path(a.Path), API: encode.API(a.Api)}, Tail: a.Tail,
	}
	if a.Outcome != nil {
		out.Outcome = Outcome(*a.Outcome)
	}
	if a.Code != nil {
		out.Code, out.Class = Code(*a.Code), Class(*a.Class)
	}
	return out
}

// names is the Nodes as attempts ran on them, each once, in order.
func names(as []Attempt) []string {
	var out []string
	for _, a := range as {
		if !slices.Contains(out, a.NodeName) {
			out = append(out, a.NodeName)
		}
	}
	return out
}
