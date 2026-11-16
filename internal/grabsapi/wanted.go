package grabsapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/works"
)

var wantedCursorStale = api.NewCode(http.StatusBadRequest, "wanted_cursor_stale", "This list changed. Load Wanted again.")

type wantedInput struct {
	ContentType string   `query:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$" doc:"Left out, every Content Type"`
	Reasons     []string `query:"reason" enum:"failed,missing,upgrade,gone" doc:"Left out, every reason but gone"`
	Sort        string   `query:"sort" enum:"next,last,title,released" default:"next" doc:"next is the next Backlog search first, last the longest since the last search first"`
	Cursor      string   `query:"cursor" doc:"The next value of the page before; left out, the first page"`
	Limit       int      `query:"limit" minimum:"1" maximum:"100" default:"100"`
}

type wantedPage struct {
	Status   queueStatus `json:"status"`
	Total    int         `json:"total" minimum:"0" doc:"The Works the filter lists"`
	LeftGone int         `json:"leftGone" minimum:"0" doc:"Works with only gone files, left out unless the filter asks for gone"`
	Rows     []wantedRow `json:"rows" nullable:"false" doc:"In the sort's order, ties by Work id"`
	Next     *string     `json:"next" doc:"The cursor of the next page; null on the last"`
}

type queueStatus struct {
	Searching   int          `json:"searching" minimum:"0" doc:"Search groups searching now"`
	Queued      int          `json:"queued" minimum:"0" doc:"Search groups queued"`
	NextBacklog *time.Time   `json:"nextBacklogAt" doc:"When Backlog search runs next; null while Backlog search is off"`
	Limits      []limitUntil `json:"limits" nullable:"false" doc:"Each Indexer whose search limit holds queued searches"`
}

type limitUntil struct {
	IndexerID   string     `json:"indexerId" format:"uuid"`
	IndexerName string     `json:"indexerName"`
	Until       *time.Time `json:"until" doc:"When its search limit frees a search; null while starved"`
	Starved     bool       `json:"starved" doc:"RSS sync uses all of its search limit, so Backlog search and Search all wait for it"`
}

type wantedWork struct {
	ID          string  `json:"id" format:"uuid"`
	ContentType string  `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
	Label       string  `json:"label"`
	PosterURL   *string `json:"posterUrl" doc:"The artwork route for its poster, without a size; null when it has none"`
}

type itemRange struct {
	First string `json:"first" doc:"The code of the first Item of the run, such as S02E01"`
	Last  string `json:"last" doc:"The code of the last Item of the run, the first's for a run of one"`
}

type wantedRow struct {
	Work   wantedWork     `json:"work"`
	Reason string         `json:"reason" enum:"failed,missing,upgrade,gone" doc:"The reason the row lists, the first in this order the Work has"`
	Others map[string]int `json:"others" doc:"The count of Items under each other reason, by reason"`
	Ranges []itemRange    `json:"ranges" nullable:"false" doc:"The Items it lists, as runs in tree order; empty for a Work of one Item"`
	LastAt *time.Time     `json:"lastSearchedAt" doc:"When its last search finished; null before one did"`
	Last   *string        `json:"lastResult" enum:"grabbed,waiting,nothing_found,nothing_acceptable,no_indexer,no_root" doc:"How its last search finished; null before one did"`
	NextAt *time.Time     `json:"nextSearchAt" doc:"When Backlog search searches it next; null for a gone row, which only a press searches, and while Backlog search is off"`
	State  string         `json:"state" enum:"idle,queued,searching"`
}

type wantedPageOutput struct{ Body wantedPage }

var listWantedOp = api.Op{
	ID:      "listWanted",
	Method:  http.MethodGet,
	Path:    "/wanted",
	Summary: "List the Works that want a Release, 100 at a time, with what the search queue is doing",
	Status:  http.StatusOK,
	Errors:  []*api.Code{wantedCursorStale},
}

func (h handlers) listWanted(ctx context.Context, by identity.Admin, in *wantedInput) (*wantedPageOutput, error) {
	q := grabs.WantedQuery{
		Filter: grabs.WantedFilter{Type: works.TypeID(in.ContentType)}, Sort: grabs.WantedSort(in.Sort), Cursor: in.Cursor, Limit: in.Limit,
	}
	for _, r := range in.Reasons {
		q.Filter.Reasons = append(q.Filter.Reasons, wantedReasons[r])
	}
	page, err := open(h.p).Wanted(ctx, by, q)
	if errors.Is(err, grabs.ErrCursorStale) {
		return nil, wantedCursorStale
	}
	if err != nil {
		return nil, err
	}
	m := minter(h.p)
	out := wantedPage{Status: queueStatusOf(page.Status), Total: page.Total, LeftGone: page.LeftGone, Rows: make([]wantedRow, len(page.Rows))}
	if page.Next != "" {
		out.Next = &page.Next
	}
	for i, r := range page.Rows {
		out.Rows[i] = wantedRowOf(m, r)
	}
	return &wantedPageOutput{Body: out}, nil
}

func queueStatusOf(s grabs.QueueStatus) queueStatus {
	out := queueStatus{Searching: s.Searching, Queued: s.Queued, NextBacklog: timeOrNil(s.NextBacklog), Limits: make([]limitUntil, len(s.Limits))}
	for i, l := range s.Limits {
		out.Limits[i] = limitUntil{IndexerID: l.Indexer.String(), IndexerName: l.Name, Until: timeOrNil(l.Until), Starved: l.Starved}
	}
	return out
}

func wantedRowOf(m artwork.Minter, r grabs.WantedRow) wantedRow {
	out := wantedRow{
		Work:   wantedWork{ID: r.Work.ID.String(), ContentType: string(r.Work.ContentType), Label: r.Work.Label},
		Reason: r.Reason.String(), Others: make(map[string]int, len(r.Others)), Ranges: make([]itemRange, len(r.Ranges)),
		NextAt: timeOrNil(r.Next), State: r.Phase.String(),
	}
	if !r.Poster.IsZero() {
		out.Work.PosterURL = new(m.URL(r.Poster))
	}
	for reason, n := range r.Others {
		out.Others[reason.String()] = n
	}
	for i, ir := range r.Ranges {
		out.Ranges[i] = itemRange{First: ir.First, Last: ir.Last}
	}
	if l := r.Last; l != nil {
		out.LastAt, out.Last = new(l.At), new(string(l.Outcome))
	}
	return out
}
