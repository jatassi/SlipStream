package transfers

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/bencode"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/ledger"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

var ErrNotFound = errors.New("no Transfer has that id")

type Filter struct {
	// Node is a Transfer's source or destination Node, zero for every
	// Node.
	Node role.NodeID
	// Source is a torrent one of its files comes from, nil for any.
	Source *bencode.InfoHash
}

func (f Filter) matches(v View) bool {
	if !f.Node.IsZero() && v.Source != f.Node && v.DestNode != f.Node {
		return false
	}
	return f.Source == nil || slices.ContainsFunc(v.Files, func(file File) bool {
		t, ok := file.Source.(source.Torrent)
		return ok && t.Hash == *f.Source
	})
}

// List is every Transfer f matches, in (Created, ID) order.
func (t *Transfers) List(ctx context.Context, f Filter) ([]View, error) {
	j, err := t.judgeNow(ctx, t.cfg.Board())
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(j.views, func(v View) bool { return !f.matches(v) }), nil
}

// Get is Transfer id, ErrNotFound when no record has it.
func (t *Transfers) Get(ctx context.Context, id ID) (View, error) {
	j, err := t.judgeNow(ctx, t.cfg.Board())
	if err != nil {
		return View{}, err
	}
	v, ok := j.view(id)
	if !ok {
		return View{}, ErrNotFound
	}
	return v, nil
}

// ViewsOf is p's Transfer for each of jobs that has one.
func (t *Transfers) ViewsOf(ctx context.Context, p PurposeKind, jobs []string) (map[string]View, error) {
	j, err := t.judgeNow(ctx, t.cfg.Board())
	if err != nil {
		return nil, err
	}
	out := map[string]View{}
	for _, v := range j.views {
		if v.Purpose == p && slices.Contains(jobs, v.Job) {
			out[v.Job] = v
		}
	}
	return out, nil
}

// Speeds is every pair's recent speed and last route.
func (t *Transfers) Speeds(ctx context.Context) ([]Pair, error) {
	var rows []ledger.Pair
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		rows, err = ledger.New(rd).ListPairs(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := make([]Pair, len(rows))
	for i, row := range rows {
		out[i] = Pair{Source: row.SourceNode, Destination: row.DestinationNode, Rate: row.Rate, RateAt: row.RateAt.Time()}
		if row.Route != nil {
			out[i].Route, out[i].RouteAt = tv1.Route(*row.Route), row.RouteAt.Time() //nolint:gosec // G115: the column checks it is 1, 2 or 3
		}
	}
	return out, nil
}
