package grabs

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
)

// View is a Grab with its live download progress, nil when none, its
// metadata search, nil unless it fetches from a magnet link, and why its
// fetch waits on its Indexer, nil unless it does.
type View struct {
	Grab
	Progress  *Progress
	Metadata  *MetadataSearch
	FetchWait *indexers.FetchWait
}

// List lists every Grab, newest first.
func (gr *Grabs) List(ctx context.Context, _ identity.Admin) ([]View, error) {
	var grabs []Grab
	var wait time.Duration
	var waits map[indexers.ID]indexers.FetchWait
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := db.New(r).ListGrabs(ctx)
		if err != nil {
			return err
		}
		if grabs, err = loadAll(ctx, r, rows); err != nil {
			return err
		}
		if waits, err = gr.fetchWaits(ctx, r, grabs); err != nil {
			return err
		}
		wait, err = metadataWaitIn(ctx, r)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := make([]View, len(grabs))
	for i, g := range grabs {
		out[i] = gr.view(g, wait, waits)
	}
	return out, nil
}

// fetchWaits is why the fetch of each of grabs that fetches a .torrent
// from its Indexer waits now, by Indexer.
func (gr *Grabs) fetchWaits(ctx context.Context, r storage.Reader[file], grabs []Grab) (map[indexers.ID]indexers.FetchWait, error) {
	var ids []indexers.ID
	for _, g := range grabs {
		if fromIndexer(g) && !slices.Contains(ids, g.Ticket.Indexer) {
			ids = append(ids, g.Ticket.Indexer)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return gr.cfg.Indexers.FetchWaits(ctx, r, ids)
}

// fromIndexer reports whether g fetches a .torrent from its Indexer now.
func fromIndexer(g Grab) bool {
	return g.Stage == Fetching && g.Release.link.Set() && !g.Ticket.Indexer.IsZero()
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
	var waits map[indexers.ID]indexers.FetchWait
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
		if waits, err = gr.fetchWaits(ctx, r, []Grab{g}); err != nil {
			return err
		}
		wait, err = metadataWaitIn(ctx, r)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return gr.view(g, wait, waits), nil
}

func (gr *Grabs) view(g Grab, wait time.Duration, waits map[indexers.ID]indexers.FetchWait) View {
	v := View{Grab: g, Progress: gr.progress(g), Metadata: gr.metadataSearch(g, wait)}
	if w, ok := waits[g.Ticket.Indexer]; ok && fromIndexer(g) {
		v.FetchWait = &w
	}
	return v
}

func metadataWaitIn(ctx context.Context, r storage.Reader[file]) (time.Duration, error) {
	th, err := downloadclients.ThresholdsIn(ctx, r)
	return th.MetadataWait, err
}
