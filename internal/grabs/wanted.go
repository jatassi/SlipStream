package grabs

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// WantedSort is the order Wanted lists its rows in, ties by Work id.
type WantedSort string

const (
	// ByNext is the next search first: queued and searching Works, then
	// by due time, then Works no search is due for.
	ByNext WantedSort = "next"
	// ByLast is the longest since the last search first, never searched
	// first of all.
	ByLast     WantedSort = "last"
	ByTitle    WantedSort = "title"
	ByReleased WantedSort = "released"
)

// ErrCursorStale is a cursor of another sort, or not one Wanted made.
var ErrCursorStale = errors.New("the cursor isn't one of this list")

// WantedQuery is one page of Wanted: the Works Filter lists in Sort's
// order, Limit of them after Cursor, from the first when Cursor is "".
type WantedQuery struct {
	Filter WantedFilter
	Sort   WantedSort
	Cursor string
	Limit  int
}

// QueueStatus is Wanted's status sentence: the groups searching and
// queued, when Backlog search runs next, zero while it is off, and each
// Indexer whose search limit holds planned lines.
type QueueStatus struct {
	Searching, Queued int
	NextBacklog       time.Time
	Limits            []Limit
}

// Limit is an Indexer whose search limit holds lines until Until, or
// whose limit RSS sync's reserve fills, Starved.
type Limit struct {
	Indexer indexers.ID
	Name    string
	Until   time.Time
	Starved bool
}

// WantedRow is one Work Wanted lists: its first Reason, the Items each
// other Reason is the first of, the runs of Items its Reason lists, and
// its search as the hero leads with it. Next is zero for a gone row,
// which only a press searches.
type WantedRow struct {
	Work   works.Summary
	Poster artwork.Ref
	Reason files.Reason
	Others map[files.Reason]int
	Ranges []ItemRange
	Last   *LastRun
	Next   time.Time
	Phase  Phase
}

// ItemRange is a run of Items in tree order, by their labels.
type ItemRange struct{ First, Last string }

// WantedPage is one page of Wanted. Total is the Works the filter lists,
// LeftGone those it leaves out because only gone cells want, and Next the
// cursor of the next page, "" on the last.
type WantedPage struct {
	Status   QueueStatus
	Total    int
	LeftGone int
	Rows     []WantedRow
	Next     string
}

// wantedListing is a Work Wanted lists and its sort key.
type wantedListing struct {
	wanted files.WorkWanted
	search WorkSearch
	key    sortKey
}

type sortKey struct {
	N int64  `json:"n"`
	T string `json:"t"`
	W string `json:"w"`
}

func (a sortKey) compare(b sortKey) int {
	return cmp.Or(cmp.Compare(a.N, b.N), cmp.Compare(a.T, b.T), cmp.Compare(a.W, b.W))
}

type cursor struct {
	Sort WantedSort `json:"s"`
	sortKey
}

// Wanted is one page of the Works that want a Release, read in one
// snapshot, with what the search queue is doing. Each row's search is the
// one Searching reads for its Work.
func (gr *Grabs) Wanted(ctx context.Context, _ identity.Admin, q WantedQuery) (WantedPage, error) {
	after, err := parseCursor(q)
	if err != nil {
		return WantedPage{}, err
	}
	next, err := gr.nextBacklog(ctx)
	if err != nil {
		return WantedPage{}, err
	}
	var page WantedPage
	err = gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		ws, err := gr.cfg.Files.Wanted(ctx, r, works.Every[works.WorkID](nil))
		if err != nil {
			return err
		}
		s, err := readStates(ctx, r, works.Every[works.WorkID](nil))
		if err != nil {
			return err
		}
		if page.Status, err = gr.status(ctx, r, s); err != nil {
			return err
		}
		if s.backlog {
			page.Status.NextBacklog = next
		}
		rows, left := filtered(ws, q.Filter)
		page.Total, page.LeftGone = len(rows), left
		if err := keyed(ctx, r, rows, s.byWork(), q.Sort); err != nil {
			return err
		}
		slices.SortFunc(rows, func(a, b wantedListing) int { return a.key.compare(b.key) })
		if after != nil {
			rows = rows[slices.IndexFunc(append(rows, wantedListing{key: sortKey{N: math.MaxInt64, T: "￿"}}), func(l wantedListing) bool {
				return l.key.compare(*after) > 0
			}):]
		}
		if len(rows) > q.Limit {
			rows = rows[:q.Limit]
			page.Next = encodeCursor(q.Sort, rows[len(rows)-1].key)
		}
		page.Rows, err = gr.wantedRows(ctx, r, rows)
		return err
	})
	if err != nil {
		return WantedPage{}, err
	}
	return page, nil
}

// filtered is the Works of ws f lists, and how many it leaves out
// because only gone cells want, as SearchSeveral reads a filter.
func filtered(ws []files.WorkWanted, f WantedFilter) ([]wantedListing, int) {
	reasons := f.Reasons
	if reasons == nil {
		reasons = []files.Reason{files.ReasonFailed, files.ReasonMissing, files.ReasonUpgrade}
	}
	var out []wantedListing
	left := 0
	for _, w := range ws {
		have := reasonsOf(w.Cells)
		switch {
		case len(have) == 0 || f.Type != "" && w.Type != f.Type:
		case slices.ContainsFunc(reasons, func(r files.Reason) bool { return slices.Contains(have, r) }):
			out = append(out, wantedListing{wanted: w})
		case slices.Equal(have, []files.Reason{files.ReasonGone}):
			left++
		}
	}
	return out, left
}

// keyed sets each row's search, from byWork, and its key under sort.
func keyed(ctx context.Context, r storage.Reader[file], rows []wantedListing, byWork map[works.WorkID]*WorkSearch, sort WantedSort) error {
	labels := map[works.WorkID]string{}
	if sort == ByTitle {
		ids := make([]works.WorkID, len(rows))
		for i, l := range rows {
			ids[i] = l.wanted.Work
		}
		trees, err := works.OutlinesIn(ctx, r, works.Only(ids))
		if err != nil {
			return err
		}
		for _, t := range trees {
			labels[t.ID] = t.SortLabel
		}
	}
	for i := range rows {
		l := &rows[i]
		if s := byWork[l.wanted.Work]; s != nil {
			l.search = *s
		}
		l.key.W = l.wanted.Work.String()
		switch sort {
		case ByNext:
			switch next := l.search.Next(); {
			case l.search.Phase() != Idle:
			case next.IsZero() || onlyGone(l.wanted):
				l.key.N = math.MaxInt64
			default:
				l.key.N = next.UnixMicro()
			}
		case ByLast:
			if last := l.search.Last(); last != nil {
				l.key.N = last.At.UnixMicro()
			}
		case ByTitle:
			l.key.T = labels[l.wanted.Work]
		case ByReleased:
			l.key.N = -newest(l.wanted.Cells).UnixMicro()
		}
	}
	return nil
}

func onlyGone(w files.WorkWanted) bool {
	lead, _, _ := files.ReasonOf(w.Cells)
	return lead == files.ReasonGone
}

// newest is the latest release of cells' Items.
func newest(cells []files.Cell) time.Time {
	var out time.Time
	for _, c := range cells {
		if from, ok := c.Timing.From(); ok && from.After(out) {
			out = from
		}
	}
	return out
}

// wantedRows is the rows of page, with each Work's summary, poster and
// the runs of Items its Reason lists.
func (gr *Grabs) wantedRows(ctx context.Context, r storage.Reader[file], page []wantedListing) ([]WantedRow, error) {
	ids := make([]works.WorkID, len(page))
	for i, l := range page {
		ids[i] = l.wanted.Work
	}
	rows, err := gr.cfg.Works.RowsIn(ctx, r, works.Scope{Works: ids})
	if err != nil {
		return nil, err
	}
	of := make(map[works.WorkID]works.Row, len(rows))
	for _, row := range rows {
		of[row.ID] = row
	}
	out := make([]WantedRow, len(page))
	for i, l := range page {
		w := of[l.wanted.Work]
		reason, others, _ := files.ReasonOf(l.wanted.Cells)
		row := WantedRow{
			Work: w.Summary, Poster: w.Poster, Reason: reason, Others: others, Ranges: rangesOf(w.Items, l.wanted.Cells, reason),
			Last: l.search.Last(), Phase: l.search.Phase(),
		}
		if reason != files.ReasonGone {
			row.Next = l.search.Next()
		}
		out[i] = row
	}
	return out, nil
}

// rangesOf is the runs, in the tree order of items, of the Items whose
// first Reason among cells is reason.
func rangesOf(items []works.Item, cells []files.Cell, reason files.Reason) []ItemRange {
	lists := map[works.ItemID]bool{}
	for len(cells) > 0 {
		n := 1 + slices.IndexFunc(cells[1:], func(c files.Cell) bool { return c.Item.ID != cells[0].Item.ID })
		if n == 0 {
			n = len(cells)
		}
		if first, _, ok := files.ReasonOf(cells[:n]); ok && first == reason {
			lists[cells[0].Item.ID] = true
		}
		cells = cells[n:]
	}
	out := []ItemRange{}
	open := false
	for _, it := range items {
		switch {
		case !lists[it.ID]:
			open = false
		case open:
			out[len(out)-1].Last = it.Label
		default:
			out, open = append(out, ItemRange{First: it.Label, Last: it.Label}), true
		}
	}
	return out
}

// status is the queue's status sentence in r, from s.
func (gr *Grabs) status(ctx context.Context, r storage.Reader[file], s states) (QueueStatus, error) {
	out := QueueStatus{Limits: []Limit{}}
	for _, d := range s.due {
		if !d.SearchID.IsZero() {
			out.Searching++
		}
		if d.Lane != nil {
			out.Queued++
		}
	}
	planned, err := db.New(r).ListPlannedLanes(ctx)
	if err != nil || len(planned) == 0 {
		return out, err
	}
	rooms, err := gr.cfg.Indexers.Room(ctx, r)
	if err != nil {
		return QueueStatus{}, err
	}
	for _, p := range planned {
		var b Behind
		b.limited(rooms, p.IndexerID, laneOf(p.Lane))
		switch i := slices.IndexFunc(out.Limits, func(l Limit) bool { return l.Indexer == p.IndexerID }); {
		case b.Indexer.IsZero():
		case i < 0:
			out.Limits = append(out.Limits, Limit{Indexer: b.Indexer, Name: b.Name, Until: b.Until, Starved: b.Starved})
		case b.Starved:
			out.Limits[i] = Limit{Indexer: b.Indexer, Name: b.Name, Starved: true}
		}
	}
	return out, nil
}

// nextBacklog is when Backlog search next opens on schedule, now while it
// runs and until Backlog search registers its task.
func (gr *Grabs) nextBacklog(ctx context.Context) (time.Time, error) {
	next, err := gr.cfg.Tasks.Next(ctx, BacklogTaskID)
	if err != nil && !errors.Is(err, tasks.ErrUnknownTask) {
		return time.Time{}, err
	}
	if next.IsZero() {
		return storage.Now().Time(), nil
	}
	return next, nil
}

func parseCursor(q WantedQuery) (*sortKey, error) {
	if q.Cursor == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(q.Cursor)
	if err != nil {
		return nil, ErrCursorStale
	}
	var c cursor
	if err := json.Unmarshal(raw, &c); err != nil || c.Sort != q.Sort {
		return nil, ErrCursorStale
	}
	return &c.sortKey, nil
}

func encodeCursor(sort WantedSort, k sortKey) string {
	raw, err := json.Marshal(cursor{Sort: sort, sortKey: k})
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}
