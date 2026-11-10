// Package grabsapi is the search and Grab operations: start a Work's
// search, list its search runs and the Releases a run saw, list an
// Indexer's Grabs, list and read Grabs with their live progress, cancel
// one, search a Work by hand and grab from what it lists, and the
// Blocklist.
package grabsapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
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
	workNotFound    = api.NewCode(http.StatusNotFound, "work_not_found", "No Work has that id")
	nothingWanted   = api.NewCode(http.StatusConflict, "nothing_wanted", "The Work has no wanted cell to search for")
	noIndexer       = api.NewCode(http.StatusConflict, "no_indexer", "No Indexer is set up")
	noRoot          = api.NewCode(http.StatusConflict, "no_root", "The Work's Content Type has no library root")
	rootUnreachable = api.NewCode(http.StatusConflict, "root_unreachable", "The Work's library root is on a Node that takes no new files now")
	notSearchable   = api.NewCode(http.StatusConflict, "not_searchable", "The Work's Content Type can't be searched")
	grabNotFound    = api.NewCode(http.StatusNotFound, "grab_not_found", "No Grab has that id")
	searchNotFound  = api.NewCode(http.StatusNotFound, "search_not_found", "No search run has that id")
	indexerNotFound = api.NewCode(http.StatusNotFound, "indexer_not_found", "No Indexer has that id")

	grabNotSendable     = api.NewCode(http.StatusConflict, "grab_not_sendable", "This Grab can't be sent to another client")
	grabTorrentInClient = api.NewCode(http.StatusConflict, "grab_torrent_in_client", "This client's Node already has the torrent")
	clientNotFound      = api.NewCode(http.StatusNotFound, "download_client_not_found", "No Download Client has this id")
	clientRefused       = api.NewCode(http.StatusUnprocessableEntity, "download_client_field_refused", "SlipStream can't use a value in this client's settings")
	nodeNotDownloading  = api.NewCode(http.StatusConflict, "node_not_downloading", "That Node doesn't hold Downloading")

	reconciling          = api.NewCode(http.StatusConflict, "reconciling", "SlipStream is Reconciling after a restore")
	noManualSearch       = api.NewCode(http.StatusNotFound, "no_manual_search", "The Work has no Manual search")
	manualSearchReplaced = api.NewCode(http.StatusConflict, "manual_search_replaced", "Someone searched this Work again, so this list is gone. Open Manual search again to see the new one.")
	queryNeedsOneItem    = api.NewCode(http.StatusUnprocessableEntity, "query_needs_one_item", "A typed query searches one Item. Open Manual search on one Item to type a query.")
	grabBusy             = api.NewCode(http.StatusConflict, "grab_busy", "This Grab is placing its file in the library. It can't be canceled until that finishes.")
	grabFinished         = api.NewCode(http.StatusConflict, "grab_finished", "This Grab already finished, so there's nothing to cancel")
	// notBuilt is an operation of S32 (Manual search and grab) whose domain
	// body a later change of that build writes. It goes with
	// grabs.ErrNotBuilt.
	notBuilt = api.NewCode(http.StatusNotImplemented, "not_built", "This operation isn't built yet")
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
	case errors.Is(err, grabs.ErrNoSearch):
		return searchNotFound
	case errors.Is(err, indexers.ErrNoIndexer):
		return indexerNotFound
	case errors.Is(err, grabs.ErrNoManualSearch):
		return noManualSearch
	case errors.Is(err, grabs.ErrQueryNeedsOneItem):
		return queryNeedsOneItem
	case errors.Is(err, grabs.ErrNotBuilt):
		return notBuilt
	}
	if e, ok := errors.AsType[*grabs.RootUnreachableError](err); ok {
		return rootUnreachable.Because(e.Why)
	}
	return err
}

type workInput struct {
	WorkID string `path:"workId" format:"uuid"`
}

type searchWorkInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Body   *searchScope
}

type searchScope struct {
	LevelIDs []string `json:"levelIds" format:"uuid" maxItems:"20000" nullable:"false" doc:"Each level stands for every Item under it"`
	ItemIDs  []string `json:"itemIds" format:"uuid" maxItems:"20000" nullable:"false"`
}

type searchStarted struct {
	SearchIDs []string `json:"searchIds" nullable:"false" doc:"One run per Item in the scope with a wanted cell"`
}

type searchStartedOutput struct{ Body searchStarted }

type reason struct {
	Code      string `json:"code" enum:"wrong_work,wrong_item,covers_more,wrong_title,cut_not_wanted,quality_not_in_target,filter,size_below,size_above,size_not_checked,score_below_minimum,not_an_upgrade,repack_other_group,blocklisted" doc:"Why the release was refused, or a note on it. wrong_item is a release whose title names another Item or no Item. covers_more is a release that holds more than one Item, such as a season pack. SlipStream grabs Releases that hold more than one Item once S33 (Import pipeline) can import them. wrong_title is a release whose title or year doesn't match the Work's. blocklisted is a release whose info hash is on the Blocklist."`
	TargetID  string `json:"targetId,omitempty" format:"uuid" doc:"The Target whose verdict it is, absent for wrong_work, wrong_item, covers_more and wrong_title"`
	Attribute string `json:"attribute,omitempty" enum:"hdr,video_codec,audio_codec,channels,language" doc:"The attribute a failed filter rules on"`
	Value     string `json:"value,omitempty" doc:"The value a filter failed on, the quality a Target doesn't accept, the cut the cell's Edition doesn't take, the score under the minimum, what a wrong_item or covers_more title names, such as S01E02, or the title and year a wrong_title release gives, such as Heat (1986)"`
	Soft      bool   `json:"soft" doc:"Whether the reason keeps only an automatic search from grabbing the release. A Manual search can grab past it. False for a hard refusal and for a note."`
}

func reasonOf(r grabs.Reason) reason {
	w := reason{Code: string(r.Code), Attribute: string(r.Attribute), Value: r.Value, Soft: r.Code.Soft()}
	if !r.Target.IsZero() {
		w.TargetID = r.Target.String()
	}
	return w
}

func reasonsOf(rs []grabs.Reason) []reason {
	out := []reason{}
	for _, r := range rs {
		out = append(out, reasonOf(r))
	}
	return out
}

type searchIndexer struct {
	IndexerID *string `json:"indexerId" format:"uuid" doc:"Null once the Indexer is deleted"`
	Name      string  `json:"name" doc:"The Indexer's name when the run started"`
	Outcome   *string `json:"outcome" enum:"found,no_releases,limited,failing,stopped,waiting,late,limit_reached,no_categories,removed,rss_only" doc:"Null while its search runs"`
	Releases  int     `json:"releases" minimum:"0" doc:"The Releases it found"`
	Sentence  string  `json:"sentence" doc:"The outcome in words, such as Limit reached until 14:05"`
}

type search struct {
	ID         string          `json:"id" format:"uuid"`
	ItemID     string          `json:"itemId" format:"uuid"`
	StartedAt  time.Time       `json:"startedAt"`
	Indexers   []searchIndexer `json:"indexers" nullable:"false" doc:"Each Indexer the run asked or skipped, by priority"`
	FinishedAt *time.Time      `json:"finishedAt" doc:"Null while the run searches"`
	Outcome    *string         `json:"outcome" enum:"grabbed,nothing_acceptable,listed" doc:"Null while the run searches. listed is a Manual search's run, which takes nothing."`
	Candidates int             `json:"candidates" minimum:"0" doc:"The Releases the Indexers found, each Release listed twice counted once"`
	GrabID     *string         `json:"grabId" format:"uuid" doc:"The run's Grab: the decision's, or the rerun's once a dropped Grab reran it; null when none"`
	Notes      []reason        `json:"notes" nullable:"false" doc:"The grabbed Release's notes, such as size_not_checked, for each Target it was grabbed for"`
}

type lostAt struct {
	Step     string  `json:"step" enum:"want,tier,revision,score,priority,freeleech,seeders,size,title" doc:"The first step of the ranking where the reference Release ranked above this one"`
	TargetID *string `json:"targetId" format:"uuid" doc:"The Target the tier, revision and score steps compare at; null at the other steps and once the Target is deleted"`
	Own      string  `json:"own" doc:"This Release's value at the step: a Target's name for want, a quality's name for tier, such as v2 or v1 REPACK for revision, a score, an Indexer priority, a download factor from 0 to 1, a seeder count, bytes, or the title"`
	Ref      string  `json:"ref" doc:"The reference Release's value at the step, written as own is. The reference is the grabbed Release, or the first a search could take when the run grabbed none."`
}

type targetScore struct {
	TargetID string `json:"targetId" format:"uuid"`
	Score    int    `json:"score"`
}

type listedRelease struct {
	Position int                  `json:"position" minimum:"0" doc:"Its place in the order the Indexers listed the run's Releases"`
	Title    string               `json:"title"`
	Words    string               `json:"words" doc:"What SlipStream reads from the title, in words, such as WEBDL-1080p · Dolby Vision · DD+ 5.1 · FLUX · 6.2 GB"`
	Indexer  *string              `json:"indexerId" format:"uuid" doc:"Null once the Indexer is deleted"`
	Name     string               `json:"indexerName" doc:"The Indexer's name when the run started"`
	Size     int64                `json:"size" minimum:"0" doc:"Bytes"`
	Seeders  int                  `json:"seeders" minimum:"0"`
	Outcome  string               `json:"outcome" enum:"grabbed,not_taken,outranked,kept,refused" doc:"grabbed is the run's Grab. not_taken ranked above it, and its Indexer stopped, reached its limit or dropped it. outranked ranked below it. kept ranked, and the run grabbed nothing. refused has a refusal of its own or is grabbable at no cell."`
	Rank     *int                 `json:"rank" minimum:"1" doc:"Its place among the Releases grabbable at some cell, from 1; null for one grabbable at none"`
	CopyOf   *int                 `json:"copyOf" minimum:"0" doc:"The position of the Release this is a copy of, listed by an Indexer of no better priority; null when it is no copy"`
	Covers   []string             `json:"covers" format:"uuid" nullable:"false" doc:"The Items of the Work it holds"`
	Scores   []targetScore        `json:"scores" nullable:"false" doc:"Its score at each Target it was judged at, in Profile order"`
	Formats  []string             `json:"formatIds" format:"uuid" nullable:"false" doc:"The Custom Formats it matched of those the Work's Targets score, as the run saw them"`
	Reasons  []wordedReason       `json:"reasons" nullable:"false" doc:"Its own refusals, then each cell's failures and notes, in Profile order"`
	LostAt   api.Nullable[lostAt] `json:"lostAt" doc:"Where an outranked or kept Release lost to the reference Release; null for the others and for one that ties it at every step"`
}

type releaseList struct {
	Releases []listedRelease `json:"releases" nullable:"false" doc:"Ranked ones first, in rank order, then the rest in the order they were listed. Empty once a later run of the Item finished."`
}

type releaseListOutput struct{ Body releaseList }

type searchInput struct {
	SearchID string `path:"searchId" format:"uuid"`
}

func releaseOf(l grabs.Listed) listedRelease {
	out := listedRelease{
		Position: l.Position, Title: l.Title, Words: l.Words, Indexer: idOrNil(l.Indexer.UUID), Name: l.IndexerName, Size: l.Size,
		Seeders: l.Seeders, Outcome: string(l.Outcome), Covers: []string{}, Scores: []targetScore{}, Formats: []string{}, Reasons: WordedOf(l.Reasons),
	}
	if l.Rank > 0 {
		out.Rank = new(l.Rank)
	}
	if l.CopyOf >= 0 {
		out.CopyOf = new(l.CopyOf)
	}
	for _, it := range l.Covers {
		out.Covers = append(out.Covers, it.String())
	}
	for _, s := range l.Scores {
		out.Scores = append(out.Scores, targetScore{TargetID: s.Target.String(), Score: s.Score})
	}
	for _, f := range l.Formats {
		out.Formats = append(out.Formats, f.String())
	}
	if lost := l.LostAt; lost != nil {
		out.LostAt = api.Some(lostAt{Step: string(lost.Step), TargetID: idOrNil(lost.Target.UUID), Own: lost.Own, Ref: lost.Ref})
	}
	return out
}

type searchList struct {
	Searches []search `json:"searches" nullable:"false" doc:"Newest first"`
}

type searchListOutput struct{ Body searchList }

func searchOf(s grabs.Search) search {
	out := search{
		ID: s.ID.String(), ItemID: s.ItemID.String(), StartedAt: s.Started, Indexers: []searchIndexer{}, Candidates: s.Candidates,
		Notes: reasonsOf(s.Notes),
	}
	for _, l := range s.Indexers {
		w := searchIndexer{IndexerID: idOrNil(l.Indexer.UUID), Name: l.Name, Sentence: l.Line}
		if l.Outcome != nil {
			w.Outcome, w.Releases = new(string(l.Outcome.Kind)), l.Outcome.Releases
		}
		out.Indexers = append(out.Indexers, w)
	}
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

func idOrNil(id storage.UUID) *string {
	if id.IsZero() {
		return nil
	}
	return new(id.String())
}

type failure struct {
	Stage    string `json:"stage" enum:"fetching,downloading,probing,transferring,importing"`
	Code     string `json:"code" enum:"indexer_unavailable,indexer_removed,indexer_stopped,home_offline,torrent_unreachable,magnet_invalid,no_metadata,no_peer_source,torrent_invalid,infohash_mismatch,torrent_v2_only,download_failed,removed,no_video_file,probe_failed,fills_nothing,name_taken,name_too_long,transfer_failed,import_failed,node_written_off,root_changed,canceled,blocklisted,release_fault"`
	Message  string `json:"message"`
	NeedsYou bool   `json:"needsYou" doc:"It goes under Needs you. False for a cancel and for a Manual Grab whose Release the Blocklist held after its fetch, which its sheet already said."`
}

type entered struct {
	Stage string    `json:"stage" enum:"fetching,downloading,probing,transferring,importing,imported,failed"`
	At    time.Time `json:"at"`
}

type progress struct {
	Done  int64 `json:"done" minimum:"0" doc:"Bytes downloaded, or received and verified while transferring"`
	Total int64 `json:"total" minimum:"0" doc:"Bytes wanted"`
}

type metadataSearch struct {
	Peers  int       `json:"peers" minimum:"0" doc:"Peers connected at the Downloading Node's last response"`
	EndsAt time.Time `json:"endsAt" doc:"When the Grab fails unless a peer has sent the metadata"`
}

type fetchWait struct {
	Sentence string     `json:"sentence" doc:"Why the fetch waits, such as Waiting to fetch the .torrent from TorrentLeech. SlipStream retries at 14:20."`
	Until    *time.Time `json:"until" doc:"When SlipStream retries; null while it waits for the Indexer's home Node"`
}

type grabClient struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
	Kind string `json:"kind" enum:"qbittorrent,transmission,deluge,rtorrent"`
}

type clientWait struct {
	NodeID string                   `json:"nodeId" format:"uuid" doc:"The client's Node"`
	Client api.Nullable[grabClient] `json:"client" doc:"The external client; null for the Node's built-in client"`
	Off    bool                     `json:"off" doc:"The Grab is pinned to this client and the client can't take it: it is off or deleted, or its Node doesn't hold Downloading. False while the client's Node is offline"`
}

type grab struct {
	ID           string                   `json:"id" format:"uuid"`
	WorkID       string                   `json:"workId" format:"uuid"`
	ItemID       string                   `json:"itemId" format:"uuid"`
	Edition      string                   `json:"edition" doc:"The Edition it was grabbed to fill: any under any cut, else a cut key"`
	TargetIDs    []string                 `json:"targetIds" nullable:"false" doc:"The Targets it was grabbed for, in Profile order"`
	ReleaseTitle string                   `json:"releaseTitle"`
	Quality      string                   `json:"quality,omitempty" enum:"sdtv,dvd,webrip-480p,hdtv-720p,webrip-720p,webdl-720p,bluray-720p,hdtv-1080p,webrip-1080p,webdl-1080p,bluray-1080p,remux-1080p,hdtv-2160p,webrip-2160p,webdl-2160p,bluray-2160p,remux-2160p,cam" doc:"The quality the release title names, absent when it names none"`
	ReleaseSize  int64                    `json:"releaseSize" minimum:"0"`
	IndexerID    *string                  `json:"indexerId" format:"uuid" doc:"The Indexer it was grabbed from, null once the Indexer is deleted"`
	TrackerID    *string                  `json:"trackerId" format:"uuid" doc:"The Tracker its Release reached, null for a Grab older than Trackers"`
	NodeID       string                   `json:"nodeId" format:"uuid" doc:"The Node that downloads it: the Indexer's home Node until the fetch, then the Downloading Node the fetch handed the .torrent to"`
	FetchNodeID  *string                  `json:"fetchNodeId" format:"uuid" doc:"The Node that fetched the .torrent: the Indexer's Searching Node, or the Downloading Node that got a magnet link's metadata from peers. Null until one did"`
	TorrentID    *string                  `json:"torrentId" format:"uuid" doc:"The Torrent it downloads, null until the .torrent is fetched"`
	Client       api.Nullable[grabClient] `json:"downloadClient" doc:"Its Torrent's external Download Client, null for its Node's built-in client or until the .torrent is fetched"`
	InfoHash     *string                  `json:"infoHash" doc:"The v1 info hash in hex, null until the .torrent is fetched"`
	Stage        string                   `json:"stage" enum:"fetching,downloading,probing,transferring,importing,imported,failed"`
	Failure      *failure                 `json:"failure,omitempty" doc:"Why it failed, absent unless it did"`
	History      []entered                `json:"history" nullable:"false" doc:"Each stage it entered, when it last entered it, oldest first"`
	Progress     *progress                `json:"progress,omitempty" doc:"The download's or the Transfer's progress, absent unless it is downloading and its Node reported it, or transferring"`
	Metadata     *metadataSearch          `json:"metadata,omitempty" doc:"The search for the magnet link's metadata, absent unless it is fetching from a magnet link and has asked its Downloading Node"`
	FetchWait    *fetchWait               `json:"fetchWait,omitempty" doc:"Why fetching the .torrent waits on its Indexer, absent unless it does"`
	WaitsFor     *clientWait              `json:"waitsFor,omitempty" doc:"The client a fetching Grab goes to, absent unless it waits for that client: its Node to be back online, or a pinned client that can't take it"`
	CreatedAt    time.Time                `json:"createdAt"`
	Source       string                   `json:"source" enum:"search,manual" doc:"How it was chosen: by a search's decision, or by an Admin in a Manual search"`
	GrabbedBy    *string                  `json:"grabbedBy" doc:"A name, or the API key; null for a search's Grab"`
	ManualSearch *string                  `json:"manualSearchId" format:"uuid" doc:"The Manual search it came from; null for a search's Grab and once a later Manual search of the Work replaced it"`
	Overrides    []string                 `json:"overrides" nullable:"false" doc:"Each choice the Admin made past the decision, as a sentence"`
	Actions      grabActions              `json:"actions"`
}

type grabActions struct {
	Cancel        bool `json:"cancel" doc:"cancelGrab can cancel it"`
	Blocklistable bool `json:"blocklistable" doc:"A cancel can also put its Release on the Blocklist: its info hash is known"`
	SendTo        bool `json:"sendTo" doc:"sendGrabTo can send it to another client"`
}

func grabOf(v grabs.View) grab {
	out := grab{
		ID: v.ID.String(), WorkID: v.WorkID.String(), ItemID: v.ItemID.String(), Edition: string(v.Edition), TargetIDs: []string{},
		ReleaseTitle: v.Release.Title, ReleaseSize: v.Release.Size, IndexerID: idOrNil(v.Ticket.Indexer.UUID),
		TrackerID: idOrNil(v.Ticket.Tracker.UUID), NodeID: v.Node.String(), Stage: string(v.Stage), History: []entered{}, CreatedAt: v.Created,
		Source: string(v.Source), ManualSearch: idOrNil(v.Manual.UUID), Overrides: append([]string{}, v.Overrides...),
		Actions: grabActions{Cancel: v.Actions.Cancel, Blocklistable: v.Actions.Blocklist, SendTo: v.Actions.SendTo},
	}
	if v.By != nil {
		out.GrabbedBy = new(v.By.Words())
	}
	if !v.FetchNode.IsZero() {
		id := v.FetchNode.String()
		out.FetchNodeID = &id
	}
	for _, t := range v.Targets {
		out.TargetIDs = append(out.TargetIDs, t.String())
	}
	if q, ok := release.QualityIn(v.Release.Title).Get(); ok {
		out.Quality = q.String()
	}
	if v.Torrent != nil {
		out.TorrentID = new(v.Torrent.ID.String())
		out.InfoHash = new(v.Torrent.InfoHash.String())
		if c := v.Torrent.Client; c != nil {
			out.Client = api.Some(grabClient{ID: c.ID.String(), Name: c.Name, Kind: string(c.Kind)})
		}
	}
	if v.Failed != nil {
		out.Failure = &failure{Stage: string(v.Failed.Stage), Code: string(v.Failed.Code), Message: v.Failed.Message, NeedsYou: v.NeedsYou()}
	}
	for _, e := range v.History {
		out.History = append(out.History, entered{Stage: string(e.Stage), At: e.At})
	}
	if v.Progress != nil {
		out.Progress = &progress{Done: v.Progress.Done, Total: v.Progress.Total}
	}
	if v.Metadata != nil {
		out.Metadata = &metadataSearch{Peers: v.Metadata.Peers, EndsAt: v.Metadata.EndsAt}
	}
	if w := v.FetchWait; w != nil {
		out.FetchWait = &fetchWait{Sentence: w.Sentence}
		if !w.Outcome.Until.IsZero() {
			out.FetchWait.Until = &w.Outcome.Until
		}
	}
	if w := v.WaitsFor; w != nil {
		out.WaitsFor = &clientWait{NodeID: w.Node.String(), Off: w.Off}
		if c := w.Client; c != nil {
			out.WaitsFor.Client = api.Some(grabClient{ID: c.ID.String(), Name: c.Name, Kind: string(c.Kind)})
		}
	}
	return out
}

type grabList struct {
	Grabs []grab `json:"grabs" nullable:"false" doc:"Newest first"`
}

type grabListOutput struct{ Body grabList }

const indexerGrabsPage = 50

type indexerGrabList struct {
	Grabs      []grab `json:"grabs" nullable:"false" doc:"Newest first"`
	NextOffset *int   `json:"nextOffset" doc:"The offset of the next page; null on the last"`
}

type indexerGrabsInput struct {
	ID     string `path:"id" format:"uuid"`
	Offset int    `query:"offset" minimum:"0" doc:"How many of the newest to skip"`
}

type indexerGrabsOutput struct{ Body indexerGrabList }

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
		Summary: "Search for each wanted Item of a Work, or of some of its levels and Items, and grab the best release",
		Status:  http.StatusAccepted,
		Errors:  []*api.Code{workNotFound, nothingWanted, noIndexer, noRoot, rootUnreachable, notSearchable},
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
		ID:      "listSearchReleases",
		Method:  http.MethodGet,
		Path:    "/searches/{searchId}/releases",
		Summary: "List the Releases a search run saw, with what it did with each and why",
		Status:  http.StatusOK,
		Errors:  []*api.Code{searchNotFound},
	}, h.listSearchReleases)
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
	api.Register(o, api.Admin, api.Op{
		ID:      "sendGrabTo",
		Method:  http.MethodPost,
		Path:    "/grabs/{grabId}/send-to",
		Summary: "Send a Grab to another Download Client: a downloading Grab's torrent, or a Grab whose fetch waits for its client's Node, which then fetches for the new client",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{grabNotFound, grabNotSendable, grabTorrentInClient, clientNotFound, clientRefused, nodeNotDownloading},
	}, h.sendTo)
	api.Register(o, api.Admin, api.Op{
		ID:      "listIndexerGrabs",
		Method:  http.MethodGet,
		Path:    "/indexers/{id}/grabs",
		Summary: "List the Grabs taken from an Indexer, 50 at a time",
		Status:  http.StatusOK,
		Errors:  []*api.Code{indexerNotFound},
	}, h.listIndexerGrabs)
	api.Register(o, api.Admin, api.Op{
		ID:      "cancelGrab",
		Method:  http.MethodPost,
		Path:    "/grabs/{grabId}/cancel",
		Summary: "Cancel a Grab that is fetching, downloading or transferring. Canceling a canceled Grab changes nothing",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{grabNotFound, grabBusy, grabFinished, reconciling},
	}, h.cancelGrab)
	api.Register(o, api.Admin, api.Op{
		ID:      "startManualSearch",
		Method:  http.MethodPost,
		Path:    "/works/{workId}/manual-search",
		Summary: "Start a Manual search of a Work, or of some of its levels and Items or one cell. 200 answers the Work's Manual search when it already has this scope and query",
		Status:  http.StatusAccepted,
		Also:    []int{http.StatusOK},
		Errors:  []*api.Code{workNotFound, reconciling, notSearchable, noIndexer, noRoot, queryNeedsOneItem},
	}, h.startManualSearch)
	api.Register(o, api.Admin, api.Op{
		ID:      "getManualSearch",
		Method:  http.MethodGet,
		Path:    "/works/{workId}/manual-search",
		Summary: "Get a Work's Manual search with its plan worked out around the pins",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, noManualSearch},
	}, h.getManualSearch)
	api.Register(o, api.Admin, api.Op{
		ID:      "listManualSearchReleases",
		Method:  http.MethodGet,
		Path:    "/works/{workId}/manual-search/releases",
		Summary: "List every Release of a Work's Manual search with what grabbing it alone does",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, noManualSearch},
	}, h.listManualSearchReleases)
	api.Register(o, api.Admin, api.Op{
		ID:      "grabFromManualSearch",
		Method:  http.MethodPost,
		Path:    "/works/{workId}/manual-search/grabs",
		Summary: "Grab the picked Releases of a Work's Manual search in one go, each judged again and refused alone",
		Status:  http.StatusOK,
		Errors:  []*api.Code{workNotFound, manualSearchReplaced, reconciling, notBuilt},
	}, h.grabFromManualSearch)
	api.Register(o, api.Admin, api.Op{
		ID:      "listBlocklist",
		Method:  http.MethodGet,
		Path:    "/blocklist",
		Summary: "List the Blocklist, 50 entries at a time, newest first",
		Status:  http.StatusOK,
	}, h.listBlocklist)
	api.Register(o, api.Admin, api.Op{
		ID:      "removeFromBlocklist",
		Method:  http.MethodPost,
		Path:    "/blocklist/remove",
		Summary: "Remove Releases from the Blocklist by info hash. Removing one that isn't listed changes nothing",
		Status:  http.StatusNoContent,
	}, h.removeFromBlocklist)
}

type handlers struct{ p Pipeline }

func (h handlers) searchWork(ctx context.Context, by identity.Admin, in *searchWorkInput) (*searchStartedOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	scope, err := scopeOf(in.Body)
	if err != nil {
		return nil, err
	}
	ids, err := open(h.p).SearchWork(ctx, by, works.WorkID{UUID: u}, scope)
	if e, ok := errors.AsType[*works.NotInWorkError](err); ok {
		return nil, notInWork(e)
	}
	if err != nil {
		return nil, answer(err)
	}
	out := searchStarted{SearchIDs: []string{}}
	for _, id := range ids {
		out.SearchIDs = append(out.SearchIDs, id.String())
	}
	return &searchStartedOutput{Body: out}, nil
}

// scopeOf is the search scope body names, the whole Work when it is left
// out.
func scopeOf(body *searchScope) (grabs.Scope, error) {
	if body == nil {
		return grabs.Scope{}, nil
	}
	if len(body.LevelIDs) == 0 && len(body.ItemIDs) == 0 {
		return grabs.Scope{}, api.Invalid.On("levelIds", "Send at least one level or Item id, or leave the body out to search the whole Work.")
	}
	part, err := withinOf(body.LevelIDs, body.ItemIDs)
	if err != nil {
		return grabs.Scope{}, err
	}
	return grabs.Within(part), nil
}

func withinOf(levels, items []string) (works.Within, error) {
	var part works.Within
	for _, s := range levels {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return works.Within{}, api.Invalid
		}
		part.Levels = append(part.Levels, works.LevelID{UUID: u})
	}
	for _, s := range items {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return works.Within{}, api.Invalid
		}
		part.Items = append(part.Items, works.ItemID{UUID: u})
	}
	return part, nil
}

// notInWork is e as the field it names.
func notInWork(e *works.NotInWorkError) error {
	field, noun := "itemIds", "Item"
	if e.Level {
		field, noun = "levelIds", "level"
	}
	return api.Invalid.On(fmt.Sprintf("%s[%d]", field, e.Index), "The Work has no "+noun+" with that id.")
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

func (h handlers) listSearchReleases(ctx context.Context, by identity.Admin, in *searchInput) (*releaseListOutput, error) {
	u, err := storage.ParseUUID(in.SearchID)
	if err != nil {
		return nil, searchNotFound
	}
	listed, err := open(h.p).Releases(ctx, by, grabs.SearchID{UUID: u})
	if err != nil {
		return nil, answer(err)
	}
	out := releaseList{Releases: []listedRelease{}}
	for _, l := range listed {
		out.Releases = append(out.Releases, releaseOf(l))
	}
	return &releaseListOutput{Body: out}, nil
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

func (h handlers) listIndexerGrabs(ctx context.Context, by identity.Admin, in *indexerGrabsInput) (*indexerGrabsOutput, error) {
	u, err := storage.ParseUUID(in.ID)
	if err != nil {
		return nil, indexerNotFound
	}
	views, err := open(h.p).OfIndexer(ctx, by, indexers.ID{UUID: u}, indexers.Page{Offset: in.Offset, Limit: indexerGrabsPage + 1})
	if err != nil {
		return nil, answer(err)
	}
	out := indexerGrabList{Grabs: []grab{}}
	if len(views) > indexerGrabsPage {
		views, out.NextOffset = views[:indexerGrabsPage], new(in.Offset+indexerGrabsPage)
	}
	for _, v := range views {
		out.Grabs = append(out.Grabs, grabOf(v))
	}
	return &indexerGrabsOutput{Body: out}, nil
}

type sendToInput struct {
	GrabID string `path:"grabId" format:"uuid"`
	Body   struct {
		ClientID  *string `json:"clientId,omitempty" format:"uuid" doc:"An external client; send this or builtInOn"`
		BuiltInOn *string `json:"builtInOn,omitempty" format:"uuid" doc:"A downloading Node, for its built-in client; send this or clientId"`
	}
}

func (h handlers) sendTo(ctx context.Context, by identity.Admin, in *sendToInput) (*struct{}, error) {
	id, err := storage.ParseUUID(in.GrabID)
	if err != nil {
		return nil, grabNotFound
	}
	c, err := choiceOf(in.Body.ClientID, in.Body.BuiltInOn)
	if err != nil {
		return nil, err
	}
	err = open(h.p).SendTo(ctx, by, grabs.GrabID{UUID: id}, c)
	if err == nil {
		return nil, nil
	}
	var (
		off      *downloadclients.ClientOffError
		inClient *downloadclients.InClientError
	)
	switch {
	case errors.Is(err, grabs.ErrNotSendable):
		return nil, grabNotSendable.Because("Only a download that hasn't finished, or a Grab waiting for its client's Node, can be sent to another client.")
	case errors.As(err, &inClient):
		return nil, grabTorrentInClient.Because(downloadclients.InClient(inClient.Client, inClient.Node))
	case errors.Is(err, downloadclients.ErrRemoving):
		return nil, grabTorrentInClient.Because("That Node is still removing this torrent. Send the Grab there once it's removed.")
	case errors.As(err, &off):
		return nil, clientRefused.On("clientId", off.Name+" is off. Turn it on, then send the Grab to it.")
	case errors.Is(err, downloadclients.ErrNoClient):
		return nil, clientNotFound
	case errors.Is(err, downloadclients.ErrNotDownloading):
		return nil, nodeNotDownloading
	}
	return nil, answer(err)
}

// choiceOf is the client a send-to's body names: exactly one of an
// external client's id or a Node for its built-in client.
func choiceOf(client, builtInOn *string) (downloadclients.Choice, error) {
	switch {
	case (client == nil) == (builtInOn == nil):
		return downloadclients.Choice{}, api.Invalid.Fields(api.FieldError{Location: "body", Message: "Send exactly one of clientId and builtInOn."})
	case client != nil:
		u, err := storage.ParseUUID(*client)
		if err != nil {
			return downloadclients.Choice{}, clientNotFound
		}
		return downloadclients.Choice{Client: &downloadclients.ClientID{UUID: u}}, nil
	}
	u, err := storage.ParseUUID(*builtInOn)
	if err != nil {
		return downloadclients.Choice{}, nodeNotDownloading
	}
	return downloadclients.Choice{BuiltInOn: role.NodeID{UUID: u}}, nil
}
