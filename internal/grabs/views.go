package grabs

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
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
	// ReleaseWords is its Release in words: "Bluray-1080p · FKW · 9.8 GB".
	ReleaseWords string
	// Overrides is each choice the Admin made past the decision, as a
	// sentence: "Jackson grabbed it as the Director's Cut."
	Overrides []string
	Actions   Actions
	// Current is whether a cell reads from it. A failure stays under
	// Needs you only while it is.
	Current bool
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
	releaseWords map[GrabID]string
	current      map[GrabID]bool
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
	if out.releaseWords, err = gr.releaseWords(ctx, r, grabs); err != nil {
		return reads{}, err
	}
	if out.current, err = grabsCellsReadFrom(ctx, r, grabs); err != nil {
		return reads{}, err
	}
	out.metadataWait, err = metadataWaitIn(ctx, r)
	return out, err
}

func grabsCellsReadFrom(ctx context.Context, r storage.Reader[file], grabs []Grab) (map[GrabID]bool, error) {
	seen := map[works.WorkID]bool{}
	var ids []works.WorkID
	for _, g := range grabs {
		if !seen[g.WorkID] {
			seen[g.WorkID] = true
			ids = append(ids, g.WorkID)
		}
	}
	q := db.New(r)
	rows, err := works.Read(works.Only(ids), nil, func(ids []works.WorkID) ([]db.ListTargetsOfWorksRow, error) {
		return q.ListTargetsOfWorks(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	out := map[GrabID]bool{}
	for _, row := range readFrom(rows) {
		if row.Settled == nil {
			out[row.ID] = true
		}
	}
	return out, nil
}

// releaseWords is each Grab's Release in words, read as its Work's
// Content Type reads a title.
func (gr *Grabs) releaseWords(ctx context.Context, r storage.Reader[file], grabs []Grab) (map[GrabID]string, error) {
	ids := make([]works.WorkID, len(grabs))
	for i, g := range grabs {
		ids[i] = g.WorkID
	}
	summaries, err := works.SummariesOf(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[GrabID]string, len(grabs))
	for _, g := range grabs {
		out[g.ID] = releaseWordsOf(g.Release.Title, g.Release.Size, gr.cfg.Names[summaries[g.WorkID].ContentType])
	}
	return out, nil
}

// releaseWordsOf is a Release titled title, of size bytes, in the words a
// Manual search's rows read it in, as names read its title. The words
// name no cut, so the title is read without the known cuts.
func releaseWordsOf(title string, size int64, names release.Names) string {
	rd := release.Title(title, names, nil)
	if size > 0 {
		rd.Facts.Size = release.Known(size)
	}
	return release.Describe(rd)
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
	return gr.views(grabs, rd)
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

// Moving counts the Grabs in flight.
func (gr *Grabs) Moving(ctx context.Context, _ identity.Admin) (int, error) {
	var n int64
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		n, err = db.New(r).CountMoving(ctx, inFlightStages)
		return err
	})
	return int(n), err
}

// Get reads Grab id, ErrNoGrab when none has it.
func (gr *Grabs) Get(ctx context.Context, _ identity.Admin, id GrabID) (View, error) {
	var g Grab
	var rd reads
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if g, err = grabIn(ctx, r, id); err != nil {
			return err
		}
		rd, err = gr.readsIn(ctx, r, []Grab{g})
		return err
	})
	if err != nil {
		return View{}, err
	}
	views, err := gr.views([]Grab{g}, rd)
	if err != nil {
		return View{}, err
	}
	return views[0], nil
}

func (gr *Grabs) views(grabs []Grab, rd reads) ([]View, error) {
	progress := gr.progress(grabs)
	out := make([]View, len(grabs))
	for i, g := range grabs {
		out[i] = View{Grab: g, Progress: progress[g.ID], Metadata: gr.metadataSearch(g, rd.metadataWait)}
		if w, ok := rd.fetchWaits[g.Ticket.Indexer]; ok && fromIndexer(g) {
			out[i].FetchWait = &w
		}
		if w, ok := rd.clientWaits[g.ID]; ok {
			out[i].WaitsFor = &w
		}
		out[i].Actions = Actions{
			Cancel: g.cancelable(), Blocklist: g.cancelable() && g.infoHash() != nil,
			SendTo: g.waitsOnTorrent() || out[i].WaitsFor != nil,
		}
		out[i].ReleaseWords = rd.releaseWords[g.ID]
		out[i].Overrides = g.overrideWords()
		out[i].Current = rd.current[g.ID]
	}
	return out, nil
}

func metadataWaitIn(ctx context.Context, r storage.Reader[file]) (time.Duration, error) {
	th, err := downloadclients.ThresholdsIn(ctx, r)
	return th.MetadataWait, err
}
