package grabs

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

// View is a Grab with its live download progress, nil when none.
type View struct {
	Grab
	Progress *Progress
}

// List lists every Grab, newest first.
func (gr *Grabs) List(ctx context.Context, _ identity.Admin) ([]View, error) {
	var grabs []Grab
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := db.New(r).ListGrabs(ctx)
		if err != nil {
			return err
		}
		grabs, err = loadAll(ctx, r, rows)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := make([]View, len(grabs))
	for i, g := range grabs {
		out[i] = View{g, gr.progress(g)}
	}
	return out, nil
}

// Moving counts the Grabs not yet imported or failed.
func (gr *Grabs) Moving(ctx context.Context, _ identity.Admin) (int, error) {
	var n int64
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		n, err = db.New(r).CountMoving(ctx)
		return err
	})
	return int(n), err
}

// Get reads Grab id, ErrNoGrab when none has it.
func (gr *Grabs) Get(ctx context.Context, _ identity.Admin, id GrabID) (View, error) {
	var g Grab
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		row, err := db.New(r).GetGrab(ctx, id)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNoGrab
		}
		if err != nil {
			return err
		}
		rel, err := readRelated(ctx, r, []db.GrabsGrab{row})
		if err != nil {
			return err
		}
		g, err = load(ctx, r, row, rel)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return View{g, gr.progress(g)}, nil
}
