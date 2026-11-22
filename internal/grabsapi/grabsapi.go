// Package grabsapi is the search and Grab operations: start a Work's
// search, list its search runs and the Releases a run saw, list an
// Indexer's Grabs, list and read Grabs with their live progress, cancel
// one, search a Work by hand and grab from what it lists, and the
// Blocklist.
package grabsapi

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type Pipeline interface {
	// Grabs is the grab pipeline, false unless the Node is a Controller.
	Grabs() (*grabs.Grabs, bool)
	// Minter signs the artwork URLs the answers carry, false unless the
	// Node is a Controller.
	Minter() (artwork.Minter, bool)
}

func minter(p Pipeline) artwork.Minter {
	m, ok := p.Minter()
	if !ok {
		panic("grabsapi: an operation ran on a Node that is no Controller")
	}
	return m
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
	nothingWanted   = api.NewCode(http.StatusConflict, "nothing_wanted", "Nothing to search for. Every monitored, released file meets its Target or is downloading.")
	noIndexer       = api.NewCode(http.StatusConflict, "no_indexer", "No Indexer is set up")
	noRoot          = api.NewCode(http.StatusConflict, "no_root", "The Work's Content Type has no library root")
	rootUnreachable = api.NewCode(http.StatusConflict, "root_unreachable", "SlipStream can't put new files in this library root now")
	notSearchable   = api.NewCode(http.StatusConflict, "not_searchable", "The Work's Content Type can't be searched")
	grabNotFound    = api.NewCode(http.StatusNotFound, "grab_not_found", "No Grab has that id")
	searchNotFound  = api.NewCode(http.StatusNotFound, "search_not_found", "No search run has that id")
	indexerNotFound = api.NewCode(http.StatusNotFound, "indexer_not_found", "No Indexer has that id")

	grabNotSendable     = api.NewCode(http.StatusConflict, "grab_not_sendable", "This Grab can't be sent to another client")
	grabTorrentInClient = api.NewCode(http.StatusConflict, "grab_torrent_in_client", "This client's Node already has the torrent")
	clientNotFound      = api.NewCode(http.StatusNotFound, "download_client_not_found", "No Download Client has this id")
	clientRefused       = api.NewCode(http.StatusUnprocessableEntity, "download_client_field_refused", "SlipStream can't use a value in this client's settings")
	nodeNotDownloading  = api.NewCode(http.StatusConflict, "node_not_downloading", "That Node doesn't have the Downloading Role")

	noManualSearch       = api.NewCode(http.StatusNotFound, "no_manual_search", "The Work has no Manual search")
	manualSearchReplaced = api.NewCode(http.StatusConflict, "manual_search_replaced", "A newer Manual search replaced this list. Open Manual search again to see it.")
	queryNeedsOneItem    = api.NewCode(http.StatusUnprocessableEntity, "query_needs_one_item", "A typed query can search only one episode. Open Manual search on one episode to type a query.")
	grabBusy             = api.NewCode(http.StatusConflict, "grab_busy", "This Grab is placing its file in the library. It can't be canceled until that finishes.")
	grabFinished         = api.NewCode(http.StatusConflict, "grab_finished", "This Grab already finished, so there's nothing to cancel")
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
	}
	if e, ok := errors.AsType[*grabs.RootUnreachableError](err); ok {
		return rootUnreachable.Because(e.Why)
	}
	return err
}

type workInput struct {
	WorkID string `path:"workId" format:"uuid"`
}

type reason struct {
	Code      string `json:"code" enum:"wrong_work,wrong_item,wrong_title,cut_not_wanted,quality_not_in_target,filter,size_below,size_above,size_not_checked,score_below_minimum,not_an_upgrade,repack_other_group,blocklisted,not_aired,tried" doc:"Why the release was refused, or a note on it. wrong_item is a release whose title names another Item, no Item, or none of the searched Items. wrong_title is a release whose title or year doesn't match the Work's. blocklisted is a release whose info hash is on the Blocklist. not_aired is a pack listing an Item that hasn't aired, its value the Item's code. tried is a release a failed Grab of the Work already tried."`
	TargetID  string `json:"targetId,omitempty" format:"uuid" doc:"The Target whose verdict it is, absent for wrong_work, wrong_item and wrong_title"`
	Attribute string `json:"attribute,omitempty" enum:"hdr,video_codec,audio_codec,channels,language" doc:"The attribute a failed filter rules on"`
	Value     string `json:"value,omitempty" doc:"The value a filter failed on, the quality a Target doesn't accept, the cut the Edition doesn't include, the score under the minimum, what a wrong_item title names, such as S01E02, or the title and year a wrong_title release gives, such as Heat (1986)"`
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
	GroupKey   *string         `json:"groupKey" doc:"The search group the run searched, to compare and never to parse; null for a Manual search's run"`
	GroupLabel *string         `json:"groupLabel" doc:"The group in words, such as season 3; null for a Manual search's run, and until its Content Type names its groups"`
	Lane       *string         `json:"lane" enum:"now,bulk,backlog" doc:"The lane it was queued in; null for a Manual search's run"`
	Trigger    *string         `json:"trigger" enum:"search,search_all,add,monitor,profile,editions,retarget,release_date,refresh,backlog,failed_grab,rss" doc:"What started it; null for a Manual search's run"`
	ItemIDs    []string        `json:"itemIds" format:"uuid" nullable:"false" doc:"The Items the run searched"`
	StartedAt  time.Time       `json:"startedAt"`
	Indexers   []searchIndexer `json:"indexers" nullable:"false" doc:"Each Indexer the run asked or skipped, by priority"`
	FinishedAt *time.Time      `json:"finishedAt" doc:"Null while the run searches"`
	Outcome    *string         `json:"outcome" enum:"grabbed,waiting,nothing_found,nothing_acceptable,no_indexer,no_root,listed" doc:"Null while the run searches. waiting is a run that waits out a Target's Hold for a better Release. nothing_found is a run that found no Release, and nothing_acceptable one whose Releases fit no Target. no_indexer is a run that no Indexer responded to, and no_root one whose Work had no reachable root. listed is a Manual search's run, which takes nothing."`
	Candidates int             `json:"candidates" minimum:"0" doc:"The Releases the Indexers found, each Release listed twice counted once"`
	Grabs      int             `json:"grabs" minimum:"0" doc:"The Grabs it made or joined when it finished, still counted once one is pruned"`
	GrabIDs    []string        `json:"grabIds" format:"uuid" nullable:"false" doc:"The Grabs the run made, then those it joined: the decision's, or the rerun's once a dropped Grab reran it"`
	Notes      []reason        `json:"notes" nullable:"false" doc:"The grabbed Release's notes, such as size_not_checked, for each Target it was grabbed for"`
}

type lostAt struct {
	Step     string  `json:"step" enum:"want,fewer_releases,tier,revision,score,chosen_by_another,fewer_unwanted,priority,freeleech,seeders,size,title" doc:"The first step of the ranking where the reference Release ranked above this one"`
	TargetID *string `json:"targetId" format:"uuid" doc:"The Target the tier, revision and score steps compare at; null at the other steps and once the Target is deleted"`
	Own      string  `json:"own" doc:"This Release's value at the step: a Target's name for want, a count of the missing files it has for fewer_releases, a quality's name for tier, such as v2 or v1 REPACK for revision, a score, whether SlipStream already picked it for another missing file for chosen_by_another, a count of the files it has that SlipStream doesn't need for fewer_unwanted, an Indexer priority, a download factor from 0 to 1, a seeder count, bytes, or the title"`
	Ref      string  `json:"ref" doc:"The reference Release's value at the step, written as own is. The reference is the grabbed Release, or the first a search could take when the run grabbed none."`
}

type targetScore struct {
	TargetID string `json:"targetId" format:"uuid"`
	Score    int    `json:"score"`
}

type listedRelease struct {
	Position int                  `json:"position" minimum:"0" doc:"Its place in the order the Indexers listed the run's Releases"`
	Title    string               `json:"title"`
	Facts    releaseFacts         `json:"facts"`
	Indexer  *string              `json:"indexerId" format:"uuid" doc:"Null once the Indexer is deleted"`
	Name     string               `json:"indexerName" doc:"The Indexer's name when the run started"`
	Size     int64                `json:"size" minimum:"0" doc:"Bytes"`
	Seeders  int                  `json:"seeders" minimum:"0"`
	Outcome  string               `json:"outcome" enum:"grabbed,not_taken,outranked,kept,refused,waiting" doc:"grabbed is the run's Grab. not_taken ranked above it, but its Indexer had stopped, reached its limit or no longer listed it. outranked ranked below it. kept ranked, but the run grabbed nothing. refused has a refusal of its own, or no Target accepts it. waiting is a candidate SlipStream keeps during a Target's Hold, until it grabs the best one."`
	Rank     *int                 `json:"rank" minimum:"1" doc:"Its place among the releases some Target accepts, from 1; null for one no Target accepts"`
	CopyOf   *int                 `json:"copyOf" minimum:"0" doc:"The position of the Release this is a copy of, listed by an Indexer of no better priority; null when it is no copy"`
	Covers   []string             `json:"covers" format:"uuid" nullable:"false" doc:"The Items of the Work it holds"`
	Scores   []targetScore        `json:"scores" nullable:"false" doc:"Its score at each Target it was judged at, in Profile order"`
	Formats  []string             `json:"formatIds" format:"uuid" nullable:"false" doc:"The Custom Formats it matched of those the Work's Targets score, as the run saw them"`
	Reasons  []wordedReason       `json:"reasons" nullable:"false" doc:"Its own refusals, then the failures and notes at each Target, in Profile order"`
	LostAt   api.Nullable[lostAt] `json:"lostAt" doc:"Where an outranked or kept Release lost to the reference Release; null for the others and for one that ties it at every step"`
}

type releaseList struct {
	Releases []listedRelease `json:"releases" nullable:"false" doc:"Ranked ones first, in rank order, then the rest in the order they were listed"`
}

type releaseListOutput struct{ Body releaseList }

type searchInput struct {
	SearchID string `path:"searchId" format:"uuid"`
}

func releaseOf(l grabs.Listed) listedRelease {
	out := listedRelease{
		Position: l.Position, Title: l.Title, Facts: factsOf(l.Summary), Indexer: idOrNil(l.Indexer.UUID), Name: l.IndexerName, Size: l.Size,
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
		ID: s.ID.String(), Lane: text(s.Lane), Trigger: text(s.Trigger), ItemIDs: []string{}, StartedAt: s.Started, Indexers: []searchIndexer{},
		Candidates: s.Candidates, Grabs: s.Took, GrabIDs: []string{}, Notes: reasonsOf(s.Notes),
	}
	if s.Group != "" {
		out.GroupKey = new(string(s.Group))
	}
	for _, it := range s.Items {
		out.ItemIDs = append(out.ItemIDs, it.String())
	}
	for _, g := range s.Grabs {
		out.GrabIDs = append(out.GrabIDs, g.String())
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
	return out
}

// text is s, nil when it is empty.
func text[S ~string](s S) *string {
	if s == "" {
		return nil
	}
	return new(string(s))
}

func idOrNil(id storage.UUID) *string {
	if id.IsZero() {
		return nil
	}
	return new(id.String())
}

type failure struct {
	Stage    string               `json:"stage" enum:"fetching,downloading,importing"`
	Code     string               `json:"code" enum:"indexer_unavailable,indexer_removed,indexer_stopped,home_offline,torrent_unreachable,magnet_invalid,no_metadata,no_peer_source,torrent_invalid,infohash_mismatch,torrent_v2_only,download_failed,marked_failed,removed,client_removed,client_deleted,goal_met,no_video_file,probe_failed,fills_nothing,name_taken,name_too_long,transfer_failed,import_failed,node_written_off,root_changed,canceled,blocklisted,release_fault,client_error,declined,unreadable,restored"`
	Message  string               `json:"message"`
	Cause    string               `json:"cause" doc:"Why the Grab failed, in words"`
	Recovery api.Nullable[string] `json:"recovery" doc:"What to do next, in words; null when there is nothing to do"`
	NeedsYou bool                 `json:"needsYou" doc:"True when Needs you lists the failure. False for a cancel, for a failure SlipStream searches again by itself, and for a Manual Grab whose fetched torrent was on the Blocklist, because the Manual search sheet already showed that."`
	Current  bool                 `json:"current" doc:"No later Grab is downloading any of the same files. Needs you lists a failure only while it is current."`
}

type entered struct {
	Stage string    `json:"stage" enum:"fetching,downloading,importing,imported,failed"`
	At    time.Time `json:"at"`
}

type progress struct {
	Done  int64 `json:"done" minimum:"0" doc:"Bytes downloaded"`
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
	Off    bool                     `json:"off" doc:"The Grab is pinned to this client and the client can't take it: it is off or deleted, or its Node doesn't have the Downloading Role. False while the client's Node is offline"`
}

type grab struct {
	ID           string                   `json:"id" format:"uuid"`
	WorkID       string                   `json:"workId" format:"uuid"`
	Items        []string                 `json:"items" nullable:"false" doc:"The ids of the Items its Release covers in the Work, in tree order"`
	Edition      string                   `json:"edition" doc:"The Edition it was grabbed for: any when the Work takes any cut, else a cut key"`
	TargetIDs    []string                 `json:"targetIds" nullable:"false" doc:"The Targets it was grabbed for, in Profile order"`
	Targets      []grabTarget             `json:"targets" nullable:"false" doc:"Each Target of its Items it downloads for, in plan order"`
	ReleaseTitle string                   `json:"releaseTitle"`
	ReleaseFacts releaseFacts             `json:"releaseFacts"`
	ReleaseSize  int64                    `json:"releaseSize" minimum:"0"`
	IndexerID    *string                  `json:"indexerId" format:"uuid" doc:"The Indexer it was grabbed from, null once the Indexer is deleted"`
	TrackerID    *string                  `json:"trackerId" format:"uuid" doc:"The Tracker its release came from, null for a Grab older than Trackers"`
	NodeID       string                   `json:"nodeId" format:"uuid" doc:"The Node that downloads it: the Indexer's home Node until the fetch, then the Downloading Node the fetch handed the .torrent to"`
	FetchNodeID  *string                  `json:"fetchNodeId" format:"uuid" doc:"The Node that fetched the .torrent: the Indexer's Searching Node, or the Downloading Node that got a magnet link's metadata from peers. Null until one did"`
	TorrentID    *string                  `json:"torrentId" format:"uuid" doc:"The Torrent it downloads, null until the .torrent is fetched"`
	Client       api.Nullable[grabClient] `json:"downloadClient" doc:"Its Torrent's external Download Client, null for its Node's built-in client or until the .torrent is fetched"`
	InfoHash     *string                  `json:"infoHash" doc:"The v1 info hash in hex, null until the .torrent is fetched"`
	Stage        string                   `json:"stage" enum:"fetching,downloading,importing,imported,failed"`
	InFlight     bool                     `json:"inFlight" doc:"It is neither imported nor failed, so it counts in Activity and no other Grab can download the same files"`
	Failure      *failure                 `json:"failure,omitempty" doc:"Why it failed, absent unless it did"`
	Import       api.Nullable[grabImport] `json:"import" doc:"Its download's Import, from importing on; null before"`
	History      []entered                `json:"history" nullable:"false" doc:"Each stage it entered, when it last entered it, oldest first"`
	Progress     *progress                `json:"progress,omitempty" doc:"The download's progress, absent unless it is downloading and its Node reported it"`
	Metadata     *metadataSearch          `json:"metadata,omitempty" doc:"The search for the magnet link's metadata, absent unless it is fetching from a magnet link and has asked its Downloading Node"`
	FetchWait    *fetchWait               `json:"fetchWait,omitempty" doc:"Why fetching the .torrent waits on its Indexer, absent unless it does"`
	WaitsFor     *clientWait              `json:"waitsFor,omitempty" doc:"The client a fetching Grab goes to, absent unless it waits for that client: its Node to be back online, or a pinned client that can't take it"`
	CreatedAt    time.Time                `json:"createdAt"`
	Source       string                   `json:"source" enum:"search,manual,rss" doc:"How it was chosen: by a search's decision, by an Admin in a Manual search, or by RSS sync's decision"`
	SearchID     *string                  `json:"searchId" format:"uuid" doc:"The search run that made it; null for a Manual Grab and once the run is pruned"`
	FoundBy      string                   `json:"foundBy" enum:"search,backlog,rss,failed_grab,manual" doc:"What found its Release: a press of Search, Backlog search, RSS sync, a search again after a failed Grab, or an Admin in a Manual search"`
	RSSIndexer   *string                  `json:"rssIndexerName" doc:"The Indexer whose feed RSS sync found it in; null unless foundBy is rss, and once the Indexer is deleted"`
	Chosen       api.Nullable[chosenWhy]  `json:"chosenBecause" doc:"Why the decision chose its Release; null when the ranking alone chose it"`
	GrabbedBy    *string                  `json:"grabbedBy" doc:"A name, or the API key; null for a search's Grab"`
	ManualSearch *string                  `json:"manualSearchId" format:"uuid" doc:"The Manual search it came from; null for a search's Grab and once a later Manual search of the Work replaced it"`
	Overrides    []string                 `json:"overrides" nullable:"false" doc:"Each choice the Admin made past the decision, as a sentence"`
	Actions      grabActions              `json:"actions"`
}

type chosenWhy struct {
	Kind  string `json:"kind" enum:"fewer_releases" doc:"fewer_releases is a Release with more of the group's Items, so the group needs fewer Releases"`
	Fills int    `json:"fills" minimum:"1" doc:"How many of the group's Items it was chosen for"`
}

type grabImport struct {
	ID      string `json:"id" format:"uuid"`
	Placed  int    `json:"placed" minimum:"0" doc:"Its files placed into the library"`
	Planned int    `json:"planned" minimum:"0" doc:"Its files that are or were sending, placing, placed or waiting"`
	Sending int    `json:"sending" minimum:"0" doc:"Its files a Transfer is sending"`
	Waiting int    `json:"waiting" minimum:"0" doc:"Its files that wait for the Admin"`
	Words   string `json:"words" doc:"Its progress in words, such as Importing 6 of 10 episodes"`
}

func importOf(im *grabs.ImportView) api.Nullable[grabImport] {
	if im == nil {
		return api.Nullable[grabImport]{}
	}
	return api.Some(grabImport{ID: im.ID.String(), Placed: im.Placed, Planned: im.Planned, Sending: im.Sending, Waiting: im.Waiting, Words: im.Words})
}

type grabTarget struct {
	TargetID string               `json:"targetId" format:"uuid"`
	ItemID   string               `json:"itemId" format:"uuid"`
	Forced   bool                 `json:"forced" doc:"An Admin forced it: a file goes in even when it isn't an upgrade"`
	Settled  api.Nullable[string] `json:"settled" enum:"placed,none,declined" doc:"How the Grab finished at this Target: it placed a file, it placed none, or the Admin declined it. Null while the Grab is still downloading for it"`
}

type grabActions struct {
	Cancel        bool `json:"cancel" doc:"cancelGrab can cancel it"`
	Blocklistable bool `json:"blocklistable" doc:"A cancel can also put its Release on the Blocklist: its info hash is known"`
	SendTo        bool `json:"sendTo" doc:"sendGrabTo can send it to another client"`
}

// targetsOf is each Target cells name, once, in their order, and each
// of cells.
func targetsOf(cells []grabs.Cell) (ids []string, out []grabTarget) {
	ids, out = []string{}, []grabTarget{}
	for _, c := range cells {
		if id := c.Key.Target.String(); !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
		t := grabTarget{TargetID: c.Key.Target.String(), ItemID: c.Key.Item.String(), Forced: c.Forced}
		if c.Settled != nil {
			t.Settled = api.Some(string(*c.Settled))
		}
		out = append(out, t)
	}
	return ids, out
}

func failureOf(v grabs.View) *failure {
	words := grabs.WordsOf(v.Failed.Code)
	f := &failure{
		Stage: string(v.Failed.Stage), Code: string(v.Failed.Code), Message: v.Failed.Message, Cause: words.Cause,
		NeedsYou: v.NeedsYou() && !v.SearchingAgain, Current: v.Current,
	}
	if words.Recovery != "" {
		f.Recovery = api.Some(words.Recovery)
	}
	return f
}

func grabOf(v grabs.View) grab {
	out := grab{
		ID: v.ID.String(), WorkID: v.WorkID.String(), Edition: string(v.Edition),
		Items: []string{}, ReleaseTitle: v.Release.Title, ReleaseFacts: factsOf(v.ReleaseSummary), ReleaseSize: v.Release.Size, IndexerID: idOrNil(v.Ticket.Indexer.UUID),
		TrackerID: idOrNil(v.Ticket.Tracker.UUID), NodeID: v.Node.String(), Stage: string(v.Stage), InFlight: v.Stage.InFlight(), History: []entered{}, CreatedAt: v.Created,
		Source: string(v.Source), SearchID: idOrNil(v.Search.UUID), FoundBy: string(v.FoundBy), ManualSearch: idOrNil(v.Manual.UUID), Overrides: append([]string{}, v.Overrides...), Import: importOf(v.Import),
		Actions: grabActions{Cancel: v.Actions.Cancel, Blocklistable: v.Actions.Blocklist, SendTo: v.Actions.SendTo},
	}
	if v.By != nil {
		out.GrabbedBy = new(v.By.Words())
	}
	if v.RSSIndexer != "" {
		out.RSSIndexer = new(v.RSSIndexer)
	}
	if v.ChosenFills > 0 {
		out.Chosen = api.Some(chosenWhy{Kind: "fewer_releases", Fills: v.ChosenFills})
	}
	if !v.FetchNode.IsZero() {
		id := v.FetchNode.String()
		out.FetchNodeID = &id
	}
	for _, item := range v.Items {
		out.Items = append(out.Items, item.String())
	}
	out.TargetIDs, out.Targets = targetsOf(v.Targets)
	if v.Torrent != nil {
		out.TorrentID = new(v.Torrent.ID.String())
		out.InfoHash = new(v.Torrent.InfoHash.String())
		if c := v.Torrent.Client; c != nil {
			out.Client = api.Some(grabClient{ID: c.ID.String(), Name: c.Name, Kind: string(c.Kind)})
		}
	}
	if v.Failed != nil {
		out.Failure = failureOf(v)
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

	api.Register(o, api.Admin, searchWorkOp, h.searchWork)
	api.Register(o, api.Admin, searchSeveralWorksOp, h.searchSeveralWorks)
	api.Register(o, api.Admin, cancelQueuedSearchesOp, h.cancelQueuedSearches)
	api.Register(o, api.Admin, getWorkSearchingOp, h.getWorkSearching)
	api.Register(o, api.Admin, grabWaitingOp, h.grabWaiting)
	api.Register(o, api.Admin, listWantedOp, h.listWanted)
	api.Register(o, api.Admin, getSearchSettingsOp, h.getSearchSettings)
	api.Register(o, api.Admin, updateSearchSettingsOp, h.updateSearchSettings)
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
		Errors:  []*api.Code{grabNotFound, grabNotSendable, grabTorrentInClient, clientNotFound, clientRefused, nodeNotDownloading, api.Reconciling},
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
		Summary: "Cancel a Grab that is fetching, downloading or importing. Canceling a canceled Grab changes nothing",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{grabNotFound, grabBusy, grabFinished, api.Reconciling},
	}, h.cancelGrab)
	api.Register(o, api.Admin, api.Op{
		ID:      "startManualSearch",
		Method:  http.MethodPost,
		Path:    "/works/{workId}/manual-search",
		Summary: "Start a Manual search of a Work, or of some of its levels and Items, or of one Item at one Target. Responds 200 with the existing Manual search when it already has this scope and query",
		Status:  http.StatusAccepted,
		Also:    []int{http.StatusOK},
		Errors:  []*api.Code{workNotFound, api.Reconciling, notSearchable, noIndexer, noRoot, queryNeedsOneItem},
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
		Errors:  []*api.Code{workNotFound, manualSearchReplaced, api.Reconciling},
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
	case errors.Is(err, grabs.ErrReconciling):
		return nil, api.Reconciling
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
