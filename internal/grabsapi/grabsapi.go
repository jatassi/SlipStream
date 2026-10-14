// Package grabsapi is the search and Grab operations: start a Work's
// search, list its search runs, and list and read Grabs with their live
// progress.
package grabsapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type Pipeline interface {
	// Grabs is the grab pipeline, false unless the Node is a Controller.
	Grabs() (*grabs.Grabs, bool)
}

func open(p Pipeline) *grabs.Grabs {
	g, ok := p.Grabs()
	if !ok {
		panic("grabsapi: an operation ran on a Node that is no Controller")
	}
	return g
}

var (
	workNotFound  = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
	nothingWanted = api.NewCode(http.StatusConflict, "nothing_wanted", "The Work has no wanted cell to search for")
	noIndexer     = api.NewCode(http.StatusConflict, "no_indexer", "No Indexer is set up")
	noRoot        = api.NewCode(http.StatusConflict, "no_root", "The Work's Content Type has no library root")
	notSearchable = api.NewCode(http.StatusConflict, "not_searchable", "The Work's Content Type can't be searched")
	grabNotFound  = api.NewCode(http.StatusNotFound, "grab_not_found", "No Grab has that id")
)

func answer(err error) error {
	switch {
	case errors.Is(err, grabs.ErrNoWork):
		return workNotFound
	case errors.Is(err, grabs.ErrNothingWanted):
		return nothingWanted
	case errors.Is(err, grabs.ErrNoIndexer):
		return noIndexer
	case errors.Is(err, grabs.ErrNoRoot):
		return noRoot
	case errors.Is(err, grabs.ErrNotSearchable):
		return notSearchable
	case errors.Is(err, grabs.ErrNoGrab):
		return grabNotFound
	}
	return err
}

type workInput struct {
	WorkID string `path:"workId" format:"uuid"`
}

type searchStarted struct {
	SearchIDs []string `json:"searchIds" nullable:"false" doc:"One run per Item with a wanted cell"`
}

type searchStartedOutput struct{ Body searchStarted }

type search struct {
	ID         string     `json:"id" format:"uuid"`
	ItemID     string     `json:"itemId" format:"uuid"`
	IndexerID  string     `json:"indexerId" format:"uuid"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt" doc:"Null while the run searches"`
	Outcome    *string    `json:"outcome" enum:"grabbed,nothing_acceptable,indexer_unavailable,indexer_refused" doc:"Null while the run searches"`
	Candidates int        `json:"candidates" minimum:"0" doc:"The releases the Indexer answered"`
	GrabID     *string    `json:"grabId" format:"uuid" doc:"The Grab the run made, null when none"`
}

type searchList struct {
	Searches []search `json:"searches" nullable:"false" doc:"Newest first"`
}

type searchListOutput struct{ Body searchList }

func searchOf(s grabs.Search) search {
	out := search{ID: s.ID.String(), ItemID: s.ItemID.String(), IndexerID: s.Indexer.String(), StartedAt: s.Started, Candidates: s.Candidates}
	if !s.Finished.IsZero() {
		finished, outcome := s.Finished, string(s.Outcome)
		out.FinishedAt, out.Outcome = &finished, &outcome
	}
	if !s.GrabID.IsZero() {
		id := s.GrabID.String()
		out.GrabID = &id
	}
	return out
}

type failure struct {
	Stage   string `json:"stage" enum:"fetching,downloading,probing,transferring,importing"`
	Code    string `json:"code" enum:"indexer_unavailable,torrent_unreachable,torrent_invalid,infohash_mismatch,torrent_v2_only,no_download_client,download_failed,no_video_file,probe_failed,fills_nothing,transfer_failed,import_failed"`
	Message string `json:"message"`
}

type entered struct {
	Stage string    `json:"stage" enum:"fetching,downloading,probing,transferring,importing,imported,failed"`
	At    time.Time `json:"at"`
}

type progress struct {
	Done  int64 `json:"done" minimum:"0" doc:"Bytes downloaded"`
	Total int64 `json:"total" minimum:"0" doc:"Bytes wanted"`
}

type grab struct {
	ID           string    `json:"id" format:"uuid"`
	WorkID       string    `json:"workId" format:"uuid"`
	ItemID       string    `json:"itemId" format:"uuid"`
	Edition      string    `json:"edition" enum:"standard"`
	TargetIDs    []string  `json:"targetIds" nullable:"false" doc:"The Targets it was grabbed for, in Profile order"`
	ReleaseTitle string    `json:"releaseTitle"`
	ReleaseSize  int64     `json:"releaseSize" minimum:"0"`
	IndexerID    string    `json:"indexerId" format:"uuid"`
	NodeID       string    `json:"nodeId" format:"uuid" doc:"The Node that downloads it: the Indexer's home Node"`
	InfoHash     *string   `json:"infoHash" doc:"The v1 info hash in hex, null until the .torrent is fetched"`
	Stage        string    `json:"stage" enum:"fetching,downloading,probing,transferring,importing,imported,failed"`
	Failure      *failure  `json:"failure,omitempty" doc:"Why it failed, absent unless it did"`
	History      []entered `json:"history" nullable:"false" doc:"Each stage it entered, oldest first"`
	Progress     *progress `json:"progress,omitempty" doc:"The download's progress, absent unless it is downloading and its Node reported it"`
	CreatedAt    time.Time `json:"createdAt"`
}

func grabOf(v grabs.View) grab {
	out := grab{
		ID: v.ID.String(), WorkID: v.WorkID.String(), ItemID: v.ItemID.String(), Edition: string(v.Edition), TargetIDs: []string{},
		ReleaseTitle: v.Release.Title, ReleaseSize: v.Release.Size, IndexerID: v.Release.Indexer.String(), NodeID: v.Node.String(),
		Stage: string(v.Stage), History: []entered{}, CreatedAt: v.Created,
	}
	for _, t := range v.Targets {
		out.TargetIDs = append(out.TargetIDs, t.String())
	}
	if v.Torrent != nil {
		h := v.Torrent.InfoHash.String()
		out.InfoHash = &h
	}
	if v.Failed != nil {
		out.Failure = &failure{Stage: string(v.Failed.Stage), Code: string(v.Failed.Code), Message: v.Failed.Message}
	}
	for _, e := range v.History {
		out.History = append(out.History, entered{Stage: string(e.Stage), At: e.At})
	}
	if v.Progress != nil {
		out.Progress = &progress{Done: v.Progress.Done, Total: v.Progress.Total}
	}
	return out
}

type grabList struct {
	Grabs []grab `json:"grabs" nullable:"false" doc:"Newest first"`
}

type grabListOutput struct{ Body grabList }

type grabInput struct {
	GrabID string `path:"grabId" format:"uuid"`
}

type grabOutput struct{ Body grab }

func Routes(r *api.Router, p Pipeline) {
	o := r.Owner("grabs", api.Admin)
	h := handlers{p}

	api.Register(o, api.Admin, api.Op{
		ID:      "searchWork",
		Method:  http.MethodPost,
		Path:    "/works/{workId}/search",
		Summary: "Search for each of a Work's wanted cells and grab the best release",
		Status:  http.StatusAccepted,
		Errors:  []*api.Code{workNotFound, nothingWanted, noIndexer, noRoot, notSearchable},
	}, h.searchWork)
	api.Register(o, api.Admin, api.Op{
		ID:      "listSearches",
		Method:  http.MethodGet,
		Path:    "/works/{workId}/searches",
		Summary: "List a Work's search runs and their outcomes",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound},
	}, h.listSearches)
	api.Register(o, api.Admin, api.Op{
		ID:      "listGrabs",
		Method:  http.MethodGet,
		Path:    "/grabs",
		Summary: "List every Grab with its stage and progress",
		Status:  http.StatusOK,
	}, h.listGrabs)
	api.Register(o, api.Admin, api.Op{
		ID:      "getGrab",
		Method:  http.MethodGet,
		Path:    "/grabs/{grabId}",
		Summary: "Get a Grab with its stage and progress",
		Status:  http.StatusOK,
		Errors:  []*api.Code{grabNotFound},
	}, h.getGrab)
}

type handlers struct{ p Pipeline }

func (h handlers) searchWork(ctx context.Context, by identity.Admin, in *workInput) (*searchStartedOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	ids, err := open(h.p).SearchWork(ctx, by, works.WorkID{UUID: u})
	if err != nil {
		return nil, answer(err)
	}
	out := searchStarted{SearchIDs: []string{}}
	for _, id := range ids {
		out.SearchIDs = append(out.SearchIDs, id.String())
	}
	return &searchStartedOutput{Body: out}, nil
}

func (h handlers) listSearches(ctx context.Context, by identity.Admin, in *workInput) (*searchListOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	runs, err := open(h.p).Searches(ctx, by, works.WorkID{UUID: u})
	if err != nil {
		return nil, answer(err)
	}
	out := searchList{Searches: []search{}}
	for _, s := range runs {
		out.Searches = append(out.Searches, searchOf(s))
	}
	return &searchListOutput{Body: out}, nil
}

func (h handlers) listGrabs(ctx context.Context, by identity.Admin, _ *struct{}) (*grabListOutput, error) {
	views, err := open(h.p).List(ctx, by)
	if err != nil {
		return nil, err
	}
	out := grabList{Grabs: []grab{}}
	for _, v := range views {
		out.Grabs = append(out.Grabs, grabOf(v))
	}
	return &grabListOutput{Body: out}, nil
}

func (h handlers) getGrab(ctx context.Context, by identity.Admin, in *grabInput) (*grabOutput, error) {
	u, err := storage.ParseUUID(in.GrabID)
	if err != nil {
		return nil, grabNotFound
	}
	v, err := open(h.p).Get(ctx, by, grabs.GrabID{UUID: u})
	if err != nil {
		return nil, answer(err)
	}
	return &grabOutput{Body: grabOf(v)}, nil
}
