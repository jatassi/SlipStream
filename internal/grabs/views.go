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

// View is a Grab with its live download or Transfer progress, nil when
// none, its metadata search, nil unless it fetches from a magnet link, why
// its fetch waits on its Indexer, nil unless it does, and the client whose
// Node it waits for, nil unless it does.
type View struct {
	Grab
	Progress  *Progress
	Metadata  *MetadataSearch
	FetchWait *indexers.FetchWait
	WaitsFor  *ClientWait
	// Overrides is each choice the Admin made past the decision, as a
	// sentence: "Jackson grabbed it as the Director's Cut."
	Overrides []string
	Actions   Actions
}

// Actions is what an Admin can do to a Grab now.
type Actions struct {
	Cancel bool
	// Blocklist is true when a cancel can also blocklist its Release: its
	// info hash is known.
	Blocklist bool
	SendTo    bool
}

// reads is what a Grab's view reads beside the Grab, in the same snapshot.
type reads struct {
	metadataWait time.Duration
	fetchWaits   map[indexers.ID]indexers.FetchWait
	clientWaits  map[GrabID]ClientWait
}

func (gr *Grabs) readsIn(ctx context.Context, r storage.Reader[file], grabs []Grab) (reads, error) {
	var (
		out reads
		err error
	)
	if out.fetchWaits, err = gr.fetchWaits(ctx, r, grabs); err != nil {
		return reads{}, err
	}
	if out.clientWaits, err = gr.clientWaits(ctx, r, grabs); err != nil {
		return reads{}, err
	}
	out.metadataWait, err = metadataWaitIn(ctx, r)
	return out, err
}

// List lists every Grab, newest first.
func (gr *Grabs) List(ctx context.Context, _ identity.Admin) ([]View, error) {
	return gr.listed(ctx, func(ctx context.Context, r storage.Reader[file]) ([]db.GrabsGrab, error) {
		return db.New(r).ListGrabs(ctx)
	})
}

// OfIndexer lists the page p of Indexer id's Grabs, newest first,
// indexers.ErrNoIndexer when no Indexer has id.
func (gr *Grabs) OfIndexer(ctx context.Context, _ identity.Admin, id indexers.ID, p indexers.Page) ([]View, error) {
	return gr.listed(ctx, func(ctx context.Context, r storage.Reader[file]) ([]db.GrabsGrab, error) {
		if _, err := indexers.Get(ctx, r, id); err != nil {
			return nil, err
		}
		return db.New(r).ListGrabsOfIndexer(ctx, db.ListGrabsOfIndexerParams{IndexerID: id, Lim: int64(p.Limit), Off: int64(p.Offset)})
	})
}

func (gr *Grabs) listed(ctx context.Context, list func(context.Context, storage.Reader[file]) ([]db.GrabsGrab, error)) ([]View, error) {
	var grabs []Grab
	var rd reads
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := list(ctx, r)
		if err != nil {
			return err
		}
		if grabs, err = loadAll(ctx, r, rows); err != nil {
			return err
		}
		rd, err = gr.readsIn(ctx, r, grabs)
		return err
	})
	if err != nil {
		return nil, err
	}
	return gr.views(ctx, grabs, rd)
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
	var rd reads
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
		rd, err = gr.readsIn(ctx, r, []Grab{g})
		return err
	})
	if err != nil {
		return View{}, err
	}
	views, err := gr.views(ctx, []Grab{g}, rd)
	if err != nil {
		return View{}, err
	}
	return views[0], nil
}

func (gr *Grabs) views(ctx context.Context, grabs []Grab, rd reads) ([]View, error) {
	progress, err := gr.progress(ctx, grabs)
	if err != nil {
		return nil, err
	}
	out := make([]View, len(grabs))
	for i, g := range grabs {
		out[i] = View{Grab: g, Progress: progress[g.ID], Metadata: gr.metadataSearch(g, rd.metadataWait)}
		if w, ok := rd.fetchWaits[g.Ticket.Indexer]; ok && fromIndexer(g) {
			out[i].FetchWait = &w
		}
		if w, ok := rd.clientWaits[g.ID]; ok {
			out[i].WaitsFor = &w
		}
	}
	return out, nil
}

func metadataWaitIn(ctx context.Context, r storage.Reader[file]) (time.Duration, error) {
	th, err := downloadclients.ThresholdsIn(ctx, r)
	return th.MetadataWait, err
}
