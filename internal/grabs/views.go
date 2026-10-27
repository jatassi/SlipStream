package grabs

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

// View is a Grab with its live download or Transfer progress, nil when
// none, and its metadata search, nil unless it fetches from a magnet
// link.
type View struct {
	Grab
	Progress *Progress
	Metadata *MetadataSearch
}

// List lists every Grab, newest first.
func (gr *Grabs) List(ctx context.Context, _ identity.Admin) ([]View, error) {
	var grabs []Grab
	var wait time.Duration
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := db.New(r).ListGrabs(ctx)
		if err != nil {
			return err
		}
		if grabs, err = loadAll(ctx, r, rows); err != nil {
			return err
		}
		wait, err = metadataWaitIn(ctx, r)
		return err
	})
	if err != nil {
		return nil, err
	}
	return gr.views(ctx, grabs, wait)
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
	var wait time.Duration
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
		if g, err = load(ctx, r, row, rel); err != nil {
			return err
		}
		wait, err = metadataWaitIn(ctx, r)
		return err
	})
	if err != nil {
		return View{}, err
	}
	views, err := gr.views(ctx, []Grab{g}, wait)
	if err != nil {
		return View{}, err
	}
	return views[0], nil
}

func (gr *Grabs) views(ctx context.Context, grabs []Grab, wait time.Duration) ([]View, error) {
	progress, err := gr.progress(ctx, grabs)
	if err != nil {
		return nil, err
	}
	out := make([]View, len(grabs))
	for i, g := range grabs {
		out[i] = View{Grab: g, Progress: progress[g.ID], Metadata: gr.metadataSearch(g, wait)}
	}
	return out, nil
}

func metadataWaitIn(ctx context.Context, r storage.Reader[file]) (time.Duration, error) {
	th, err := downloadclients.ThresholdsIn(ctx, r)
	return th.MetadataWait, err
}
